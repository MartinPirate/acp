// Package x402 implements the x402 bridge for USDC payments on EVM chains.
package x402

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the x402 bridge configuration.
type Config struct {
	FacilitatorURL string
	Network        string // e.g. "eip155:8453" for Base
	PrivateKey     string // client-side signing key (hex)
	// HTTPClient is the HTTP client used for facilitator API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Authorization represents an EIP-3009 style transfer authorization.
type Authorization struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	ValidAfter  int64  `json:"validAfter"`
	ValidBefore int64  `json:"validBefore"`
	Nonce       string `json:"nonce"`
}

// Payload is the x402-specific payment payload.
type Payload struct {
	Signature     string        `json:"signature"`
	Authorization Authorization `json:"authorization"`
}

// NetworkExtra is the extra field added to PaymentOption for network info.
type NetworkExtra struct {
	Network        string `json:"network"`
	FacilitatorURL string `json:"facilitatorUrl"`
}

// X402Method implements core.Method for x402 USDC bridge payments.
type X402Method struct {
	config Config
}

// New creates a new x402 payment method. Returns an error if the config is invalid.
func New(cfg Config) (*X402Method, error) {
	if cfg.FacilitatorURL == "" {
		return nil, fmt.Errorf("x402: FacilitatorURL is required")
	}
	if cfg.Network == "" {
		return nil, fmt.Errorf("x402: Network is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("x402: PrivateKey is required")
	}
	return &X402Method{config: cfg}, nil
}

func (m *X402Method) Name() string { return "x402" }

func (m *X402Method) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *X402Method) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.USDC}
}

func (m *X402Method) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("x402 does not support intent %q", intent))
	}
	if price.Currency != core.USDC {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("x402 only supports USDC, got %q", price.Currency))
	}

	extra, _ := json.Marshal(NetworkExtra{
		Network:        m.config.Network,
		FacilitatorURL: m.config.FacilitatorURL,
	})

	return core.PaymentOption{
		Intent:      intent,
		Method:      "x402",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: fmt.Sprintf("USDC payment via x402 on %s", m.config.Network),
		Extra:       extra,
	}, nil
}

func (m *X402Method) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Sign an EIP-3009 authorization using the configured private key.

	now := time.Now()
	nonce := fmt.Sprintf("0x%x", now.UnixNano())

	p := Payload{
		Signature: fmt.Sprintf("0xsig_%d", now.UnixNano()),
		Authorization: Authorization{
			From:        "0x0000000000000000000000000000000000000001",
			To:          "0x0000000000000000000000000000000000000002",
			Value:       option.Amount,
			ValidAfter:  now.Unix(),
			ValidBefore: now.Add(5 * time.Minute).Unix(),
			Nonce:       nonce,
		},
	}
	return json.Marshal(p)
}

func (m *X402Method) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid x402 payload: "+err.Error())
	}
	if p.Signature == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing signature"}, nil
	}
	if p.Authorization.From == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing authorization.from"}, nil
	}
	if p.Authorization.To == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing authorization.to"}, nil
	}
	if p.Authorization.Value == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing authorization.value"}, nil
	}
	if p.Authorization.ValidBefore <= time.Now().Unix() {
		return &core.VerifyResponse{Valid: false, Reason: "authorization expired"}, nil
	}

	// TODO: Forward to x402 facilitator for on-chain verification.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: p.Authorization.From}, nil
}

func (m *X402Method) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid x402 payload: "+err.Error())
	}
	if p.Signature == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing signature")
	}

	// TODO: Forward to x402 facilitator for on-chain settlement.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("x402_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"network":   m.config.Network,
		"txHash":    fmt.Sprintf("0xtx_%d", now.UnixNano()),
		"from":      p.Authorization.From,
		"to":        p.Authorization.To,
		"value":     p.Authorization.Value,
		"signature": p.Signature,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "x402",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
