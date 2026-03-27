// Package openbanking implements Open Banking (PSD2) payments.
package openbanking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Open Banking payment configuration.
type Config struct {
	APIKey      string
	Provider    string // e.g. "truelayer", "plaid"
	RedirectURL string
	// HTTPClient is the HTTP client used for provider API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Payload is the Open Banking-specific payment payload.
type Payload struct {
	ConsentID string `json:"consentId"`
	PaymentID string `json:"paymentId"`
	Provider  string `json:"provider"`
}

// OpenBankingMethod implements core.Method for Open Banking payments.
type OpenBankingMethod struct {
	config Config
}

// New creates a new Open Banking payment method. Returns an error if the config is invalid.
func New(cfg Config) (*OpenBankingMethod, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openbanking: APIKey is required")
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("openbanking: Provider is required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("openbanking: RedirectURL is required")
	}
	return &OpenBankingMethod{config: cfg}, nil
}

func (m *OpenBankingMethod) Name() string { return "openbanking" }

func (m *OpenBankingMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge, core.IntentMandate}
}

func (m *OpenBankingMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.EUR, core.GBP}
}

func (m *OpenBankingMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("openbanking does not support intent %q", intent))
	}
	if !core.SupportsCurrency(m, price.Currency) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("openbanking does not support currency %q", price.Currency))
	}

	extra, _ := json.Marshal(map[string]string{
		"provider":    m.config.Provider,
		"redirectUrl": m.config.RedirectURL,
	})

	return core.PaymentOption{
		Intent:      intent,
		Method:      "openbanking",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: fmt.Sprintf("Open Banking payment via %s", m.config.Provider),
		Extra:       extra,
	}, nil
}

func (m *OpenBankingMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Call provider API to create a payment initiation request.
	_ = m.config.httpClient()

	now := time.Now()
	p := Payload{
		ConsentID: fmt.Sprintf("consent_%d", now.UnixNano()),
		PaymentID: fmt.Sprintf("pay_%d", now.UnixNano()),
		Provider:  m.config.Provider,
	}
	return json.Marshal(p)
}

func (m *OpenBankingMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid openbanking payload: "+err.Error())
	}
	if p.ConsentID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing consentId"}, nil
	}
	if p.PaymentID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing paymentId"}, nil
	}
	if p.Provider == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing provider"}, nil
	}

	// TODO: Call provider API to verify payment initiation status and consent.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: "ob-payer"}, nil
}

func (m *OpenBankingMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid openbanking payload: "+err.Error())
	}
	if p.PaymentID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing paymentId")
	}

	// TODO: Call provider API to execute the payment.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("provider_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"consentId": p.ConsentID,
		"paymentId": p.PaymentID,
		"provider":  p.Provider,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "openbanking",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
