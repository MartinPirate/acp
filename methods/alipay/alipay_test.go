package alipay

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *AlipayMethod {
	t.Helper()
	m, err := New(Config{
		AppID:           "2021000000000000",
		PrivateKey:      "test_private_key",
		AlipayPublicKey: "test_alipay_public_key",
		Gateway:         "https://openapi.alipay.com/gateway.do",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return m
}

func TestNew_Validation(t *testing.T) {
	base := Config{
		AppID: "app", PrivateKey: "pk",
		AlipayPublicKey: "apk", Gateway: "https://gw.com",
	}
	tests := []struct {
		name    string
		mutate  func(Config) Config
		wantErr bool
	}{
		{"valid", func(c Config) Config { return c }, false},
		{"missing appid", func(c Config) Config { c.AppID = ""; return c }, true},
		{"missing private key", func(c Config) Config { c.PrivateKey = ""; return c }, true},
		{"missing alipay public key", func(c Config) Config { c.AlipayPublicKey = ""; return c }, true},
		{"missing gateway", func(c Config) Config { c.Gateway = ""; return c }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.mutate(base))
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildOption_CurrencyRestriction(t *testing.T) {
	m := newTestMethod(t)

	_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD")
	}

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.CNY})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "alipay" {
		t.Errorf("Method = %q, want %q", opt.Method, "alipay")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		TradeNo:    "alipay_123",
		OutTradeNo: "acp_123",
		BuyerID:    "2088000000000000",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "alipay", Currency: core.CNY, Amount: "100"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
	}
	if resp.Payer != "2088000000000000" {
		t.Errorf("Payer = %q, want buyer ID", resp.Payer)
	}

	// Missing tradeNo
	badPayload, _ := json.Marshal(Payload{OutTradeNo: "acp_123", BuyerID: "2088000000000000"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing tradeNo")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		TradeNo:    "alipay_123",
		OutTradeNo: "acp_123",
		BuyerID:    "2088000000000000",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "alipay", Currency: core.CNY, Amount: "100"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false")
	}
	if resp.Method != "alipay" {
		t.Errorf("Method = %q, want %q", resp.Method, "alipay")
	}
}
