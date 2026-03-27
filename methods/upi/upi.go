// Package upi implements UPI payments via Razorpay.
package upi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Razorpay UPI configuration.
type Config struct {
	APIKey      string
	APISecret   string
	MerchantVPA string // e.g. "merchant@upi"
	// HTTPClient is the HTTP client used for Razorpay API calls.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Payload is the UPI-specific payment payload.
type Payload struct {
	VPA              string `json:"vpa"`
	TransactionRef   string `json:"transactionRef"`
	UPITransactionID string `json:"upiTransactionId"`
}

// UPIMethod implements core.Method for UPI payments via Razorpay.
type UPIMethod struct {
	config Config
}

// New creates a new UPI payment method. Returns an error if the config is invalid.
func New(cfg Config) (*UPIMethod, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("upi: APIKey is required")
	}
	if cfg.APISecret == "" {
		return nil, fmt.Errorf("upi: APISecret is required")
	}
	if cfg.MerchantVPA == "" {
		return nil, fmt.Errorf("upi: MerchantVPA is required")
	}
	if !strings.Contains(cfg.MerchantVPA, "@") {
		return nil, fmt.Errorf("upi: MerchantVPA must be a valid VPA (e.g. merchant@upi)")
	}
	return &UPIMethod{config: cfg}, nil
}

func (m *UPIMethod) Name() string { return "upi" }

func (m *UPIMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *UPIMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.INR}
}

func (m *UPIMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if !core.SupportsIntent(m, intent) {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("upi does not support intent %q", intent))
	}
	if price.Currency != core.INR {
		return core.PaymentOption{}, core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("upi only supports INR, got %q", price.Currency))
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "upi",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "UPI payment via Razorpay",
	}, nil
}

func (m *UPIMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: Call Razorpay API to create a UPI payment request.
	_ = m.config.httpClient()

	p := Payload{
		VPA:              m.config.MerchantVPA,
		TransactionRef:   fmt.Sprintf("upi_ref_%d", time.Now().UnixNano()),
		UPITransactionID: fmt.Sprintf("upi_txn_%d", time.Now().UnixNano()),
	}
	return json.Marshal(p)
}

func (m *UPIMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid upi payload: "+err.Error())
	}
	if p.VPA == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing vpa"}, nil
	}
	if !strings.Contains(p.VPA, "@") {
		return &core.VerifyResponse{Valid: false, Reason: "invalid vpa format"}, nil
	}
	if p.TransactionRef == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing transactionRef"}, nil
	}
	if p.UPITransactionID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing upiTransactionId"}, nil
	}

	// TODO: Call Razorpay API to verify the UPI transaction status.
	_ = m.config.httpClient()

	return &core.VerifyResponse{Valid: true, Payer: p.VPA}, nil
}

func (m *UPIMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := json.Unmarshal(payment.Payload, &p); err != nil {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "invalid upi payload: "+err.Error())
	}
	if p.UPITransactionID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing upiTransactionId")
	}

	// TODO: Call Razorpay API to capture/settle the UPI payment.
	_ = m.config.httpClient()

	txnID := fmt.Sprintf("razorpay_txn_%d", time.Now().UnixNano())
	now := time.Now()

	receipt, _ := json.Marshal(map[string]string{
		"upiTransactionId": p.UPITransactionID,
		"transactionRef":   p.TransactionRef,
		"vpa":              p.VPA,
	})

	return &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "upi",
		Transaction: txnID,
		SettledAt:   now.Format(time.RFC3339),
		Receipt:     receipt,
	}, nil
}
