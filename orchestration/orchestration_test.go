package orchestration

import (
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func makeOptions() []core.PaymentOption {
	return []core.PaymentOption{
		{Intent: core.IntentCharge, Method: "card", Currency: core.USD, Amount: "10.00"},
		{Intent: core.IntentCharge, Method: "upi", Currency: core.INR, Amount: "10.00"},
		{Intent: core.IntentCharge, Method: "pix", Currency: core.BRL, Amount: "10.00"},
	}
}

func TestCheapestStrategy(t *testing.T) {
	ft := NewFeeTable(map[string]FeeInfo{
		"card": {FixedFee: "0.30", PercentFee: 0.029, Currency: core.USD, SettlementTime: 2 * 24 * time.Hour},
		"upi":  {FixedFee: "0.00", PercentFee: 0.001, Currency: core.INR, SettlementTime: time.Hour},
		"pix":  {FixedFee: "0.00", PercentFee: 0.005, Currency: core.BRL, SettlementTime: 30 * time.Minute},
	})

	s := &CheapestStrategy{FeeTable: ft}
	result, err := s.Select(makeOptions(), SelectionContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "upi" {
		t.Errorf("expected upi (cheapest), got %s", result.Method)
	}
}

func TestFastestStrategy(t *testing.T) {
	ft := NewFeeTable(map[string]FeeInfo{
		"card": {SettlementTime: 2 * 24 * time.Hour},
		"upi":  {SettlementTime: time.Hour},
		"pix":  {SettlementTime: 10 * time.Second},
	})

	s := &FastestStrategy{FeeTable: ft}
	result, err := s.Select(makeOptions(), SelectionContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "pix" {
		t.Errorf("expected pix (fastest), got %s", result.Method)
	}
}

func TestPreferredStrategy(t *testing.T) {
	s := &PreferredStrategy{}
	ctx := SelectionContext{Preferences: []string{"pix", "upi"}}
	result, err := s.Select(makeOptions(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "pix" {
		t.Errorf("expected pix (first preference), got %s", result.Method)
	}
}

func TestPreferredStrategyFallback(t *testing.T) {
	s := &PreferredStrategy{}
	ctx := SelectionContext{Preferences: []string{"crypto"}} // not available
	result, err := s.Select(makeOptions(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to first option.
	if result.Method != "card" {
		t.Errorf("expected card (fallback), got %s", result.Method)
	}
}

func TestRegionStrategy(t *testing.T) {
	s := &RegionStrategy{
		RegionMethods: map[string][]string{
			"IN": {"upi"},
			"BR": {"pix"},
			"US": {"card"},
		},
	}

	result, err := s.Select(makeOptions(), SelectionContext{Region: "IN"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "upi" {
		t.Errorf("expected upi for India, got %s", result.Method)
	}

	result, err = s.Select(makeOptions(), SelectionContext{Region: "BR"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "pix" {
		t.Errorf("expected pix for Brazil, got %s", result.Method)
	}
}

func TestRegionStrategyUnknownRegion(t *testing.T) {
	s := &RegionStrategy{RegionMethods: map[string][]string{}}
	result, err := s.Select(makeOptions(), SelectionContext{Region: "XX"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "card" {
		t.Errorf("expected fallback to first, got %s", result.Method)
	}
}

func TestCompositeStrategy(t *testing.T) {
	ft := NewFeeTable(map[string]FeeInfo{
		"card": {FixedFee: "0.30", PercentFee: 0.029, Currency: core.USD, SettlementTime: 2 * 24 * time.Hour},
		"upi":  {FixedFee: "0.00", PercentFee: 0.001, Currency: core.INR, SettlementTime: time.Hour},
		"pix":  {FixedFee: "0.00", PercentFee: 0.005, Currency: core.BRL, SettlementTime: 30 * time.Minute},
	})

	// Preferred first, then cheapest as fallback.
	s := &CompositeStrategy{
		Strategies: []Strategy{
			&PreferredStrategy{},
			&CheapestStrategy{FeeTable: ft},
		},
	}

	// With a preference, preferred wins.
	ctx := SelectionContext{Preferences: []string{"card"}}
	result, err := s.Select(makeOptions(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "card" {
		t.Errorf("expected card (preferred), got %s", result.Method)
	}
}

func TestRoundRobinStrategy(t *testing.T) {
	s := &RoundRobinStrategy{}
	options := makeOptions()
	ctx := SelectionContext{}

	// Should cycle through options.
	methods := make([]string, 3)
	for i := 0; i < 3; i++ {
		result, err := s.Select(options, ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		methods[i] = result.Method
	}

	// All three methods should be selected exactly once.
	seen := make(map[string]bool)
	for _, m := range methods {
		seen[m] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 different methods in round robin, got %d: %v", len(seen), methods)
	}
}

func TestEmptyOptions(t *testing.T) {
	strategies := []Strategy{
		&CheapestStrategy{FeeTable: NewFeeTable(nil)},
		&FastestStrategy{FeeTable: NewFeeTable(nil)},
		&PreferredStrategy{},
		&RegionStrategy{RegionMethods: map[string][]string{}},
		&RoundRobinStrategy{},
	}

	for _, s := range strategies {
		_, err := s.Select(nil, SelectionContext{})
		if err == nil {
			t.Errorf("expected error for empty options with %T", s)
		}
	}
}

func TestOrchestratorSetStrategy(t *testing.T) {
	o := NewOrchestrator(&PreferredStrategy{})

	options := makeOptions()
	ctx := SelectionContext{Preferences: []string{"pix"}}
	result, err := o.Select(options, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "pix" {
		t.Errorf("expected pix, got %s", result.Method)
	}

	// Switch to region strategy.
	o.SetStrategy(&RegionStrategy{
		RegionMethods: map[string][]string{"US": {"card"}},
	})
	ctx2 := SelectionContext{Region: "US"}
	result, err = o.Select(options, ctx2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Method != "card" {
		t.Errorf("expected card for US region, got %s", result.Method)
	}
}
