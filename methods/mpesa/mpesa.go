// Package mpesa implements M-Pesa payments via the Daraja API.
package mpesa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the M-Pesa Daraja configuration.
type Config struct {
	ConsumerKey    string
	ConsumerSecret string
	ShortCode      string
	PassKey        string
	Environment    string // "sandbox" or "production"
	// HTTPClient is the HTTP client used for Daraja API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// baseURL returns the Daraja API base URL for the configured environment.
func (c *Config) baseURL() string {
	if c.Environment == "production" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

// Payload is the M-Pesa-specific payment payload.
type Payload struct {
	PhoneNumber       string `json:"phoneNumber"`
	AccountRef        string `json:"accountRef"`
	TransactionID     string `json:"transactionId"`
	CheckoutRequestID string `json:"checkoutRequestId"`
}

var phoneRegexp = regexp.MustCompile(`^254\d{9}$`)

// MpesaMethod implements core.Method for M-Pesa payments.
type MpesaMethod struct {
	config Config
}

// New creates a new M-Pesa payment method. Returns an error if the config is invalid.
func New(cfg Config) (*MpesaMethod, error) {
	if cfg.ConsumerKey == "" {
		return nil, fmt.Errorf("mpesa: ConsumerKey is required")
	}
	if cfg.ConsumerSecret == "" {
		return nil, fmt.Errorf("mpesa: ConsumerSecret is required")
	}
	if cfg.ShortCode == "" {
		return nil, fmt.Errorf("mpesa: ShortCode is required")
	}
	if cfg.PassKey == "" {
		return nil, fmt.Errorf("mpesa: PassKey is required")
	}
	if cfg.Environment != "sandbox" && cfg.Environment != "production" {
		return nil, fmt.Errorf("mpesa: Environment must be \"sandbox\" or \"production\"")
	}
	return &MpesaMethod{config: cfg}, nil
}

func (m *MpesaMethod) Name() string { return "mpesa" }

func (m *MpesaMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *MpesaMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.KES}
}

func (m *MpesaMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("mpesa does not support intent %q", intent))
	}
	if price.Currency != core.KES {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("mpesa only supports KES, got %q", price.Currency))
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "mpesa",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "M-Pesa payment via Daraja",
	}, nil
}

func (m *MpesaMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Call Daraja API to initiate an STK Push request.
	_ = m.config.httpClient()
	_ = m.config.baseURL()

	now := time.Now()
	p := Payload{
		PhoneNumber:       "254700000000",
		AccountRef:        fmt.Sprintf("ACP_%d", now.UnixNano()),
		TransactionID:     fmt.Sprintf("mpesa_txn_%d", now.UnixNano()),
		CheckoutRequestID: fmt.Sprintf("ws_CO_%d", now.UnixNano()),
	}
	return json.Marshal(p)
}

func (m *MpesaMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid mpesa payload: "+err.Error())
	}
	if p.PhoneNumber == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing phoneNumber"}, nil
	}
	if !phoneRegexp.MatchString(p.PhoneNumber) {
		return &core.VerifyResponse{Valid: false, Reason: "invalid phoneNumber format (expected 254XXXXXXXXX)"}, nil
	}
	if p.CheckoutRequestID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing checkoutRequestId"}, nil
	}
	if p.AccountRef == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing accountRef"}, nil
	}

	// TODO: Call Daraja API to query STK Push status.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: p.PhoneNumber}, nil
}

func (m *MpesaMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid mpesa payload: "+err.Error())
	}
	if p.CheckoutRequestID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing checkoutRequestId")
	}

	// TODO: Call Daraja API to confirm settlement.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("provider_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"checkoutRequestId": p.CheckoutRequestID,
		"phoneNumber":       p.PhoneNumber,
		"accountRef":        p.AccountRef,
		"shortCode":         m.config.ShortCode,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "mpesa",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
