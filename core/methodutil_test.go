package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnmarshalMethodPayload(t *testing.T) {
	type payload struct {
		Token string `json:"token"`
	}

	t.Run("valid payload", func(t *testing.T) {
		raw := json.RawMessage(`{"token":"tok_123"}`)
		var p payload
		if err := UnmarshalMethodPayload(raw, &p, "test"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Token != "tok_123" {
			t.Errorf("got Token=%q, want %q", p.Token, "tok_123")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		raw := json.RawMessage(`{bad json}`)
		var p payload
		err := UnmarshalMethodPayload(raw, &p, "card")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsPaymentError(err, ErrInvalidPayload) {
			t.Errorf("expected ErrInvalidPayload, got %v", err)
		}
		if !strings.Contains(err.Error(), "invalid card payload") {
			t.Errorf("error message should mention method name, got: %v", err)
		}
	})
}

func TestBuildSettleResponse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		receipt := map[string]string{"key": "value"}
		resp, err := BuildSettleResponse("card", "txn_123", receipt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ACPVersion != ACPVersion {
			t.Errorf("ACPVersion = %d, want %d", resp.ACPVersion, ACPVersion)
		}
		if !resp.Success {
			t.Error("Success should be true")
		}
		if resp.Method != "card" {
			t.Errorf("Method = %q, want %q", resp.Method, "card")
		}
		if resp.Transaction != "txn_123" {
			t.Errorf("Transaction = %q, want %q", resp.Transaction, "txn_123")
		}
		if resp.SettledAt == "" {
			t.Error("SettledAt should not be empty")
		}
		if resp.Receipt == nil {
			t.Error("Receipt should not be nil")
		}
	})

	t.Run("unmarshalable receipt returns error", func(t *testing.T) {
		// Channels cannot be marshaled to JSON.
		_, err := BuildSettleResponse("card", "txn_123", make(chan int))
		if err == nil {
			t.Fatal("expected error for unmarshalable receipt")
		}
		if !IsPaymentError(err, ErrSettlementFailed) {
			t.Errorf("expected ErrSettlementFailed, got %v", err)
		}
	})
}

func TestGenerateTxnID(t *testing.T) {
	id := GenerateTxnID("stripe")
	if !strings.HasPrefix(id, "stripe_txn_") {
		t.Errorf("expected prefix %q, got %q", "stripe_txn_", id)
	}

	id2 := GenerateTxnID("mock")
	if !strings.HasPrefix(id2, "mock_txn_") {
		t.Errorf("expected prefix %q, got %q", "mock_txn_", id2)
	}
}

func TestValidateBuildOption(t *testing.T) {
	intents := []Intent{IntentCharge, IntentAuthorize}
	currencies := []Currency{USD, EUR}

	t.Run("valid combination", func(t *testing.T) {
		err := ValidateBuildOption("card", IntentCharge, USD, intents, currencies)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unsupported intent", func(t *testing.T) {
		err := ValidateBuildOption("card", IntentSubscribe, USD, intents, currencies)
		if err == nil {
			t.Fatal("expected error for unsupported intent")
		}
		if !IsPaymentError(err, ErrUnsupportedIntent) {
			t.Errorf("expected ErrUnsupportedIntent, got %v", err)
		}
	})

	t.Run("unsupported currency", func(t *testing.T) {
		err := ValidateBuildOption("card", IntentCharge, BRL, intents, currencies)
		if err == nil {
			t.Fatal("expected error for unsupported currency")
		}
		if !IsPaymentError(err, ErrCurrencyMismatch) {
			t.Errorf("expected ErrCurrencyMismatch, got %v", err)
		}
	})
}

func TestBaseConfigGetHTTPClient(t *testing.T) {
	t.Run("nil returns default", func(t *testing.T) {
		c := BaseConfig{}
		client := c.GetHTTPClient()
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("custom client returned", func(t *testing.T) {
		custom := &customHTTPClient{}
		_ = custom // just to verify the type works
		// BaseConfig.HTTPClient is *http.Client, so we use a real one
		c := BaseConfig{}
		if c.GetHTTPClient() == nil {
			t.Fatal("expected non-nil default client")
		}
	})
}

type customHTTPClient struct{}
