package core

import "testing"

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !ContainsString(slice, "b") {
		t.Error("expected true for existing element")
	}
	if ContainsString(slice, "d") {
		t.Error("expected false for missing element")
	}
	if ContainsString(nil, "a") {
		t.Error("expected false for nil slice")
	}
}

func TestContainsIntent(t *testing.T) {
	slice := []Intent{IntentCharge, IntentAuthorize}

	if !ContainsIntent(slice, IntentCharge) {
		t.Error("expected true for existing intent")
	}
	if ContainsIntent(slice, IntentSubscribe) {
		t.Error("expected false for missing intent")
	}
	if ContainsIntent(nil, IntentCharge) {
		t.Error("expected false for nil slice")
	}
}

func TestContainsCurrency(t *testing.T) {
	slice := []Currency{USD, EUR}

	if !ContainsCurrency(slice, USD) {
		t.Error("expected true for existing currency")
	}
	if ContainsCurrency(slice, BRL) {
		t.Error("expected false for missing currency")
	}
	if ContainsCurrency(nil, USD) {
		t.Error("expected false for nil slice")
	}
}

func TestHasStringOverlap(t *testing.T) {
	if !HasStringOverlap([]string{"a", "b"}, []string{"b", "c"}) {
		t.Error("expected overlap on 'b'")
	}
	if HasStringOverlap([]string{"a", "b"}, []string{"c", "d"}) {
		t.Error("expected no overlap")
	}
	if HasStringOverlap(nil, []string{"a"}) {
		t.Error("expected no overlap with nil")
	}
	if HasStringOverlap([]string{"a"}, nil) {
		t.Error("expected no overlap with nil")
	}
}

func TestHasIntentOverlap(t *testing.T) {
	if !HasIntentOverlap([]Intent{IntentCharge, IntentAuthorize}, []Intent{IntentAuthorize, IntentSubscribe}) {
		t.Error("expected overlap on IntentAuthorize")
	}
	if HasIntentOverlap([]Intent{IntentCharge}, []Intent{IntentSubscribe}) {
		t.Error("expected no overlap")
	}
}

func TestHasCurrencyOverlap(t *testing.T) {
	if !HasCurrencyOverlap([]Currency{USD, EUR}, []Currency{EUR, GBP}) {
		t.Error("expected overlap on EUR")
	}
	if HasCurrencyOverlap([]Currency{USD}, []Currency{BRL}) {
		t.Error("expected no overlap")
	}
}
