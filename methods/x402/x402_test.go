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
	if currencies[0] != core.USDC {
		t.Errorf("SupportedCurrencies()[0] = %q, want %q", currencies[0], core.USDC)
	}
}

func TestBuildOption(t *testing.T) {
	m := newTestMethod(t)

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "1.00", Currency: core.USDC})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "x402" {
		t.Errorf("Method = %q, want %q", opt.Method, "x402")
	}
	if opt.Amount != "1.00" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "1.00")
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "1.00", Currency: core.USDC})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
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

func TestCreatePayload(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "x402", Currency: core.USDC, Amount: "1.00"}

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
	if p.Signature == "" {
		t.Error("CreatePayload() Signature is empty")
	}
	if p.Authorization.From == "" {
		t.Error("CreatePayload() Authorization.From is empty")
	}
	if p.Authorization.To == "" {
		t.Error("CreatePayload() Authorization.To is empty")
	}
	if p.Authorization.Value == "" {
		t.Error("CreatePayload() Authorization.Value is empty")
	}
}

func TestVerify(t *testing.T) {
	m := newTestMethod(t)
	ctx := context.Background()
	option := core.PaymentOption{Method: "x402", Currency: core.USDC, Amount: "1.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig_123",
			Authorization: Authorization{
				From: "0xabc", To: "0xdef", Value: "1.00",
				ValidAfter:  time.Now().Add(-1 * time.Minute).Unix(),
				ValidBefore: time.Now().Add(5 * time.Minute).Unix(),
				Nonce:       "0x1",
			},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "0xabc" {
			t.Errorf("Payer = %q, want %q", resp.Payer, "0xabc")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Authorization: Authorization{From: "0xabc", To: "0xdef", Value: "1.00",
				ValidBefore: time.Now().Add(5 * time.Minute).Unix()},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing signature")
		}
	})

	t.Run("missing from", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig",
			Authorization: Authorization{To: "0xdef", Value: "1.00",
				ValidBefore: time.Now().Add(5 * time.Minute).Unix()},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing from")
		}
	})

	t.Run("missing to", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig",
			Authorization: Authorization{From: "0xabc", Value: "1.00",
				ValidBefore: time.Now().Add(5 * time.Minute).Unix()},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing to")
		}
	})

	t.Run("missing value", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig",
			Authorization: Authorization{From: "0xabc", To: "0xdef",
				ValidBefore: time.Now().Add(5 * time.Minute).Unix()},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for missing value")
		}
	})

	t.Run("expired authorization", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig",
			Authorization: Authorization{From: "0xabc", To: "0xdef", Value: "1.00",
				ValidBefore: time.Now().Add(-1 * time.Minute).Unix()},
		})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for expired authorization")
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
	option := core.PaymentOption{Method: "x402", Currency: core.USDC, Amount: "1.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Signature: "0xsig_123",
			Authorization: Authorization{
				From: "0xabc", To: "0xdef", Value: "1.00",
				ValidBefore: time.Now().Add(5 * time.Minute).Unix(),
				Nonce:       "0x1",
			},
		})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false, want true")
		}
		if resp.Method != "x402" {
			t.Errorf("Settle() method = %q, want %q", resp.Method, "x402")
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
		if receipt["network"] != "eip155:8453" {
			t.Errorf("receipt network = %q, want %q", receipt["network"], "eip155:8453")
		}
		if receipt["from"] != "0xabc" {
			t.Errorf("receipt from = %q, want %q", receipt["from"], "0xabc")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		p, _ := json.Marshal(Payload{
			Authorization: Authorization{From: "0xabc", To: "0xdef", Value: "1.00"},
		})
		_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err == nil {
			t.Error("Settle() expected error for missing signature")
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

	option, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "1.00", Currency: core.USDC})
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
