package openbanking

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestName(t *testing.T) {
	m := newTestMethod(t)
	if m.Name() != "openbanking" {
		t.Errorf("Name() = %q, want %q", m.Name(), "openbanking")
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
	if !found[core.IntentMandate] {
		t.Error("SupportedIntents() missing IntentMandate")
	}
}

func TestSupportedCurrencies(t *testing.T) {
	m := newTestMethod(t)
	currencies := m.SupportedCurrencies()
	if len(currencies) != 2 {
		t.Fatalf("SupportedCurrencies() returned %d, want 2", len(currencies))
	}
	found := map[core.Currency]bool{}
	for _, c := range currencies {
		found[c] = true
	}
	if !found[core.EUR] {
		t.Error("SupportedCurrencies() missing EUR")
	}
	if !found[core.GBP] {
		t.Error("SupportedCurrencies() missing GBP")
	}
}

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "openbanking", Currency: core.EUR, Amount: "10.00"}

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
	if p.ConsentID == "" {
		t.Error("CreatePayload() ConsentID is empty")
	}
	if p.PaymentID == "" {
		t.Error("CreatePayload() PaymentID is empty")
	}
	if p.Provider == "" {
		t.Error("CreatePayload() Provider is empty")
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
	option := core.PaymentOption{Method: "openbanking", Currency: core.EUR, Amount: "10.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{ConsentID: "consent_123", PaymentID: "pay_123", Provider: "truelayer"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
	})

	t.Run("missing consentId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{PaymentID: "pay_123", Provider: "truelayer"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing consentId")
		}
	})

	t.Run("missing paymentId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{ConsentID: "consent_123", Provider: "truelayer"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing paymentId")
		}
	})

	t.Run("missing provider", func(t *testing.T) {
		p, _ := json.Marshal(Payload{ConsentID: "consent_123", PaymentID: "pay_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing provider")
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
	option := core.PaymentOption{Method: "openbanking", Currency: core.EUR, Amount: "10.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{ConsentID: "consent_123", PaymentID: "pay_123", Provider: "truelayer"})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "openbanking" {
			t.Errorf("Method = %q, want %q", resp.Method, "openbanking")
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
		if receipt["consentId"] != "consent_123" {
			t.Errorf("receipt consentId = %q, want %q", receipt["consentId"], "consent_123")
		}
		if receipt["provider"] != "truelayer" {
			t.Errorf("receipt provider = %q, want %q", receipt["provider"], "truelayer")
		}
	})

	t.Run("missing paymentId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{ConsentID: "consent_123", Provider: "truelayer"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing paymentId")
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

	option, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.EUR})
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
