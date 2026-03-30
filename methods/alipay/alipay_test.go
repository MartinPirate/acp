package alipay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestName(t *testing.T) {
	m := newTestMethod(t)
	if m.Name() != "alipay" {
		t.Errorf("Name() = %q, want %q", m.Name(), "alipay")
	}
}

func TestSupportedIntents(t *testing.T) {
	m := newTestMethod(t)
	intents := m.SupportedIntents()
	if len(intents) != 1 {
		t.Fatalf("SupportedIntents() returned %d, want 1", len(intents))
	}
	if intents[0] != core.IntentCharge {
		t.Errorf("SupportedIntents()[0] = %q, want %q", intents[0], core.IntentCharge)
	}
}

func TestSupportedCurrencies(t *testing.T) {
	m := newTestMethod(t)
	currencies := m.SupportedCurrencies()
	if len(currencies) != 1 {
		t.Fatalf("SupportedCurrencies() returned %d, want 1", len(currencies))
	}
	if currencies[0] != core.CNY {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.CNY)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.CNY})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "alipay" {
		t.Errorf("Method = %q, want %q", opt.Method, "alipay")
	}
	if opt.Amount != "100" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "100")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "100", Currency: core.CNY})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
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

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "alipay", Currency: core.CNY, Amount: "100"}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}
	if raw == nil {
		t.Fatal("CreatePayload() returned nil")
	}

	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}
	if p.TradeNo == "" {
		t.Error("CreatePayload() TradeNo is empty")
	}
	if p.OutTradeNo == "" {
		t.Error("CreatePayload() OutTradeNo is empty")
	}
	if p.BuyerID == "" {
		t.Error("CreatePayload() BuyerID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "alipay", Currency: core.CNY, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{TradeNo: "alipay_123", OutTradeNo: "acp_123", BuyerID: "2088000000000000"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "2088000000000000" {
			t.Errorf("Payer = %q, want buyer ID", resp.Payer)
		}
	})

	t.Run("missing tradeNo", func(t *testing.T) {
		p, _ := json.Marshal(Payload{OutTradeNo: "acp_123", BuyerID: "2088000000000000"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing tradeNo")
		}
	})

	t.Run("missing outTradeNo", func(t *testing.T) {
		p, _ := json.Marshal(Payload{TradeNo: "alipay_123", BuyerID: "2088000000000000"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing outTradeNo")
		}
	})

	t.Run("missing buyerId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{TradeNo: "alipay_123", OutTradeNo: "acp_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing buyerId")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := m.Verify(ctx, core.PaymentPayload{Payload: json.RawMessage(`{bad}`)}, option)
		if err == nil {
			t.Error("Verify() expected error for malformed JSON")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: json.RawMessage(`{}`)}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for empty payload")
		}
	})
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "alipay", Currency: core.CNY, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{TradeNo: "alipay_123", OutTradeNo: "acp_123", BuyerID: "2088000000000000"})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "alipay" {
			t.Errorf("Method = %q, want %q", resp.Method, "alipay")
		}
		if resp.Transaction == "" {
			t.Error("Settle() transaction is empty")
		}
		if resp.SettledAt == "" {
			t.Error("Settle() settledAt is empty")
		}
		if _, err := time.Parse(time.RFC3339, resp.SettledAt); err != nil {
			t.Errorf("Settle() settledAt not valid RFC3339: %v", err)
		}
		if resp.Receipt == nil {
			t.Error("Settle() receipt is nil")
		}

		var receipt map[string]string
		if err := json.Unmarshal(resp.Receipt, &receipt); err != nil {
			t.Fatalf("receipt is not valid JSON: %v", err)
		}
		if receipt["tradeNo"] != "alipay_123" {
			t.Errorf("receipt tradeNo = %q, want %q", receipt["tradeNo"], "alipay_123")
		}
		if receipt["appId"] != "2021000000000000" {
			t.Errorf("receipt appId = %q, want %q", receipt["appId"], "2021000000000000")
		}
	})

	t.Run("missing tradeNo", func(t *testing.T) {
		p, _ := json.Marshal(Payload{OutTradeNo: "acp_123", BuyerID: "2088000000000000"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing tradeNo")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: json.RawMessage(`{bad}`)}, option)
		if err == nil {
			t.Error("Settle() expected error for malformed JSON")
		}
	})
}

func TestFullRoundTrip(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	option, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.CNY})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	payment := core.PaymentPayload{Payload: raw}

	vr, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !vr.Valid {
		t.Fatalf("Verify() valid = false; reason: %s", vr.Reason)
	}

	sr, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !sr.Success {
		t.Fatal("Settle() success = false")
	}
	if sr.Transaction == "" {
		t.Error("Settle() transaction is empty")
	}
}
