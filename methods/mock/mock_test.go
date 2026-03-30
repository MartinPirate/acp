package mock

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func TestNew(t *testing.T) {
	// Mock always succeeds (no required config fields)
	m := New(Config{})
	if m == nil {
		t.Fatal("New() returned nil")
	}

	// With ShouldFail config
	m2 := New(Config{ShouldFail: true})
	if m2 == nil {
		t.Fatal("New(ShouldFail) returned nil")
	}
}

func TestName(t *testing.T) {
	m := New(Config{})
	if m.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", m.Name(), "mock")
	}
}

func TestSupportedIntents(t *testing.T) {
	m := New(Config{})
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
	if !found[core.IntentAuthorize] {
		t.Error("SupportedIntents() missing IntentAuthorize")
	}
}

func TestSupportedCurrencies(t *testing.T) {
	m := New(Config{})
	currencies := m.SupportedCurrencies()
	if len(currencies) == 0 {
		t.Fatal("SupportedCurrencies() returned empty")
	}
	found := map[core.Currency]bool{}
	for _, c := range currencies {
		found[c] = true
	}
	for _, want := range []core.Currency{core.USD, core.EUR, core.GBP, core.INR, core.BRL, core.KES} {
		if !found[want] {
			t.Errorf("SupportedCurrencies() missing %s", want)
		}
	}
}

func TestBuildOption(t *testing.T) {
	m := New(Config{})

	opt, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USD})
	if err != nil {
		t.Fatalf("BuildOption() error: %v", err)
	}
	if opt.Method != "mock" {
		t.Errorf("Method = %q, want %q", opt.Method, "mock")
	}
	if opt.Amount != "10.00" {
		t.Errorf("Amount = %q, want %q", opt.Amount, "10.00")
	}
	if opt.Currency != core.USD {
		t.Errorf("Currency = %q, want %q", opt.Currency, core.USD)
	}

	// Unsupported intent
	_, err = m.BuildOption(core.IntentMandate, core.Price{Amount: "10.00", Currency: core.USD})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got: %v", err)
	}

	// Unsupported currency
	_, err = m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USDC})
	if err == nil {
		t.Error("BuildOption() expected error for unsupported currency")
	}
	if !core.IsPaymentError(err, core.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got: %v", err)
	}
}

func TestCreatePayload(t *testing.T) {
	m := New(Config{})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}

	raw, err := m.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}
	if raw == nil {
		t.Fatal("CreatePayload() returned nil")
	}

	var p mockPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("CreatePayload() returned invalid JSON: %v", err)
	}
	if p.Token == "" {
		t.Error("CreatePayload() Token is empty")
	}
}

func TestVerify(t *testing.T) {
	m := New(Config{})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(mockPayload{Token: "mock_tok_123"})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if !resp.Valid {
			t.Errorf("Verify() valid = false; reason: %s", resp.Reason)
		}
		if resp.Payer != "mock-payer" {
			t.Errorf("Payer = %q, want %q", resp.Payer, "mock-payer")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		p, _ := json.Marshal(mockPayload{Token: ""})
		resp, err := m.Verify(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if resp.Valid {
			t.Error("Verify() valid = true, want false for empty token")
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
	m := New(Config{})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}

	t.Run("valid payload", func(t *testing.T) {
		p, _ := json.Marshal(mockPayload{Token: "mock_tok_123"})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
		if resp.Method != "mock" {
			t.Errorf("Method = %q, want %q", resp.Method, "mock")
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
	})

	t.Run("malformed JSON payload still settles", func(t *testing.T) {
		// Settle does not unmarshal the payload, it just records it
		p, _ := json.Marshal(mockPayload{Token: "tok"})
		resp, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
		if err != nil {
			t.Fatalf("Settle() error: %v", err)
		}
		if !resp.Success {
			t.Error("Settle() success = false")
		}
	})
}

func TestFullRoundTrip(t *testing.T) {
	m := New(Config{})
	ctx := context.Background()

	option, err := m.BuildOption(core.IntentCharge, core.Price{Amount: "10.00", Currency: core.USD})
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

func TestTransactions(t *testing.T) {
	m := New(Config{})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}

	// Initially empty
	if len(m.Transactions()) != 0 {
		t.Fatalf("Transactions() = %d, want 0", len(m.Transactions()))
	}

	// Settle a payment
	p, _ := json.Marshal(mockPayload{Token: "tok_1"})
	_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}

	txns := m.Transactions()
	if len(txns) != 1 {
		t.Fatalf("Transactions() = %d, want 1", len(txns))
	}
	if txns[0].Amount != "10.00" {
		t.Errorf("Transaction amount = %q, want %q", txns[0].Amount, "10.00")
	}
	if txns[0].Currency != core.USD {
		t.Errorf("Transaction currency = %q, want %q", txns[0].Currency, core.USD)
	}
	if txns[0].ID == "" {
		t.Error("Transaction ID is empty")
	}

	// Settle another
	p2, _ := json.Marshal(mockPayload{Token: "tok_2"})
	option2 := core.PaymentOption{Method: "mock", Currency: core.EUR, Amount: "25.00"}
	_, err = m.Settle(ctx, core.PaymentPayload{Payload: p2}, option2)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}

	if len(m.Transactions()) != 2 {
		t.Fatalf("Transactions() = %d, want 2", len(m.Transactions()))
	}
}

func TestReset(t *testing.T) {
	m := New(Config{})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}

	// Settle a payment
	p, _ := json.Marshal(mockPayload{Token: "tok_1"})
	_, err := m.Settle(ctx, core.PaymentPayload{Payload: p}, option)
	if err != nil {
		t.Fatalf("Settle() error: %v", err)
	}
	if len(m.Transactions()) != 1 {
		t.Fatalf("Transactions() = %d, want 1", len(m.Transactions()))
	}

	// Reset
	m.Reset()
	if len(m.Transactions()) != 0 {
		t.Fatalf("After Reset(), Transactions() = %d, want 0", len(m.Transactions()))
	}
}

func TestShouldFail(t *testing.T) {
	m := New(Config{ShouldFail: true})
	ctx := context.Background()
	option := core.PaymentOption{Method: "mock", Currency: core.USD, Amount: "10.00"}
	p, _ := json.Marshal(mockPayload{Token: "tok_1"})
	payment := core.PaymentPayload{Payload: p}

	// Verify should return Valid=false
	resp, err := m.Verify(ctx, payment, option)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if resp.Valid {
		t.Error("Verify() valid = true, want false when ShouldFail is set")
	}
	if resp.Reason == "" {
		t.Error("Verify() reason should not be empty when ShouldFail is set")
	}

	// Settle should return an error
	_, err = m.Settle(ctx, payment, option)
	if err == nil {
		t.Error("Settle() expected error when ShouldFail is set")
	}
	if !core.IsPaymentError(err, core.ErrSettlementFailed) {
		t.Errorf("expected ErrSettlementFailed, got: %v", err)
	}
}
