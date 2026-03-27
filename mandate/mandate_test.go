package mandate

import (
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func TestNewMandate(t *testing.T) {
	m := NewMandate(
		WithID("m-1"),
		WithAgentID("agent-1"),
		WithMaxAmount("100.00"),
		WithCurrency(core.USD),
		WithMaxPerRequest("10.00"),
		WithAllowedMethods("card", "mock"),
		WithAllowedIntents(core.IntentCharge),
		WithScope("/api/*"),
		WithExpiry(time.Now().Add(time.Hour)),
		WithMetadata(map[string]string{"env": "test"}),
	)

	if m.ID != "m-1" {
		t.Errorf("ID = %q, want %q", m.ID, "m-1")
	}
	if m.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", m.AgentID, "agent-1")
	}
	if m.MaxAmount != "100.00" {
		t.Errorf("MaxAmount = %q, want %q", m.MaxAmount, "100.00")
	}
	if m.Currency != core.USD {
		t.Errorf("Currency = %q, want %q", m.Currency, core.USD)
	}
	if len(m.AllowedMethods) != 2 {
		t.Errorf("AllowedMethods len = %d, want 2", len(m.AllowedMethods))
	}
	if m.Metadata["env"] != "test" {
		t.Errorf("Metadata[env] = %q, want %q", m.Metadata["env"], "test")
	}
}

func makePayload(method string, intent core.Intent, amount string, currency core.Currency, resourceURL string) core.PaymentPayload {
	return core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Resource:   core.Resource{URL: resourceURL},
		Accepted: core.PaymentOption{
			Method:   method,
			Intent:   intent,
			Amount:   amount,
			Currency: currency,
		},
	}
}

func TestEnforcerCheckExpiry(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-expired"),
		WithCurrency(core.USD),
		WithExpiry(time.Now().Add(-time.Hour)),
	)
	payload := makePayload("mock", core.IntentCharge, "5.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for expired mandate")
	}
	if !core.IsPaymentError(err, core.ErrMandateExpired) {
		t.Errorf("expected ErrMandateExpired, got %v", err)
	}
}

func TestEnforcerCheckCurrencyMismatch(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-cur"),
		WithCurrency(core.USD),
		WithExpiry(time.Now().Add(time.Hour)),
	)
	payload := makePayload("mock", core.IntentCharge, "5.00", core.EUR, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for currency mismatch")
	}
	if !core.IsPaymentError(err, core.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestEnforcerCheckMethod(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-method"),
		WithCurrency(core.USD),
		WithAllowedMethods("card"),
		WithExpiry(time.Now().Add(time.Hour)),
	)
	payload := makePayload("crypto", core.IntentCharge, "5.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for disallowed method")
	}
	if !core.IsPaymentError(err, core.ErrMethodUnavailable) {
		t.Errorf("expected ErrMethodUnavailable, got %v", err)
	}
}

func TestEnforcerCheckIntent(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-intent"),
		WithCurrency(core.USD),
		WithAllowedIntents(core.IntentCharge),
		WithExpiry(time.Now().Add(time.Hour)),
	)
	payload := makePayload("mock", core.IntentSubscribe, "5.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for disallowed intent")
	}
	if !core.IsPaymentError(err, core.ErrUnsupportedIntent) {
		t.Errorf("expected ErrUnsupportedIntent, got %v", err)
	}
}

func TestEnforcerCheckScope(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-scope"),
		WithCurrency(core.USD),
		WithScope("/api/*"),
		WithExpiry(time.Now().Add(time.Hour)),
	)

	// Out of scope.
	payload := makePayload("mock", core.IntentCharge, "5.00", core.USD, "/other/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for out-of-scope resource")
	}
	if !core.IsPaymentError(err, core.ErrMandateExceeded) {
		t.Errorf("expected ErrMandateExceeded, got %v", err)
	}

	// In scope.
	payload2 := makePayload("mock", core.IntentCharge, "5.00", core.USD, "/api/data")
	err = e.Check(m, payload2, payload2.Resource)
	if err != nil {
		t.Errorf("expected nil for in-scope resource, got %v", err)
	}
}

func TestEnforcerCheckPerRequestCap(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-perreq"),
		WithCurrency(core.USD),
		WithMaxPerRequest("10.00"),
		WithExpiry(time.Now().Add(time.Hour)),
	)
	payload := makePayload("mock", core.IntentCharge, "15.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for per-request cap exceeded")
	}
	if !core.IsPaymentError(err, core.ErrAmountTooHigh) {
		t.Errorf("expected ErrAmountTooHigh, got %v", err)
	}
}

