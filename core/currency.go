package core

import (
	"fmt"
	"math/big"
	"strings"
)

// Currency represents an ISO 4217 currency code or crypto token symbol.
type Currency string

const (
	USD  Currency = "USD"
	EUR  Currency = "EUR"
	GBP  Currency = "GBP"
	JPY  Currency = "JPY"
	INR  Currency = "INR"
	BRL  Currency = "BRL"
	KES  Currency = "KES"
	NGN  Currency = "NGN"
	ZAR  Currency = "ZAR"
	CNY  Currency = "CNY"
	PHP  Currency = "PHP"
	THB  Currency = "THB"
	IDR  Currency = "IDR"
	MXN  Currency = "MXN"
	SEK  Currency = "SEK"
	NOK  Currency = "NOK"
	SAR  Currency = "SAR"
	EGP  Currency = "EGP"
	USDC Currency = "USDC"
)

// CurrencyInfo holds metadata about a currency.
type CurrencyInfo struct {
	Code       Currency
	Name       string
	MinorUnits int  // 2 for USD, 0 for JPY, 6 for USDC
	IsCrypto   bool
}

var knownCurrencies = map[Currency]CurrencyInfo{
	USD:  {USD, "US Dollar", 2, false},
	EUR:  {EUR, "Euro", 2, false},
	GBP:  {GBP, "British Pound", 2, false},
	JPY:  {JPY, "Japanese Yen", 0, false},
	INR:  {INR, "Indian Rupee", 2, false},
	BRL:  {BRL, "Brazilian Real", 2, false},
	KES:  {KES, "Kenyan Shilling", 2, false},
	NGN:  {NGN, "Nigerian Naira", 2, false},
	ZAR:  {ZAR, "South African Rand", 2, false},
	CNY:  {CNY, "Chinese Yuan", 2, false},
	PHP:  {PHP, "Philippine Peso", 2, false},
	THB:  {THB, "Thai Baht", 2, false},
	IDR:  {IDR, "Indonesian Rupiah", 2, false},
	MXN:  {MXN, "Mexican Peso", 2, false},
	SEK:  {SEK, "Swedish Krona", 2, false},
	NOK:  {NOK, "Norwegian Krone", 2, false},
	SAR:  {SAR, "Saudi Riyal", 2, false},
	EGP:  {EGP, "Egyptian Pound", 2, false},
	USDC: {USDC, "USD Coin", 6, true},
}

// LookupCurrency returns CurrencyInfo for a known currency.
func LookupCurrency(code Currency) (CurrencyInfo, bool) {
	info, ok := knownCurrencies[code]
	return info, ok
}

// ParseAmount validates that an amount string is a valid non-negative decimal.
func ParseAmount(amount string) error {
	if amount == "" {
		return fmt.Errorf("amount is empty")
	}
	amount = strings.TrimSpace(amount)
	r := new(big.Rat)
	if _, ok := r.SetString(amount); !ok {
		return fmt.Errorf("invalid amount: %q", amount)
	}
	if r.Sign() < 0 {
		return fmt.Errorf("amount must be non-negative: %q", amount)
	}
	return nil
}

// CompareAmounts compares two amount strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareAmounts(a, b string) (int, error) {
	ra := new(big.Rat)
	if _, ok := ra.SetString(a); !ok {
		return 0, fmt.Errorf("invalid amount: %q", a)
	}
	rb := new(big.Rat)
	if _, ok := rb.SetString(b); !ok {
		return 0, fmt.Errorf("invalid amount: %q", b)
	}
	return ra.Cmp(rb), nil
}
