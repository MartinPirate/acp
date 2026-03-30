package mpesa

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *MpesaMethod {
	t.Helper()
	m, err := New(Config{
		ConsumerKey:    "test_key",
		ConsumerSecret: "test_secret",
		ShortCode:      "174379",
		PassKey:        "test_passkey",
		Environment:    "sandbox",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return m
}

func TestNew_Validation(t *testing.T) {
	base := Config{
		ConsumerKey: "k", ConsumerSecret: "s",
		ShortCode: "174379", PassKey: "pk", Environment: "sandbox",
	}
	tests := []struct {
		name    string
		mutate  func(Config) Config
		wantErr bool
		errMsg  string
	}{
		{"valid", func(c Config) Config { return c }, false, ""},
		{"valid production", func(c Config) Config { c.Environment = "production"; return c }, false, ""},
		{"missing key", func(c Config) Config { c.ConsumerKey = ""; return c }, true, "ConsumerKey"},
		{"missing secret", func(c Config) Config { c.ConsumerSecret = ""; return c }, true, "ConsumerSecret"},
		{"missing shortcode", func(c Config) Config { c.ShortCode = ""; return c }, true, "ShortCode"},
		{"missing passkey", func(c Config) Config { c.PassKey = ""; return c }, true, "PassKey"},
		{"bad env", func(c Config) Config { c.Environment = "staging"; return c }, true, "Environment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.mutate(base))
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
	if m.Name() != "mpesa" {
		t.Errorf("Name() = %q, want %q", m.Name(), "mpesa")
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
	if currencies[0] != core.KES {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.KES)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.KES})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "mpesa" {
		t.Errorf("Method = %q, want %q", opt.Method, "mpesa")
	}
	if opt.Amount != "100" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "100")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentSubscribe, core.Price{Amount: "100", Currency: core.KES})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
	}

	// Unsupported currency
	_, err = m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.USD})
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
	option := core.PaymentOption{Method: "mpesa", Currency: core.KES, Amount: "100"}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}
	if p.PhoneNumber == "" {
		t.Error("CreatePayload() PhoneNumber is empty")
	}
	if p.AccountRef == "" {
		t.Error("CreatePayload() AccountRef is empty")
	}
	if p.CheckoutRequestID == "" {
		t.Error("CreatePayload() CheckoutRequestID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "mpesa", Currency: core.KES, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			PhoneNumber:       "254712345678",
			AccountRef:        "ACP_123",
			TransactionID:     "txn_123",
			CheckoutRequestID: "ws_CO_123",
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "254712345678" {
			t.Errorf("Payer = %q, want %q", resp.Payer, "254712345678")
		}
	})

	t.Run("missing phoneNumber", func(t *testing.T) {
		p, _ := json.Marshal(Payload{AccountRef: "ACP_123", CheckoutRequestID: "ws_CO_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing phoneNumber")
		}
	})

	t.Run("invalid phone format", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			PhoneNumber:       "0712345678",
			AccountRef:        "ACP_123",
			CheckoutRequestID: "ws_CO_123",
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for bad phone format")
		}
	})

	t.Run("missing checkoutRequestId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PhoneNumber: "254712345678", AccountRef: "ACP_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing checkoutRequestId")
		}
	})

	t.Run("missing accountRef", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PhoneNumber: "254712345678", CheckoutRequestID: "ws_CO_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing accountRef")
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
	option := core.PaymentOption{Method: "mpesa", Currency: core.KES, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			PhoneNumber:       "254712345678",
			AccountRef:        "ACP_123",
			TransactionID:     "txn_123",
			CheckoutRequestID: "ws_CO_123",
		})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "mpesa" {
			t.Errorf("Method = %q, want %q", resp.Method, "mpesa")
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
		if receipt["checkoutRequestId"] != "ws_CO_123" {
			t.Errorf("receipt checkoutRequestId = %q, want %q", receipt["checkoutRequestId"], "ws_CO_123")
		}
		if receipt["shortCode"] != "174379" {
			t.Errorf("receipt shortCode = %q, want %q", receipt["shortCode"], "174379")
		}
	})

	t.Run("missing checkoutRequestId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PhoneNumber: "254712345678", AccountRef: "ACP_123"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing checkoutRequestId")
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
	option := core.PaymentOption{Intent: core.IntentCharge, Method: "mpesa", Currency: core.KES, Amount: "100"}

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
