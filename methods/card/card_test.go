package card

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *CardMethod {
	t.Helper()
	m, err := New(Config{APIKey: "sk_test_123", WebhookSecret: "whsec_test_123"})
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
	}{
		{"valid", Config{APIKey: "sk_test", WebhookSecret: "whsec_test"}, false},
		{"missing api key", Config{WebhookSecret: "whsec_test"}, true},
		{"missing webhook secret", Config{APIKey: "sk_test"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestName(t *testing.T) {
	m := newTestMethod(t)
	if m.Name() != "card" {
		t.Errorf("Name() = %q, want %q", m.Name(), "card")
	}
}

func TestSupportedIntents(t *testing.T) {
	m := newTestMethod(t)
	intents := m.SupportedIntents()
	if len(intents) != 2 {
		t.Fatalf("SupportedIntents() returned %d, want 2", len(intents))
	}
	found := map[core.Intent]bool{}
	for _, i := range intents {
		found[i] = true
	}
	if !found[core.IntentCharge] {
		t.Error("SupportedIntents() missing IntentCharge")
	}
	if !found[core.IntentAuthorize] {
		t.Error("SupportedIntents() missing IntentAuthorize")
	}
}

func TestSupportedCurrencies(t *testing.T) {
	m := newTestMethod(t)
	currencies := m.SupportedCurrencies()
	if len(currencies) == 0 {
		t.Fatal("SupportedCurrencies() returned empty")
	}
	found := map[core.Currency]bool{}
	for _, c := range currencies {
		found[c] = true
	}
	for _, want := range []core.Currency{core.USD, core.EUR, core.GBP} {
		if !found[want] {
			t.Errorf("SupportedCurrencies() missing %s", want)
		}
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USD})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "card" {
		t.Errorf("Method = %q, want %q", opt.Method, "card")
	}
	if opt.Amount != "10.00" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "10.00")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "10.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
	}

	// Unsupported currency
	_, err = m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.KES})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported currency")
	}
	if !core.IsPaymentError(err, core.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got: %v", err)
	}
}

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "card", Currency: core.USD, Amount: "10.00"}

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
	if p.Token == "" {
		t.Error("CreatePayload() Token is empty")
	}
	if p.PaymentIntentID == "" {
		t.Error("CreatePayload() PaymentIntentID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{Token: "tok_123", PaymentIntentID: "pi_456"})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "card", Currency: core.USD, Amount: "10.00"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false, want true; reason: %s", resp.Reason)
	}

	// Missing token
	badPayload, _ := json.Marshal(Payload{PaymentIntentID: "pi_456"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing token")
	}

	// Invalid format
	badFormat, _ := json.Marshal(Payload{Token: "bad", PaymentIntentID: "pi_456"})
	payment.Payload = badFormat
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for bad token format")
	}

	// Malformed JSON
	payment.Payload = json.RawMessage(`{bad json}`)
	_, err = m.Verify(ctx, payment, option)
	if err == nil {
		t.Error("Verify() expected error for malformed JSON")
	}

	// Empty JSON object
	payment.Payload = json.RawMessage(`{}`)
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for empty payload")
	}

	// Missing paymentIntentId
	missingPI, _ := json.Marshal(Payload{Token: "tok_123"})
	payment.Payload = missingPI
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing paymentIntentId")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "card", Currency: core.USD, Amount: "10.00"}

	t.Run("valid payload", func(t *testing.T) {
		validPayload, _ := json.Marshal(Payload{Token: "tok_123", PaymentIntentID: "pi_456"})
		payment := core.PaymentPayload{Payload: validPayload}

		resp, err := m.Settle(ctx, payment, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false, want true")
		}
		if resp.Method != "card" {
			t.Errorf("Settle() method = %q, want %q", resp.Method, "card")
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
		if receipt["paymentIntentId"] != "pi_456" {
			t.Errorf("receipt paymentIntentId = %q, want %q", receipt["paymentIntentId"], "pi_456")
		}
	})

	t.Run("missing paymentIntentId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{Token: "tok_123"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing paymentIntentId")
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

	// BuildOption
	option, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USD})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}

	// CreatePayload
	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	payment := core.PaymentPayload{Payload: raw}

	// Verify
	vr, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !vr.Valid {
		t.Fatalf("Verify() valid = false; reason: %s", vr.Reason)
	}

	// Settle
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
