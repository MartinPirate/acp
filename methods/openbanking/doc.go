// Package openbanking implements Open Banking (PSD2) payments via account
// information and payment initiation APIs.
//
// Open Banking enables direct bank-to-bank payments in the UK and EU
// without card networks. This method supports the charge and mandate
// intents in EUR and GBP, using providers such as TrueLayer or Plaid.
//
// # Key Types
//
//   - [Config] -- API key, provider name, and OAuth redirect URL.
//   - [OpenBankingMethod] -- the [core.Method] implementation.
//   - [Payload] -- Open Banking-specific payload containing consent ID,
//     payment ID, and provider name.
//
// # Usage
//
//	method, err := openbanking.New(openbanking.Config{
//	    APIKey:      "key_...",
//	    Provider:    "truelayer",
//	    RedirectURL: "https://app.example.com/callback",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package openbanking
