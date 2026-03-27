package openbanking

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *OpenBankingMethod {
	t.Helper()
	m, err := New(Config{
		APIKey:      "test_key",
		Provider:    "truelayer",
		RedirectURL: "https://example.com/callback",
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
		{"valid", Config{APIKey: "k", Provider: "truelayer", RedirectURL: "https://x.com/cb"}, false},
		{"missing api key", Config{Provider: "truelayer", RedirectURL: "https://x.com/cb"}, true},
		{"missing provider", Config{APIKey: "k", RedirectURL: "https://x.com/cb"}, true},
		{"missing redirect", Config{APIKey: "k", Provider: "truelayer"}, true},
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

func TestBuildOption_Intents(t *testing.T) {
	m := newTestMethod(t)

	// Should support charge and mandate
	for _, intent := range []core.Intent{core.IntentCharge, core.IntentMandate} {
		opt, err := m.BuildOption(intent, core.Price{Amount: "10.00", Currency: core.EUR})
		if err != nil {
			t.Errorf("BuildOption(%s) error: %v", intent, err)
		}
		if opt.Extra == nil {
			t.Errorf("BuildOption(%s) extra is nil", intent)
		}
	}

	// Should not support subscribe
	_, err := m.BuildOption(core.IntentSubscribe, core.Price{Amount: "10.00", Currency: core.EUR})
	if err == nil {
		t.Error("BuildOption(subscribe) expected error")
	}
}

func TestBuildOption_Currencies(t *testing.T) {
	m := newTestMethod(t)

	// EUR and GBP should work
	for _, cur := range []core.Currency{core.EUR, core.GBP} {
		_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: cur})
		if err != nil {
			t.Errorf("BuildOption(%s) error: %v", cur, err)
		}
	}

	// USD should fail
	_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption(USD) expected error")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		ConsentID: "consent_123",
		PaymentID: "pay_123",
		Provider:  "truelayer",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "openbanking", Currency: core.EUR, Amount: "10.00"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
	}

	// Missing consentId
	badPayload, _ := json.Marshal(Payload{PaymentID: "pay_123", Provider: "truelayer"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing consentId")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		ConsentID: "consent_123",
		PaymentID: "pay_123",
		Provider:  "truelayer",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "openbanking", Currency: core.EUR, Amount: "10.00"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false")
	}
	if resp.Method != "openbanking" {
		t.Errorf("Method = %q, want %q", resp.Method, "openbanking")
	}
}
