package pix

import (
	"context"
	"encoding/json"
	"testing"

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
	}{
		{"valid", Config{APIKey: "k", PixKey: "pk", Provider: "stripe"}, false},
		{"missing api key", Config{PixKey: "pk", Provider: "stripe"}, true},
		{"missing pix key", Config{APIKey: "k", Provider: "stripe"}, true},
		{"missing provider", Config{APIKey: "k", PixKey: "pk"}, true},
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

func TestBuildOption_CurrencyRestriction(t *testing.T) {
	m := newTestMethod(t)

	_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "50.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD")
	}

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "50.00", Currency: core.BRL})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "pix" {
		t.Errorf("Method = %q, want %q", opt.Method, "pix")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123", TxID: "tx_123"})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "pix", Currency: core.BRL, Amount: "50.00"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
	}

	// Missing field
	badPayload, _ := json.Marshal(Payload{PixKey: "12345678901"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing e2eId")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{PixKey: "12345678901", E2eID: "E123", TxID: "tx_123"})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "pix", Currency: core.BRL, Amount: "50.00"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false")
	}
	if resp.Method != "pix" {
		t.Errorf("Method = %q, want %q", resp.Method, "pix")
	}
}
