package card

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Stripe card payment configuration.
type Config struct {
	APIKey        string
	WebhookSecret string
	core.BaseConfig
}

// Payload is the card-specific payment payload.
type Payload struct {
	Token           string `json:"token"`
	PaymentIntentID string `json:"paymentIntentId"`
}

// CardMethod implements core.Method for Stripe card payments.
type CardMethod struct {
	config Config
}

// New creates a new card payment method. Returns an error if the config is invalid.
func New(cfg Config) (*CardMethod, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("card: APIKey is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("card: WebhookSecret is required")
	}
	return &CardMethod{config: cfg}, nil
}

func (m *CardMethod) Name() string { return "card" }

func (m *CardMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge, core.IntentAuthorize}
}

func (m *CardMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{
		core.USD, core.EUR, core.GBP, core.JPY, core.INR,
		core.BRL, core.MXN, core.SEK, core.NOK, core.SAR,
		core.PHP, core.THB, core.IDR, core.ZAR, core.CNY,
	}
}

func (m *CardMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if err := core.ValidateBuildOption("card", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "card",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "Card payment via Stripe",
	}, nil
}

func (m *CardMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	p := Payload{
		Token:           fmt.Sprintf("tok_%d", time.Now().UnixNano()),
		PaymentIntentID: fmt.Sprintf("pi_%d", time.Now().UnixNano()),
	}
	return json.Marshal(p)
}

func (m *CardMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "card"); err != nil {
		return nil, err
	}
	if p.Token == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing token"}, nil
	}
	if p.PaymentIntentID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing paymentIntentId"}, nil
	}
	if !strings.HasPrefix(p.PaymentIntentID, "pi_") {
		return &core.VerifyResponse{Valid: false, Reason: "invalid paymentIntentId format"}, nil
	}
	if !strings.HasPrefix(p.Token, "tok_") {
		return &core.VerifyResponse{Valid: false, Reason: "invalid token format"}, nil
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: "card-holder"}, nil
}

func (m *CardMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "card"); err != nil {
		return nil, err
	}
	if p.PaymentIntentID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing paymentIntentId")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("stripe")
	now := time.Now()

	receipt := map[string]string{
		"paymentIntentId": p.PaymentIntentID,
		"stripeChargeId":  fmt.Sprintf("ch_%d", now.UnixNano()),
	}

	return core.BuildSettleResponse("card", txnID, receipt)
}
