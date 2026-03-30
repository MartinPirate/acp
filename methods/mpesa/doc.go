// Package mpesa implements M-Pesa mobile money payments via the Safaricom
// Daraja API for the East African market.
//
// M-Pesa is Kenya's dominant mobile money platform. This method supports
// the charge intent in KES, using STK Push (Lipa Na M-Pesa Online) for
// customer-initiated payments.
//
// # Key Types
//
//   - [Config] -- Daraja consumer key/secret, short code, passkey, and
//     environment ("sandbox" or "production").
//   - [MpesaMethod] -- the [core.Method] implementation.
//   - [Payload] -- M-Pesa-specific payload containing phone number (254...),
//     account reference, transaction ID, and checkout request ID.
//
// # Usage
//
//	method, err := mpesa.New(mpesa.Config{
//	    ConsumerKey:    "key",
//	    ConsumerSecret: "secret",
//	    ShortCode:      "174379",
//	    PassKey:        "passkey",
//	    Environment:    "sandbox",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package mpesa
