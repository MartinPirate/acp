package pix

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the PIX payment configuration.
type Config struct {
	APIKey   string
	PixKey   string // CPF/CNPJ, email, phone, or random key
	Provider string // e.g. "stripe", "pagseguro"
	core.BaseConfig
}

// Payload is the PIX-specific payment payload.
type Payload struct {
	PixKey string `json:"pixKey"`
	E2eID  string `json:"e2eId"`
	TxID   string `json:"txId"`
}

// PixMethod implements core.Method for PIX payments.
type PixMethod struct {
	config Config
}

// New creates a new PIX payment method. Returns an error if the config is invalid.
func New(cfg Config) (*PixMethod, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("pix: APIKey is required")
	}
	if cfg.PixKey == "" {
		return nil, fmt.Errorf("pix: PixKey is required")
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("pix: Provider is required")
	}
	return &PixMethod{config: cfg}, nil
}

func (m *PixMethod) Name() string { return "pix" }

func (m *PixMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *PixMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.BRL}
}

func (m *PixMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if err := core.ValidateBuildOption("pix", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "pix",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: fmt.Sprintf("PIX payment via %s", m.config.Provider),
	}, nil
}

func (m *PixMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	now := time.Now()
	p := Payload{
		PixKey: m.config.PixKey,
		E2eID:  fmt.Sprintf("E%032d", now.UnixNano()),
		TxID:   fmt.Sprintf("pix_tx_%d", now.UnixNano()),
	}
	return json.Marshal(p)
}

func (m *PixMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "pix"); err != nil {
		return nil, err
	}
	if p.PixKey == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing pixKey"}, nil
	}
	if p.E2eID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing e2eId"}, nil
	}
	if p.TxID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing txId"}, nil
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: "pix-payer"}, nil
}

func (m *PixMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "pix"); err != nil {
		return nil, err
	}
	if p.TxID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing txId")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("provider")

	receipt := map[string]string{
		"e2eId":    p.E2eID,
		"txId":     p.TxID,
		"pixKey":   p.PixKey,
		"provider": m.config.Provider,
	}

	return core.BuildSettleResponse("pix", txnID, receipt)
}
