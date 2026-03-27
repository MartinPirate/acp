package card

import (
	"context"
	"encoding/json"
	"testing"

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
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{Token: "tok_123", PaymentIntentID: "pi_456"})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "card", Currency: core.USD, Amount: "10.00"}

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
}
