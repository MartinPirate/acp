// Package sepa implements SEPA Instant credit transfer payments.
package sepa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the SEPA payment configuration.
type Config struct {
	APIKey   string
	IBAN     string
	BIC      string
	Provider string // e.g. "stripe", "adyen"
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

// Payload is the SEPA-specific payment payload.
type Payload struct {
	IBAN       string `json:"iban"`
	BIC        string `json:"bic"`
	Reference  string `json:"reference"`
	EndToEndID string `json:"endToEndId"`
}

var ibanRegexp = regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]{4,30}$`)

// SepaMethod implements core.Method for SEPA Instant credit transfers.
type SepaMethod struct {
	config Config
}

// New creates a new SEPA payment method. Returns an error if the config is invalid.
func New(cfg Config) (*SepaMethod, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("sepa: APIKey is required")
	}
	if cfg.IBAN == "" {
		return nil, fmt.Errorf("sepa: IBAN is required")
	}
	if !ibanRegexp.MatchString(cfg.IBAN) {
		return nil, fmt.Errorf("sepa: IBAN format is invalid")
	}
	if cfg.BIC == "" {
		return nil, fmt.Errorf("sepa: BIC is required")
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("sepa: Provider is required")
	}
	return &SepaMethod{config: cfg}, nil
}

func (m *SepaMethod) Name() string { return "sepa" }

func (m *SepaMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *SepaMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.EUR}
}

func (m *SepaMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("sepa does not support intent %q", intent))
	}
	if price.Currency != core.EUR {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("sepa only supports EUR, got %q", price.Currency))
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "sepa",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "SEPA Instant credit transfer",
	}, nil
}

func (m *SepaMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Call provider API to initiate a SEPA credit transfer.
	_ = m.config.httpClient()

	now := time.Now()
	p := Payload{
		IBAN:       m.config.IBAN,
		BIC:        m.config.BIC,
		Reference:  fmt.Sprintf("SEPA-%d", now.UnixNano()),
		EndToEndID: fmt.Sprintf("E2E-%d", now.UnixNano()),
	}
	return json.Marshal(p)
}

func (m *SepaMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid sepa payload: "+err.Error())
	}
	if p.IBAN == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing iban"}, nil
	}
	if !ibanRegexp.MatchString(p.IBAN) {
		return &core.VerifyResponse{Valid: false, Reason: "invalid iban format"}, nil
	}
	if p.BIC == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing bic"}, nil
	}
	if p.Reference == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing reference"}, nil
	}
	if p.EndToEndID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing endToEndId"}, nil
	}

	// TODO: Call provider API to verify SEPA transfer status.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: p.IBAN}, nil
}

func (m *SepaMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid sepa payload: "+err.Error())
	}
	if p.EndToEndID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing endToEndId")
	}

	// TODO: Call provider API to confirm SEPA settlement.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("provider_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"endToEndId": p.EndToEndID,
		"reference":  p.Reference,
		"iban":       p.IBAN,
		"bic":        p.BIC,
		"provider":   m.config.Provider,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "sepa",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
