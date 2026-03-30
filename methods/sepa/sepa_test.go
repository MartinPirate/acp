package sepa

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
		errMsg  string
	}{
		{"valid", Config{APIKey: "k", IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Provider: "stripe"}, false, ""},
		{"missing api key", Config{IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Provider: "stripe"}, true, "APIKey"},
		{"missing iban", Config{APIKey: "k", BIC: "COBADEFFXXX", Provider: "stripe"}, true, "IBAN"},
		{"bad iban", Config{APIKey: "k", IBAN: "invalid", BIC: "COBADEFFXXX", Provider: "stripe"}, true, "IBAN format"},
		{"missing bic", Config{APIKey: "k", IBAN: "DE89370400440532013000", Provider: "stripe"}, true, "BIC"},
		{"missing provider", Config{APIKey: "k", IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX"}, true, "Provider"},
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
	if m.Name() != "sepa" {
		t.Errorf("Name() = %q, want %q", m.Name(), "sepa")
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
	if currencies[0] != core.EUR {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.EUR)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.EUR})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "sepa" {
		t.Errorf("Method = %q, want %q", opt.Method, "sepa")
	}
	if opt.Amount != "100" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "100")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentAuthorize, core.Price{Amount: "100", Currency: core.EUR})
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
	option := core.PaymentOption{Method: "sepa", Currency: core.EUR, Amount: "100"}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}
	if p.IBAN == "" {
		t.Error("CreatePayload() IBAN is empty")
	}
	if p.BIC == "" {
		t.Error("CreatePayload() BIC is empty")
	}
	if p.Reference == "" {
		t.Error("CreatePayload() Reference is empty")
	}
	if p.EndToEndID == "" {
		t.Error("CreatePayload() EndToEndID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "sepa", Currency: core.EUR, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX",
			Reference: "SEPA-123", EndToEndID: "E2E-123",
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "DE89370400440532013000" {
			t.Errorf("Payer = %q, want IBAN", resp.Payer)
		}
	})

	t.Run("missing iban", func(t *testing.T) {
		p, _ := json.Marshal(Payload{BIC: "COBADEFFXXX", Reference: "ref", EndToEndID: "e2e"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing iban")
		}
	})

	t.Run("invalid iban format", func(t *testing.T) {
		p, _ := json.Marshal(Payload{IBAN: "bad", BIC: "COBADEFFXXX", Reference: "ref", EndToEndID: "e2e"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for bad iban")
		}
	})

	t.Run("missing bic", func(t *testing.T) {
		p, _ := json.Marshal(Payload{IBAN: "DE89370400440532013000", Reference: "ref", EndToEndID: "e2e"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing bic")
		}
	})

	t.Run("missing reference", func(t *testing.T) {
		p, _ := json.Marshal(Payload{IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", EndToEndID: "e2e"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing reference")
		}
	})

	t.Run("missing endToEndId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Reference: "ref"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing endToEndId")
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
	option := core.PaymentOption{Method: "sepa", Currency: core.EUR, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX",
			Reference: "SEPA-123", EndToEndID: "E2E-123",
		})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "sepa" {
			t.Errorf("Method = %q, want %q", resp.Method, "sepa")
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
		if receipt["endToEndId"] != "E2E-123" {
			t.Errorf("receipt endToEndId = %q, want %q", receipt["endToEndId"], "E2E-123")
		}
		if receipt["provider"] != "stripe" {
			t.Errorf("receipt provider = %q, want %q", receipt["provider"], "stripe")
		}
	})

	t.Run("missing endToEndId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{IBAN: "DE89370400440532013000", BIC: "COBADEFFXXX", Reference: "ref"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing endToEndId")
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
	option := core.PaymentOption{Intent: core.IntentCharge, Method: "sepa", Currency: core.EUR, Amount: "100"}

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
