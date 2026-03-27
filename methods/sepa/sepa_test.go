package sepa

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *SepaMethod {
	t.Helper()
	m, err := New(Config{
		APIKey:   "test_key",
		IBAN:     "DE89370400440532013000",
		BIC:      "COBADEFFXXX",
		Provider: "stripe",
	})
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
		{"valid", Config{APIKey: "k", IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Provider: "stripe"}, false},
		{"missing api key", Config{IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Provider: "stripe"}, true},
		{"missing iban", Config{APIKey: "k", BIC: "COBADEFFXXX", Provider: "stripe"}, true},
		{"bad iban", Config{APIKey: "k", IBAN: "invalid", BIC: "COBADEFFXXX", Provider: "stripe"}, true},
		{"missing bic", Config{APIKey: "k", IBAN: "DE89370400440532013000", Provider: "stripe"}, true},
		{"missing provider", Config{APIKey: "k", IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX"}, true},
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

	_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD")
	}

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.EUR})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "sepa" {
		t.Errorf("Method = %q, want %q", opt.Method, "sepa")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		IBAN:       "DE89370400440532013000",
		BIC:        "COBADEFFXXX",
		Reference:  "SEPA-123",
		EndToEndID: "E2E-123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "sepa", Currency: core.EUR, Amount: "100"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
	}
	if resp.Payer != "DE89370400440532013000" {
		t.Errorf("Payer = %q, want IBAN", resp.Payer)
	}

	// Invalid IBAN format
	badPayload, _ := json.Marshal(Payload{IBAN: "bad", BIC: "COBADEFFXXX", Reference: "ref", EndToEndID: "e2e"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for bad iban")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		IBAN:       "DE89370400440532013000",
		BIC:        "COBADEFFXXX",
		Reference:  "SEPA-123",
		EndToEndID: "E2E-123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "sepa", Currency: core.EUR, Amount: "100"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false")
	}
	if resp.Method != "sepa" {
		t.Errorf("Method = %q, want %q", resp.Method, "sepa")
	}
}
