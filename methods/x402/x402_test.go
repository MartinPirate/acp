package x402

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func newTestMethod(t *testing.T) *X402Method {
	t.Helper()
	m, err := New(Config{
		FacilitatorURL: "https://facilitator.example.com",
		Network:        "eip155:8453",
		PrivateKey:     "0xdeadbeef",
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
		{"valid", Config{FacilitatorURL: "https://f.co", Network: "eip155:8453", PrivateKey: "0x1"}, false},
		{"missing facilitator", Config{Network: "eip155:8453", PrivateKey: "0x1"}, true},
		{"missing network", Config{FacilitatorURL: "https://f.co", PrivateKey: "0x1"}, true},
		{"missing key", Config{FacilitatorURL: "https://f.co", Network: "eip155:8453"}, true},
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
	if m.Name() != "x402" {
		t.Errorf("Name() = %q, want %q", m.Name(), "x402")
	}
}

func TestBuildOption_CurrencyRestriction(t *testing.T) {
	m := newTestMethod(t)

	_, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "1.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for USD (only USDC supported)")
	}

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "1.00", Currency: core.USDC})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Extra == nil {
		t.Error("BuildOption() extra is nil, expected network info")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		Signature: "0xsig_123",
		Authorization: Authorization{
			From:        "0xabc",
			To:          "0xdef",
			Value:       "1.00",
			ValidAfter:  time.Now().Add(-1 * time.Minute).Unix(),
			ValidBefore: time.Now().Add(5 * time.Minute).Unix(),
			Nonce:       "0x1",
		},
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "x402", Currency: core.USDC, Amount: "1.00"}

	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Verify() valid = false, want true; reason: %s", resp.Reason)
	}

	// Missing signature
	badPayload, _ := json.Marshal(Payload{
		Authorization: Authorization{From: "0xabc", To: "0xdef", Value: "1.00",
			ValidBefore: time.Now().Add(5 * time.Minute).Unix()},
	})
	payment.Payload = badPayload
	resp, err = m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false for missing signature")
	}
}

func TestSettle(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()

	validPayload, _ := json.Marshal(Payload{
		Signature: "0xsig_123",
		Authorization: Authorization{
			From:        "0xabc",
			To:          "0xdef",
			Value:       "1.00",
			ValidBefore: time.Now().Add(5 * time.Minute).Unix(),
			Nonce:       "0x1",
		},
	})
	payment := core.PaymentPayload{Payload: validPayload}
	option := core.PaymentOption{Method: "x402", Currency: core.USDC, Amount: "1.00"}

	resp, err := m.Settle(ctx, payment, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if !resp.Success {
		t.Error("Settle() success = false, want true")
	}
	if resp.Method != "x402" {
		t.Errorf("Settle() method = %q, want %q", resp.Method, "x402")
	}
}
