// Package sepa implements SEPA (Single Euro Payments Area) Instant credit
// transfer payments for the European market.
//
// SEPA Instant enables real-time EUR bank transfers across 36 European
// countries. This method supports the charge intent in EUR, using IBAN
// and BIC for account identification.
//
// # Key Types
//
//   - [Config] -- API key, IBAN, BIC, and provider (e.g., "stripe", "adyen").
//   - [SepaMethod] -- the [core.Method] implementation.
//   - [Payload] -- SEPA-specific payload containing IBAN, BIC, payment
//     reference, and end-to-end ID.
//
// # Usage
//
//	method, err := sepa.New(sepa.Config{
//	    APIKey:   "key_...",
//	    IBAN:     "DE89370400440532013000",
//	    BIC:      "COBADEFFXXX",
//	    Provider: "stripe",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package sepa
