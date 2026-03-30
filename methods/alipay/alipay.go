package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config holds the Alipay configuration.
type Config struct {
	AppID           string
	PrivateKey      string // RSA2 private key for signing
	AlipayPublicKey string // Alipay's public key for verification
	Gateway         string // e.g. "https://openapi.alipay.com/gateway.do"
	core.BaseConfig
}

// Payload is the Alipay-specific payment payload.
type Payload struct {
	TradeNo    string `json:"tradeNo"`
	OutTradeNo string `json:"outTradeNo"`
	BuyerID    string `json:"buyerId"`
}

// AlipayMethod implements core.Method for Alipay payments.
type AlipayMethod struct {
	config Config
}

// New creates a new Alipay payment method. Returns an error if the config is invalid.
func New(cfg Config) (*AlipayMethod, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("alipay: AppID is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("alipay: PrivateKey is required")
	}
	if cfg.AlipayPublicKey == "" {
		return nil, fmt.Errorf("alipay: AlipayPublicKey is required")
	}
	if cfg.Gateway == "" {
		return nil, fmt.Errorf("alipay: Gateway is required")
	}
	return &AlipayMethod{config: cfg}, nil
}

func (m *AlipayMethod) Name() string { return "alipay" }

func (m *AlipayMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge}
}

func (m *AlipayMethod) SupportedCurrencies() []core.Currency {
	return []core.Currency{core.CNY}
}

func (m *AlipayMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if err := core.ValidateBuildOption("alipay", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "alipay",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "Alipay payment",
	}, nil
}

func (m *AlipayMethod) CreatePayload(_ context.Context, option core.PaymentOption) (json.RawMessage, error) {
	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	now := time.Now()
	p := Payload{
		TradeNo:    fmt.Sprintf("alipay_%d", now.UnixNano()),
		OutTradeNo: fmt.Sprintf("acp_%d", now.UnixNano()),
		BuyerID:    "2088000000000000",
	}
	return json.Marshal(p)
}

func (m *AlipayMethod) Verify(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.VerifyResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "alipay"); err != nil {
		return nil, err
	}
	if p.TradeNo == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing tradeNo"}, nil
	}
	if p.OutTradeNo == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing outTradeNo"}, nil
	}
	if p.BuyerID == "" {
		return &core.VerifyResponse{Valid: false, Reason: "missing buyerId"}, nil
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	return &core.VerifyResponse{Valid: true, Payer: p.BuyerID}, nil
}

func (m *AlipayMethod) Settle(_ context.Context, payment core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	var p Payload
	if err := core.UnmarshalMethodPayload(payment.Payload, &p, "alipay"); err != nil {
		return nil, err
	}
	if p.TradeNo == "" {
		return nil, core.NewPaymentError(core.ErrInvalidPayload, "missing tradeNo")
	}

	// TODO: call m.config.BaseConfig.GetHTTPClient().Do(req)

	txnID := core.GenerateTxnID("provider")

	receipt := map[string]string{
		"tradeNo":    p.TradeNo,
		"outTradeNo": p.OutTradeNo,
		"buyerId":    p.BuyerID,
		"appId":      m.config.AppID,
	}

	return core.BuildSettleResponse("alipay", txnID, receipt)
}
