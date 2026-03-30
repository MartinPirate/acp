// Package acp implements the Agentic Commerce Protocol -- a universal middleware
// that enables AI agents to request and make payments through any payment
// provider worldwide.
//
// ACP uses HTTP 402 (Payment Required) as the signaling mechanism and supports
// any payment rail: cards, mobile money, bank transfers, real-time payment
// systems, crypto, and more.
//
// # Architecture
//
// ACP separates what kind of payment (Intents) from how it's executed (Methods).
// A Gateway coordinates registered Methods to build payment requirements,
// verify payments, and settle transactions.
//
// Key types re-exported from [github.com/paideia-ai/acp/core]:
//   - [Price] declares the amount and currency for a resource.
//   - [Method] is a concrete payment rail (card, UPI, PIX, M-Pesa, etc.).
//   - [Intent] is the kind of payment (charge, authorize, subscribe, mandate).
//   - [PaymentRequired] is the 402 response sent to the agent.
//   - [PaymentPayload] is the agent's payment submission.
//   - [Budget] enforces per-request and per-session spending limits.
//
// # Quick Start
//
// Server side -- protect an endpoint with a paywall:
//
//	cardMethod, _ := card.New(card.Config{APIKey: "sk_...", WebhookSecret: "whsec_..."})
//	gateway := acp.NewGateway(
//	    acp.WithMethod(cardMethod),
//	)
//	mux.Handle("/api/data", acphttp.Paywall(gateway, acp.Price{
//	    Amount: "5.99", Currency: "USD",
//	}, handler))
//
// Client side -- agent pays automatically:
//
//	client := acphttp.NewClient(gateway, acphttp.WithBudget(core.Budget{
//	    MaxPerRequest: "10.00",
//	    MaxPerSession: "50.00",
//	    Currency:      "USD",
//	}))
//	resp, err := client.Get("https://api.example.com/api/data")
package acp
