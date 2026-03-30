// Package card implements card payments via Stripe's PaymentIntent API.
//
// It supports the charge and authorize intents across 15+ currencies
// including USD, EUR, GBP, JPY, INR, BRL, and more.
//
// # Key Types
//
//   - [Config] -- Stripe API key and webhook secret configuration.
//   - [CardMethod] -- the [core.Method] implementation.
//   - [Payload] -- card-specific payment payload containing a Stripe token
//     and PaymentIntent ID.
//
// # Usage
//
//	method, err := card.New(card.Config{
//	    APIKey:        "sk_live_...",
//	    WebhookSecret: "whsec_...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package card
