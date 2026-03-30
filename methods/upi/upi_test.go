package upi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *UPIMethod {
	t.Helper()
	m, err := New(Config{APIKey: "rzp_test_123", APISecret: "secret_123", MerchantVPA: "merchant@upi"})
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
		{"valid", Config{APIKey: "k", APISecret: "s", MerchantVPA: "m@upi"}, false, ""},
		{"missing api key", Config{APISecret: "s", MerchantVPA: "m@upi"}, true, "APIKey"},
		{"missing secret", Config{APIKey: "k", MerchantVPA: "m@upi"}, true, "APISecret"},
		{"missing vpa", Config{APIKey: "k", APISecret: "s"}, true, "MerchantVPA"},
		{"invalid vpa", Config{APIKey: "k", APISecret: "s", MerchantVPA: "novpa"}, true, "valid VPA"},
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
	if m.Name() != "upi" {
		t.Errorf("Name() = %q, want %q", m.Name(), "upi")
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
	if currencies[0] != core.INR {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.INR)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	// Happy path
	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.INR})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "upi" {
		t.Errorf("Method = %q, want %q", opt.Method, "upi")
	}
	if opt.Amount != "100" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "100")
	}
	if opt.Currency != core.INR {
		t.Errorf("Currency = %q, want %q", opt.Currency, core.INR)
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "100", Currency: core.INR})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
	}

	// Unsupported currency
	_, err = m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD (only INR supported)")
	}
	if !core.IsPaymentError(err, core.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got: %v", err)
	}
}

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	option := core.PaymentOption{Method: "upi", Currency: core.INR, Amount: "100"}
	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}

	// Must be valid JSON
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}

	// Required fields
	if p.VPA == "" {
		t.Error("CreatePayload() VPA is empty")
	}
	if p.TransactionRef == "" {
		t.Error("CreatePayload() TransactionRef is empty")
	}
	if p.UPITransactionID == "" {
		t.Error("CreatePayload() UPITransactionID is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "upi", Currency: core.INR, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		validPayload, _ := json.Marshal(Payload{
			VPA:              "user@upi",
			TransactionRef:   "ref_123",
			UPITransactionID: "txn_123",
		})
		payment := core.PaymentPayload{Payload: validPayload}
		resp, err := m.Verify(ctx, payment, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "user@upi" {
			t.Errorf("Payer = %q, want %q", resp.Payer, "user@upi")
		}
	})

	t.Run("missing vpa", func(t *testing.T) {
		p, _ := json.Marshal(Payload{TransactionRef: "ref", UPITransactionID: "txn"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing vpa")
		}
	})

	t.Run("invalid vpa format", func(t *testing.T) {
		p, _ := json.Marshal(Payload{VPA: "novpa", TransactionRef: "ref", UPITransactionID: "txn"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for bad vpa format")
		}
	})

	t.Run("missing transactionRef", func(t *testing.T) {
		p, _ := json.Marshal(Payload{VPA: "user@upi", UPITransactionID: "txn"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing transactionRef")
		}
	})

	t.Run("missing upiTransactionId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{VPA: "user@upi", TransactionRef: "ref"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing upiTransactionId")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		payment := core.PaymentPayload{Payload: json.RawMessage(`{bad json`)}
		_, err := m.Verify(ctx, payment, option)
		if err == nil {
			t.Error("Verify() expected error for malformed JSON")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		payment := core.PaymentPayload{Payload: json.RawMessage(`{}`)}
		resp, err := m.Verify(ctx, payment, option)
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
	option := core.PaymentOption{Method: "upi", Currency: core.INR, Amount: "100"}

	t.Run("valid payload", func(t *testing.T) {
		validPayload, _ := json.Marshal(Payload{
			VPA:              "user@upi",
			TransactionRef:   "ref_123",
			UPITransactionID: "txn_123",
		})
		payment := core.PaymentPayload{Payload: validPayload}
		resp, err := m.Settle(ctx, payment, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "upi" {
			t.Errorf("Method = %q, want %q", resp.Method, "upi")
		}
		if resp.Transaction == "" {
			t.Error("Settle() transaction is empty")
		}
		if resp.SettledAt == "" {
			t.Error("Settle() settledAt is empty")
		}
		if _, err := time.Parse(time.RFC3339, resp.SettledAt); err != nil {
			t.Errorf("Settle() settledAt is not valid RFC3339: %v", err)
		}

		// Check receipt
		var receipt map[string]string
		if err := json.Unmarshal(resp.Receipt, &receipt); err != nil {
			t.Fatalf("Settle() receipt is not valid JSON: %v", err)
		}
		if receipt["upiTransactionId"] != "txn_123" {
			t.Errorf("receipt upiTransactionId = %q, want %q", receipt["upiTransactionId"], "txn_123")
		}
		if receipt["vpa"] != "user@upi" {
			t.Errorf("receipt vpa = %q, want %q", receipt["vpa"], "user@upi")
		}
	})

	t.Run("missing upiTransactionId", func(t *testing.T) {
		p, _ := json.Marshal(Payload{VPA: "user@upi", TransactionRef: "ref"})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing upiTransactionId")
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
	option := core.PaymentOption{Intent: core.IntentCharge, Method: "upi", Currency: core.INR, Amount: "100"}

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
