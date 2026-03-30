// Package pix implements PIX instant payments for the Brazilian market.
//
// PIX is Brazil's instant payment system operated by the Central Bank.
// This method supports the charge intent in BRL, using a PIX key
// (CPF/CNPJ, email, phone, or random key) for collections.
//
// # Key Types
//
//   - [Config] -- API key, PIX key, and provider (e.g., "stripe", "pagseguro").
//   - [PixMethod] -- the [core.Method] implementation.
//   - [Payload] -- PIX-specific payload containing the PIX key, end-to-end
//     ID, and transaction ID.
//
// # Usage
//
//	method, err := pix.New(pix.Config{
//	    APIKey:   "key_...",
//	    PixKey:   "merchant@example.com",
//	    Provider: "stripe",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package pix
