package mpesa

import (
	"context"
	"encoding/json"
	"testing"

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
	}{
		{"valid", func(c Config) Config { return c }, false},
		{"missing key", func(c Config) Config { c.ConsumerKey = ""; return c }, true},
		{"missing secret", func(c Config) Config { c.ConsumerSecret = ""; return c }, true},
		{"missing shortcode", func(c Config) Config { c.ShortCode = ""; return c }, true},
		{"missing passkey", func(c Config) Config { c.PassKey = ""; return c }, true},
		{"bad env", func(c Config) Config { c.Environment = "staging"; return c }, true},
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

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.KES})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "mpesa" {
		t.Errorf("Method = %q, want %q", opt.Method, "mpesa")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		PhoneNumber:       "254712345678",
		AccountRef:        "ACP_123",
		TransactionID:     "txn_123",
		CheckoutRequestID: "ws_CO_123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "mpesa", Currency: core.KES, Amount: "100"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
	}
	if resp.Payer != "254712345678" {
		t.Errorf("Payer = %q, want %q", resp.Payer, "254712345678")
	}

	// Invalid phone number
	badPayload, _ := json.Marshal(Payload{
		PhoneNumber:       "0712345678",
		AccountRef:        "ACP_123",
		CheckoutRequestID: "ws_CO_123",
	})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for bad phone format")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		PhoneNumber:       "254712345678",
		AccountRef:        "ACP_123",
		TransactionID:     "txn_123",
		CheckoutRequestID: "ws_CO_123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "mpesa", Currency: core.KES, Amount: "100"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false")
	}
	if resp.Method != "mpesa" {
		t.Errorf("Method = %q, want %q", resp.Method, "mpesa")
	}
}
