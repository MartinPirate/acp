// Package card implements card payments via Stripe's PaymentIntent API.
package card

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Stripe card payment configuration.
type Config struct {
	APIKey        string
	WebhookSecret string
	// HTTPClient is the HTTP client used for Stripe API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
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
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("card does not support intent %q", intent))
	}
	if !core.SupportsCurrency(m, price.Currency) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("card does not support currency %q", price.Currency))
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
	// TODO: Call Stripe API to create a PaymentIntent and return the client secret.
	_ = m.config.httpClient()

	p := Payload{
		Token:           fmt.Sprintf("tok_%d", time.Now().UnixNano()),
		PaymentIntentID: fmt.Sprintf("pi_%d", time.Now().UnixNano()),
	}
	return json.Marshal(p)
}

func (m *CardMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid card payload: "+err.Error())
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

	// TODO: Call Stripe API to retrieve the PaymentIntent and verify its status.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: "card-holder"}, nil
}

func (m *CardMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid card payload: "+err.Error())
	}
	if p.PaymentIntentID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing paymentIntentId")
	}

	// TODO: Call Stripe API to confirm/capture the PaymentIntent.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("stripe_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"paymentIntentId": p.PaymentIntentID,
		"stripeChargeId":  fmt.Sprintf("ch_%d", now.UnixNano()),
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "card",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
