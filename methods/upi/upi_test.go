package upi

import (
	"context"
	"encoding/json"
	"testing"

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
	}{
		{"valid", Config{APIKey: "k", APISecret: "s", MerchantVPA: "m@upi"}, false},
		{"missing api key", Config{APISecret: "s", MerchantVPA: "m@upi"}, true},
		{"missing secret", Config{APIKey: "k", MerchantVPA: "m@upi"}, true},
		{"missing vpa", Config{APIKey: "k", APISecret: "s"}, true},
		{"invalid vpa", Config{APIKey: "k", APISecret: "s", MerchantVPA: "novpa"}, true},
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
		t.Error("BuildOption() expected error for USD (only INR supported)")
	}

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "100", Currency: core.INR})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "upi" {
		t.Errorf("Method = %q, want %q", opt.Method, "upi")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		VPA:              "user@upi",
		TransactionRef:   "ref_123",
		UPITransactionID: "txn_123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "upi", Currency: core.INR, Amount: "100"}

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

	// Invalid VPA format
	badPayload, _ := json.Marshal(Payload{VPA: "novpa", TransactionRef: "ref", UPITransactionID: "txn"})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for bad vpa format")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		VPA:              "user@upi",
		TransactionRef:   "ref_123",
		UPITransactionID: "txn_123",
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "upi", Currency: core.INR, Amount: "100"}

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
}