func TestEnforcerCheckCumulativeSpend(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-cumulative"),
		WithCurrency(core.USD),
		WithMaxAmount("20.00"),
		WithExpiry(time.Now().Add(time.Hour)),
	)

	// Record some spend.
	if err := e.Record(m, "15.00"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// This should exceed the total.
	payload := makePayload("mock", core.IntentCharge, "10.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err == nil {
		t.Fatal("expected error for cumulative spend exceeded")
	}
	if !core.IsPaymentError(err, core.ErrMandateExceeded) {
		t.Errorf("expected ErrMandateExceeded, got %v", err)
	}

	// This should still be within limits.
	payload2 := makePayload("mock", core.IntentCharge, "5.00", core.USD, "/api/resource")
	err = e.Check(m, payload2, payload2.Resource)
	if err != nil {
		t.Errorf("expected nil for within-budget payment, got %v", err)
	}
}

func TestEnforcerSpent(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(WithID("m-spent"))

	if got := e.Spent(m.ID); got != "0" {
		t.Errorf("Spent() = %q, want %q", got, "0")
	}

	if err := e.Record(m, "5.50"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	if err := e.Record(m, "3.25"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	spent := e.Spent(m.ID)
	// Should be 8.75
	cmp, err := core.CompareAmounts(spent, "8.75")
	if err != nil {
		t.Fatalf("CompareAmounts error: %v", err)
	}
	if cmp != 0 {
		t.Errorf("Spent() = %q, want equivalent to 8.75", spent)
	}
}

func TestEnforcerCheckValidMandate(t *testing.T) {
	e := NewEnforcer()
	m := NewMandate(
		WithID("m-valid"),
		WithCurrency(core.USD),
		WithMaxAmount("100.00"),
		WithMaxPerRequest("25.00"),
		WithAllowedMethods("mock"),
		WithAllowedIntents(core.IntentCharge),
		WithScope("/api/*"),
		WithExpiry(time.Now().Add(time.Hour)),
	)

	payload := makePayload("mock", core.IntentCharge, "10.00", core.USD, "/api/resource")
	err := e.Check(m, payload, payload.Resource)
	if err != nil {
		t.Errorf("expected nil for valid mandate check, got %v", err)
	}
}

// --- Store Tests ---

func TestMemoryStoreSaveAndGet(t *testing.T) {
	store := NewMemoryStore()
	m := NewMandate(WithID("s-1"), WithAgentID("agent-1"))

	if err := store.Save(m); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Get("s-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-1")
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mandate")
	}
}

func TestMemoryStoreSaveEmptyID(t *testing.T) {
	store := NewMemoryStore()
	m := NewMandate()
	err := store.Save(m)
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestMemoryStoreList(t *testing.T) {
	store := NewMemoryStore()
	store.Save(NewMandate(WithID("l-1"), WithAgentID("agent-a")))
	store.Save(NewMandate(WithID("l-2"), WithAgentID("agent-b")))
	store.Save(NewMandate(WithID("l-3"), WithAgentID("agent-a")))

	list, err := store.List("agent-a")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List returned %d mandates, want 2", len(list))
	}
}

func TestMemoryStoreRevoke(t *testing.T) {
	store := NewMemoryStore()
	store.Save(NewMandate(WithID("r-1"), WithAgentID("agent-1")))

	if err := store.Revoke("r-1"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	_, err := store.Get("r-1")
	if err == nil {
		t.Fatal("expected error after revocation")
	}
}

func TestMemoryStoreRevokeNotFound(t *testing.T) {
	store := NewMemoryStore()
	err := store.Revoke("nonexistent")
	if err == nil {
		t.Fatal("expected error for revoking nonexistent mandate")
	}
}

func TestMatchesScope(t *testing.T) {
	tests := []struct {
		patterns []string
		url      string
		want     bool
	}{
		{[]string{"/api/*"}, "/api/data", true},
		{[]string{"/api/*"}, "/other/data", false},
		{[]string{"/api/*", "/v2/*"}, "/v2/resource", true},
		{[]string{"/*"}, "/anything", true},
		{nil, "/anything", false},
	}

	for _, tt := range tests {
		got := matchesScope(tt.patterns, tt.url)
		if got != tt.want {
			t.Errorf("matchesScope(%v, %q) = %v, want %v", tt.patterns, tt.url, got, tt.want)
		}
	}
}
