// Package acphttp provides the HTTP 402 transport binding for the Agentic
// Commerce Protocol.
//
// It implements both server-side middleware (paywall) and a client that
// automatically handles 402 Payment Required responses.
//
// # Headers
//
// Three custom headers carry ACP data as base64-encoded JSON:
//   - ACP-Payment-Required -- server to agent: payment options (402 response).
//   - ACP-Payment -- agent to server: selected option + method-specific payload.
//   - ACP-Payment-Response -- server to agent: settlement receipt (200 response).
//
// # Server Side
//
// Wrap any [http.Handler] with [Paywall] to require payment before access:
//
//	mux.Handle("/premium", acphttp.Paywall(gateway, acp.Price{
//	    Amount: "1.00", Currency: "USD",
//	}, handler))
//
// The middleware returns 402 with payment options when no ACP-Payment header is
// present. When a valid payment is submitted, it verifies and settles before
// serving the resource.
//
// # Client Side
//
// [Client] intercepts 402 responses and pays automatically:
//
//	client := acphttp.NewClient(gateway,
//	    acphttp.WithBudget(core.Budget{MaxPerSession: "20.00", Currency: "USD"}),
//	)
//	resp, err := client.Get("https://api.example.com/premium")
//
// The client selects a compatible payment method, creates the payload, and
// retries the request with the ACP-Payment header attached.
package acphttp
