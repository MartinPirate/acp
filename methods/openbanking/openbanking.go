package openbanking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Open Banking payment configuration.
type Config struct {
	APIKey      string
	Provider    string // e.g. "truelayer", "plaid"
	RedirectURL string
	core.BaseConfig
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
	if err := core.ValidateBuildOption("openbanking", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
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
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

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
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "openbanking"); err != nil {
		return nil, err
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

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: "ob-payer"}, nil
}

func (m *OpenBankingMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "openbanking"); err != nil {
		return nil, err
	}
	if p.PaymentID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing paymentId")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("provider")

	receipt := map[string]string{
		"consentId": p.ConsentID,
		"paymentId": p.PaymentID,
		"provider":  p.Provider,
	}

	return core.BuildSettleResponse("openbanking", txnID, receipt)
}
