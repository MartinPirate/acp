package pix

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *PixMethod {
	t.Helper()
	m, err := New(Config{APIKey: "test_key", PixKey: "12345678901", Provider: "stripe"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return m
}

func TestNew_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{"valid", Config{APIKey: "k", PixKey: "pk", Provider: "stripe"}, false, ""},
		{"missing api key", Config{PixKey: "pk", Provider: "stripe"}, true, "APIKey"},
		{"missing pix key", Config{APIKey: "k", Provider: "stripe"}, true, "PixKey"},
		{"missing provider", Config{APIKey: "k", PixKey: "pk"}, true, "Provider"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestName(t *testing.T) {
	m := newTestMethod(t)
	if m.Name() != "pix" {
		t.Errorf("Name() = %q, want %q", m.Name(), "pix")
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
	if currencies[0] != core.BRL {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.BRL)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	// Happy path
	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "50.00", Currency: core.BRL})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "pix" {
		t.Errorf("Method = %q, want %q", opt.Method, "pix")
	}
	if opt.Amount != "50.00" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "50.00")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "50.00", Currency: core.BRL})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
	}

	// Unsupported currency
	_, err = m.BuildOption(core.IntentCharge, core.Price{Amount: "50.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD")
	}
	if !core.IsPaymentError(err, core.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got: %v", err)
	}
}

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "pix", Currency: core.BRL, Amount: "50.00"}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}
	if p.PixKey == "" {
		t.Error("CreatePayload() PixKey is empty")
	}
	if p.E2eID == "" {
		t.Error("CreatePayload() E2eID is empty")
	}
	if p.TxID == "" {
		t.Error("CreatePayload() TxID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "pix", Currency: core.BRL, Amount: "50.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123", TxID: "tx_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
	})

	t.Run("missing pixKey", func(t *testing.T) {
		p, _ := json.Marshal(Payload{E2eID: "E123", TxID: "tx_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing pixKey")
		}
	})

	t.Run("missing e2eId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PixKey: "12345678901", TxID: "tx_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing e2eId")
		}
	})

	t.Run("missing txId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing txId")
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
	option := core.PaymentOption{Method: "pix", Currency: core.BRL, Amount: "50.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123", TxID: "tx_123"})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "pix" {
			t.Errorf("Method = %q, want %q", resp.Method, "pix")
		}
		if resp.Transaction == "" {
			t.Error("Settle() transaction is empty")
		}
		if _, err := time.Parse(time.RFC3339, resp.SettledAt); err != nil {
			t.Errorf("Settle() settledAt not valid RFC3339: %v", err)
		}

		var receipt map[string]string
		if err := json.Unmarshal(resp.Receipt, &receipt); err != nil {
			t.Fatalf("receipt is not valid JSON: %v", err)
		}
		if receipt["e2eId"] != "E123" {
			t.Errorf("receipt e2eId = %q, want %q", receipt["e2eId"], "E123")
		}
		if receipt["provider"] != "stripe" {
			t.Errorf("receipt provider = %q, want %q", receipt["provider"], "stripe")
		}
	})

	t.Run("missing txId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing txId")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: json.RawMessage(`{bad}`)}, option)
		if err == nil {
			t.Error("Settle() expected error for malformed JSON")
		}
	})
}

func TestVerifySettleRoundTrip(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Intent: core.IntentCharge, Method: "pix", Currency: core.BRL, Amount: "50.00"}

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
}
