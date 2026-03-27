// Package fx provides multi-currency FX rate lookups and amount conversion.
// It is a utility package, not a Method implementation.
package fx

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/paideia-ai/acp/core"
)

// RateProvider is the interface for fetching exchange rates.
type RateProvider interface {
	// Rate returns the exchange rate from one currency to another.
	// The rate represents how many units of `to` equal one unit of `from`.
	Rate(from, to core.Currency) (*big.Rat, error)
}

// StaticRates is a RateProvider backed by a fixed rate table.
// All rates are expressed relative to USD (i.e., how many units of currency X per 1 USD).
type StaticRates struct {
	mu    sync.RWMutex
	rates map[core.Currency]*big.Rat // rate per 1 USD
}

// NewStaticRates creates a StaticRates provider with the given USD-based rates.
// The rates map should contain the value of 1 USD in each currency.
func NewStaticRates(rates map[core.Currency]string) (*StaticRates, error) {
	parsed := make(map[core.Currency]*big.Rat, len(rates))
	for cur, rateStr := range rates {
		r := new(big.Rat)
		if _, ok := r.SetString(rateStr); !ok {
			return nil, fmt.Errorf("fx: invalid rate for %s: %q", cur, rateStr)
		}
		if r.Sign() <= 0 {
			return nil, fmt.Errorf("fx: rate for %s must be positive", cur)
		}
		parsed[cur] = r
	}
	// USD to USD is always 1.
	parsed[core.USD] = new(big.Rat).SetInt64(1)

	return &StaticRates{rates: parsed}, nil
}

// SetRate updates or adds a rate for a currency (relative to USD).
func (s *StaticRates) SetRate(currency core.Currency, rate string) error {
	r := new(big.Rat)
	if _, ok := r.SetString(rate); !ok {
		return fmt.Errorf("fx: invalid rate: %q", rate)
	}
	if r.Sign() <= 0 {
		return fmt.Errorf("fx: rate must be positive")
	}
	s.mu.Lock()
	s.rates[currency] = r
	s.mu.Unlock()
	return nil
}

// Rate returns the conversion rate from `from` to `to`.
func (s *StaticRates) Rate(from, to core.Currency) (*big.Rat, error) {
	if from == to {
		return new(big.Rat).SetInt64(1), nil
	}

	s.mu.RLock()
	fromRate, fromOK := s.rates[from]
	toRate, toOK := s.rates[to]
	s.mu.RUnlock()

	if !fromOK {
		return nil, fmt.Errorf("fx: unknown currency %q", from)
	}
	if !toOK {
		return nil, fmt.Errorf("fx: unknown currency %q", to)
	}

	// rate = toRate / fromRate
	// e.g., if fromRate=1 (USD) and toRate=0.85 (EUR), then 1 USD = 0.85 EUR
	result := new(big.Rat).Quo(toRate, fromRate)
	return result, nil
}

// ConvertAmount converts an amount from one currency to another using the given provider.
func ConvertAmount(provider RateProvider, amount string, from, to core.Currency) (string, error) {
	if from == to {
		return amount, nil
	}

	amtRat := new(big.Rat)
	if _, ok := amtRat.SetString(amount); !ok {
		return "", fmt.Errorf("fx: invalid amount: %q", amount)
	}

	rate, err := provider.Rate(from, to)
	if err != nil {
		return "", err
	}

	result := new(big.Rat).Mul(amtRat, rate)
	return result.FloatString(6), nil
}

// DefaultRates returns a StaticRates provider with common exchange rates (approximate).
func DefaultRates() *StaticRates {
	rates, _ := NewStaticRates(map[core.Currency]string{
		core.EUR:  "0.92",
		core.GBP:  "0.79",
		core.JPY:  "149.50",
		core.INR:  "83.10",
		core.BRL:  "4.97",
		core.KES:  "153.50",
		core.NGN:  "1550.00",
		core.ZAR:  "18.90",
		core.CNY:  "7.24",
		core.PHP:  "56.20",
		core.THB:  "35.80",
		core.IDR:  "15680.00",
		core.MXN:  "17.15",
		core.SEK:  "10.45",
		core.NOK:  "10.55",
		core.SAR:  "3.75",
		core.EGP:  "30.90",
		core.USDC: "1.00",
	})
	return rates
}
