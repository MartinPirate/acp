package upi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Razorpay UPI configuration.
type Config struct {
	APIKey      string
	APISecret   string
	MerchantVPA string // e.g. "merchant@upi"
	core.BaseConfig
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
	if err := core.ValidateBuildOption("upi", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
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
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	p := Payload{
		VPA:              m.config.MerchantVPA,
		TransactionRef:   fmt.Sprintf("upi_ref_%d", time.Now().UnixNano()),
		UPITransactionID: fmt.Sprintf("upi_txn_%d", time.Now().UnixNano()),
	}
	return json.Marshal(p)
}

func (m *UPIMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "upi"); err != nil {
		return nil, err
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

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: p.VPA}, nil
}

func (m *UPIMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "upi"); err != nil {
		return nil, err
	}
	if p.UPITransactionID == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing upiTransactionId")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("razorpay")

	receipt := map[string]string{
		"upiTransactionId": p.UPITransactionID,
		"transactionRef":   p.TransactionRef,
		"vpa":              p.VPA,
	}

	return core.BuildSettleResponse("upi", txnID, receipt)
}
