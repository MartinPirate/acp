// Package alipay implements Alipay payments.
package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Alipay configuration.
type Config struct {
	AppID           string
	PrivateKey      string // RSA2 private key for signing
	AlipayPublicKey string // Alipay's public key for verification
	Gateway         string // e.g. "https://openapi.alipay.com/gateway.do"
	// HTTPClient is the HTTP client used for Alipay API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Payload is the Alipay-specific payment payload.
type Payload struct {
	TradeNo    string `json:"tradeNo"`
	OutTradeNo string `json:"outTradeNo"`
	BuyerID    string `json:"buyerId"`
}

// AlipayMethod implements core.Method for Alipay payments.
type AlipayMethod struct {
	config Config
}

// New creates a new Alipay payment method. Returns an error if the config is invalid.
func New(cfg Config) (*AlipayMethod, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("alipay: AppID is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("alipay: PrivateKey is required")
	}
	if cfg.AlipayPublicKey == "" {
		return nil, fmt.Errorf("alipay: AlipayPublicKey is required")
	}
	if cfg.Gateway == "" {
		return nil, fmt.Errorf("alipay: Gateway is required")
	}
	return &AlipayMethod{config: cfg}, nil
}

func (m *AlipayMethod) Name() string { return "alipay" }

func (m *AlipayMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *AlipayMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.CNY}
}

func (m *AlipayMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("alipay does not support intent %q", intent))
	}
	if price.Currency != core.CNY {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("alipay only supports CNY, got %q", price.Currency))
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "alipay",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "Alipay payment",
	}, nil
}

func (m *AlipayMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Call Alipay gateway to create a trade and return trade number.
	_ = m.config.httpClient()

	now := time.Now()
	p := Payload{
		TradeNo:    fmt.Sprintf("alipay_%d", now.UnixNano()),
		OutTradeNo: fmt.Sprintf("acp_%d", now.UnixNano()),
		BuyerID:    "2088000000000000",
	}
	return json.Marshal(p)
}

func (m *AlipayMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid alipay payload: "+err.Error())
	}
	if p.TradeNo == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing tradeNo"}, nil
	}
	if p.OutTradeNo == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing outTradeNo"}, nil
	}
	if p.BuyerID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing buyerId"}, nil
	}

	// TODO: Call Alipay gateway to query trade status and verify signature.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: p.BuyerID}, nil
}

func (m *AlipayMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid alipay payload: "+err.Error())
	}
	if p.TradeNo == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing tradeNo")
	}

	// TODO: Call Alipay gateway to settle/close the trade.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("provider_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"tradeNo":    p.TradeNo,
		"outTradeNo": p.OutTradeNo,
		"buyerId":    p.BuyerID,
		"appId":      m.config.AppID,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "alipay",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
