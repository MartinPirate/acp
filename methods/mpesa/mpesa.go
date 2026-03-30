package mpesa

import (
	"context"
	"encoding/json"
	"fmt"
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
	core.BaseConfig
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
	if err := core.ValidateBuildOption("mpesa", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
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
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)
	_ = m.config.baseURL()

	now := time.Now()
	p := Payload{
		PhoneNumber:       "254700000000",
		AccountRef:        fmt.Sprintf("ACP_%d", now.UnixNano()),
		TransactionID:     core.GenerateTxnID("mpesa"),
		CheckoutRequestID: fmt.Sprintf("ws_CO_%d", now.UnixNano()),
	}
	return json.Marshal(p)
}

func (m *MpesaMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "mpesa"); err != nil {
		return nil, err
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

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: p.PhoneNumber}, nil
}

func (m *MpesaMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "mpesa"); err != nil {
		return nil, err
	}
	if p.CheckoutRequestID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing checkoutRequestId")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("provider")

	receipt := map[string]string{
		"checkoutRequestId": p.CheckoutRequestID,
		"phoneNumber":       p.PhoneNumber,
		"accountRef":        p.AccountRef,
		"shortCode":         m.config.ShortCode,
	}

	return core.BuildSettleResponse("mpesa", txnID, receipt)
}
