package fx

import (
	"math/big"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func TestNewStaticRates_Validation(t *testing.T) {
	_, err := NewStaticRates(map[core.Currency]string{
		core.EUR: "invalid",
	})
	if err == nil {
		t.Error("NewStaticRates() expected error for invalid rate")
	}

	_, err = NewStaticRates(map[core.Currency]string{
		core.EUR: "-1.0",
	})
	if err == nil {
		t.Error("NewStaticRates() expected error for negative rate")
	}

	_, err = NewStaticRates(map[core.Currency]string{
		core.EUR: "0.92",
	})
	if err != nil {
		t.Fatalf("NewStaticRates() error: %v", err)
	}
}

func TestStaticRates_SameCurrency(t *testing.T) {
	sr := DefaultRates()
	rate, err := sr.Rate(core.USD, core.USD)
	if err != nil {
		t.Fatalf("Rate() error: %v", err)
	}
	if rate.Cmp(new(big.Rat).SetInt64(1)) != 0 {
		t.Errorf("Rate(USD, USD) = %s, want 1", rate.FloatString(2))
	}
}

func TestStaticRates_CrossRate(t *testing.T) {
	sr := DefaultRates()

	// USD -> EUR should be approximately 0.92
	rate, err := sr.Rate(core.USD, core.EUR)
	if err != nil {
		t.Fatalf("Rate() error: %v", err)
	}
	rateFloat, _ := rate.Float64()
	if rateFloat < 0.9 || rateFloat > 0.95 {
		t.Errorf("Rate(USD, EUR) = %f, want ~0.92", rateFloat)
	}
}

func TestStaticRates_UnknownCurrency(t *testing.T) {
	sr, _ := NewStaticRates(map[core.Currency]string{
		core.EUR: "0.92",
	})
	_, err := sr.Rate(core.USD, core.Currency("XYZ"))
	if err == nil {
		t.Error("Rate() expected error for unknown currency")
	}
}

func TestConvertAmount(t *testing.T) {
	sr := DefaultRates()

	// Same currency
	result, err := ConvertAmount(sr, "100.00", core.USD, core.USD)
	if err != nil {
		t.Fatalf("ConvertAmount() error: %v", err)
	}
	if result != "100.00" {
		t.Errorf("ConvertAmount(USD->USD) = %q, want %q", result, "100.00")
	}

	// USD -> EUR
	result, err = ConvertAmount(sr, "100.00", core.USD, core.EUR)
	if err != nil {
		t.Fatalf("ConvertAmount() error: %v", err)
	}
	if result != "92.000000" {
		t.Errorf("ConvertAmount(USD->EUR) = %q, want %q", result, "92.000000")
	}

	// Invalid amount
	_, err = ConvertAmount(sr, "bad", core.USD, core.EUR)
	if err == nil {
		t.Error("ConvertAmount() expected error for invalid amount")
	}
}

func TestSetRate(t *testing.T) {
	sr := DefaultRates()

	err := sr.SetRate(core.EUR, "0.95")
	if err != nil {
		t.Fatalf("SetRate() error: %v", err)
	}

	rate, _ := sr.Rate(core.USD, core.EUR)
	rateFloat, _ := rate.Float64()
	if rateFloat < 0.94 || rateFloat > 0.96 {
		t.Errorf("Rate(USD, EUR) after SetRate = %f, want ~0.95", rateFloat)
	}

	// Invalid rate
	err = sr.SetRate(core.EUR, "bad")
	if err == nil {
		t.Error("SetRate() expected error for invalid rate")
	}
}

func TestDefaultRates(t *testing.T) {
	sr := DefaultRates()
	if sr == nil {
		t.Fatal("DefaultRates() returned nil")
	}

	// Verify a few known currencies exist
	for _, cur := range []core.Currency{core.EUR, core.GBP, core.JPY, core.INR, core.BRL, core.USDC} {
		_, err := sr.Rate(core.USD, cur)
		if err != nil {
			t.Errorf("Rate(USD, %s) error: %v", cur, err)
		}
	}
}
