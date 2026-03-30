// Package fx provides multi-currency FX rate lookups and amount conversion.
// It is a utility package, not a [core.Method] implementation.
//
// All rates are expressed relative to USD. The [RateProvider] interface
// allows plugging in live rate feeds; [StaticRates] provides an in-memory
// implementation seeded with approximate rates for 18+ currencies.
//
// # Key Types
//
//   - [RateProvider] -- interface returning the exchange rate between two
//     currencies as a [math/big.Rat].
//   - [StaticRates] -- thread-safe static rate table with USD as the base
//     currency.
//   - [ConvertAmount] -- converts a decimal amount string between currencies.
//   - [DefaultRates] -- returns a [StaticRates] with approximate rates for
//     common ACP currencies.
//
// # Usage
//
//	rates := fx.DefaultRates()
//	converted, err := fx.ConvertAmount(rates, "100.00", core.USD, core.EUR)
//	// converted = "92.000000"
//
//	// Or build a custom rate table:
//	custom, err := fx.NewStaticRates(map[core.Currency]string{
//	    core.EUR: "0.92",
//	    core.GBP: "0.79",
//	})
//	custom.SetRate(core.JPY, "149.50")
package fx
