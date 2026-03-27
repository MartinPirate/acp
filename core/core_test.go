package core

import (
	"encoding/json"
	"testing"
)

func TestIntentIsValid(t *testing.T) {
	tests := []struct {
		intent Intent
		valid  bool
	}{
		{IntentCharge, true},
		{IntentAuthorize, true},
		{IntentSubscribe, true},
		{IntentMandate, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.intent.IsValid(); got != tt.valid {
			t.Errorf("Intent(%q).IsValid() = %v, want %v", tt.intent, got, tt.valid)
		}
	}
}

func TestValidIntents(t *testing.T) {
	intents := ValidIntents()
	if len(intents) != 4 {
		t.Errorf("ValidIntents() returned %d intents, want 4", len(intents))
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		amount string
		valid  bool
	}{
		{"5.99", true},
		{"0", true},
		{"0.00", true},
		{"1000000", true},
		{"0.000001", true},
		{"", false},
		{"abc", false},
		{"-1", false},
	}
	for _, tt := range tests {
		err := ParseAmount(tt.amount)
		if (err == nil) != tt.valid {
			t.Errorf("ParseAmount(%q) error = %v, wantValid = %v", tt.amount, err, tt.valid)
		}
	}
}

func TestCompareAmounts(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"5.99", "5.99", 0},
		{"5.99", "6.00", -1},
		{"10.00", "5.99", 1},
		{"0", "0", 0},
		{"0.000001", "0.000002", -1},
	}
	for _, tt := range tests {
		got, err := CompareAmounts(tt.a, tt.b)
		if err != nil {
			t.Errorf("CompareAmounts(%q, %q) error = %v", tt.a, tt.b, err)
			continue
		}
		if got != tt.want {
			t.Errorf("CompareAmounts(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLookupCurrency(t *testing.T) {
	info, ok := LookupCurrency(USD)
	if !ok {
		t.Fatal("LookupCurrency(USD) not found")
	}
	if info.MinorUnits != 2 {
		t.Errorf("USD minor units = %d, want 2", info.MinorUnits)
	}

	info, ok = LookupCurrency(USDC)
	if !ok {
		t.Fatal("LookupCurrency(USDC) not found")
	}
	if info.MinorUnits != 6 || !info.IsCrypto {
		t.Errorf("USDC info = %+v, want minorUnits=6, isCrypto=true", info)
	}

	_, ok = LookupCurrency("FAKE")
	if ok {
		t.Error("LookupCurrency(FAKE) should not be found")
	}
}

func TestPaymentRequiredJSON(t *testing.T) {
	pr := PaymentRequired{
		ACPVersion: ACPVersion,
		Resource: Resource{
			URL:         "https://api.example.com/data",
			Description: "Market data",
			MimeType:    "application/json",
		},
		Accepts: []PaymentOption{
			{
				Intent:      IntentCharge,
				Method:      "mock",
				Currency:    USD,
				Amount:      "5.99",
				Description: "Mock payment",
			},
		},
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded PaymentRequired
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ACPVersion != ACPVersion {
		t.Errorf("acpVersion = %d, want %d", decoded.ACPVersion, ACPVersion)
	}
	if decoded.Resource.URL != pr.Resource.URL {
		t.Errorf("resource.url = %q, want %q", decoded.Resource.URL, pr.Resource.URL)
	}
	if len(decoded.Accepts) != 1 {
		t.Fatalf("accepts length = %d, want 1", len(decoded.Accepts))
	}
	if decoded.Accepts[0].Method != "mock" {
		t.Errorf("accepts[0].method = %q, want %q", decoded.Accepts[0].Method, "mock")
	}
}

func TestPaymentErrorInterface(t *testing.T) {
	err := NewPaymentError(ErrInsufficientFunds, "not enough money")
	if err.Error() != "insufficient_funds: not enough money" {
		t.Errorf("Error() = %q", err.Error())
	}

	methodErr := NewMethodError(ErrSettlementFailed, "card", "declined")
	if methodErr.Error() != "settlement_failed [card]: declined" {
		t.Errorf("Error() = %q", methodErr.Error())
	}

	if !IsPaymentError(err) {
		t.Error("IsPaymentError should return true")
	}
	if !IsPaymentError(err, ErrInsufficientFunds) {
		t.Error("IsPaymentError should match ErrInsufficientFunds")
	}
	if IsPaymentError(err, ErrTimeout) {
		t.Error("IsPaymentError should not match ErrTimeout")
	}
}

func TestBudgetEnforcer(t *testing.T) {
	b := NewBudgetEnforcer(Budget{
		MaxPerRequest: "10.00",
		MaxPerSession: "50.00",
		Currency:      USD,
	})

	// Within limits.
	if err := b.Check("5.00", USD); err != nil {
		t.Errorf("Check(5.00) should pass: %v", err)
	}

	// Exceeds per-request.
	if err := b.Check("15.00", USD); err == nil {
		t.Error("Check(15.00) should fail per-request limit")
	}

	// Wrong currency.
	if err := b.Check("5.00", EUR); err == nil {
		t.Error("Check with EUR should fail currency mismatch")
	}

	// Record spending and check session limit.
	if err := b.Record("20.00", USD); err != nil {
		t.Errorf("Record(20.00) error: %v", err)
	}
	if err := b.Record("20.00", USD); err != nil {
		t.Errorf("Record(20.00) error: %v", err)
	}
	// Now at 40.00 — next 10.01 should exceed session limit (50.00).
	if err := b.Check("10.01", USD); err == nil {
		t.Error("Check(10.01) should fail session limit after 40.00 spent")
	}
	// But 10.00 should be fine.
	if err := b.Check("10.00", USD); err != nil {
		t.Errorf("Check(10.00) should pass: %v", err)
	}
}

func TestIdempotencyKey(t *testing.T) {
	k1 := IdempotencyKey("agent-1", "/api/data", "mock", "5.99")
	k2 := IdempotencyKey("agent-1", "/api/data", "mock", "5.99")
	k3 := IdempotencyKey("agent-2", "/api/data", "mock", "5.99")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different inputs should produce different key")
	}
	if len(k1) != 32 { // 16 bytes hex-encoded
		t.Errorf("key length = %d, want 32", len(k1))
	}
}
