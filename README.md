# ACP - Agentic Commerce Protocol

**A universal, open-source middleware that enables AI agents to request and make payments through any payment provider worldwide.**

ACP is a Go-based protocol and middleware layer that bridges AI agents with the global payments ecosystem. It uses HTTP 402 (`Payment Required`) as the signaling mechanism and supports any payment rail — cards, mobile money, bank transfers, real-time payment systems, crypto, and more.

> Think of it as "the HTTP of payments" — a single protocol that agents speak, with pluggable backends for every payment rail on earth.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-227%20passing-brightgreen.svg)](#testing)

---

## Quick Start

### Install

```bash
go get github.com/paideia-ai/acp
```

### Server — Paywall an endpoint in 3 lines

```go
gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))

mux.Handle("/api/data", acphttp.Paywall(gateway, acp.Price{
    Amount: "1.00", Currency: "USD",
}, handler))
```

### Client — Agent pays automatically

```go
client := acphttp.NewClient(gateway)
resp, err := client.Get("https://api.example.com/api/data")
// 402 is handled automatically — resp is 200 with your data
```

### Try it now

```bash
# Terminal 1: start the demo server
go run ./examples/demo

# Terminal 2: pay for an API call
go run ./cmd/acp-pay --url http://localhost:8080/api/data

# See the 402 response directly
curl -i http://localhost:8080/api/data

# Open the dashboard
open http://localhost:9090
```

---

## Why ACP Exists

The agent economy is here. AI agents book flights, purchase APIs, and orchestrate multi-step workflows that cost money. But today's agent payment protocols are fragmented and rail-locked:

| Protocol | Creator | Limitation |
|---|---|---|
| **x402** | Coinbase | Crypto-only (USDC on EVM/Solana) |
| **MPP** | Stripe/Tempo | Locked to Stripe + Tempo blockchain |
| **ACP (Stripe)** | Stripe/OpenAI | Human-present checkout only, Stripe-only |
| **AP2** | Google | Card-network focused, V0.1, no settlement layer |

Meanwhile, the world pays with:

- **UPI** — 14B+ transactions/month in India
- **PIX** — 4B+ transactions/month in Brazil
- **M-Pesa** — 37M+ monthly active users across Africa
- **SEPA Instant** — covering 36 European countries
- **FedNow** — real-time payments in the US
- **Alipay/WeChat Pay** — dominant in China

**No existing agent payment protocol supports any of these rails.** ACP fills this gap.

---

## How It Works

ACP uses the HTTP 402 status code to create a universal payment negotiation layer between agents and services. Two requests, three headers — that's the entire protocol.

```
Agent                        Service                      Facilitator
  |                             |                              |
  |-- GET /api/resource ------->|                              |
  |                             |                              |
  |<-- 402 Payment Required ----|                              |
  |    ACP-Payment-Required:    |                              |
  |    (methods, amounts)       |                              |
  |                             |                              |
  |-- GET /api/resource ------->|                              |
  |   ACP-Payment: (proof)     |------- /verify -------------->|
  |                             |<------ valid ----------------|
  |                             |------- /settle ------------->|
  |                             |<------ settled --------------|
  |                             |                              |
  |<-- 200 OK + resource -------|                              |
  |   ACP-Payment-Response:     |                              |
  |   (receipt)                 |                              |
```

### What goes over the wire

**Request 1** — Agent sends a normal HTTP request:
```http
GET /api/data HTTP/1.1
Host: api.example.com
```

**Response 1** — Server responds 402 with payment options:
```http
HTTP/1.1 402 Payment Required
ACP-Payment-Required: <base64-encoded JSON>
```
```json
{
  "acpVersion": 1,
  "resource": { "url": "/api/data" },
  "accepts": [
    { "intent": "charge", "method": "card",  "currency": "USD", "amount": "5.99" },
    { "intent": "charge", "method": "upi",   "currency": "INR", "amount": "500" },
    { "intent": "charge", "method": "mpesa", "currency": "KES", "amount": "770" },
    { "intent": "charge", "method": "pix",   "currency": "BRL", "amount": "30.00" },
    { "intent": "charge", "method": "x402",  "currency": "USDC", "amount": "5990000" }
  ]
}
```

**Request 2** — Agent picks a method and retries with payment proof:
```http
GET /api/data HTTP/1.1
ACP-Payment: <base64-encoded PaymentPayload>
```

**Response 2** — Server verifies, settles, and returns the resource:
```http
HTTP/1.1 200 OK
ACP-Payment-Response: <base64-encoded receipt>

{ "data": "your premium content" }
```

The `ACP-Payment` payload is **method-specific** — each rail puts different data in it:

| Method | Payload contains |
|---|---|
| `card` | `token`, `paymentIntentId` |
| `upi` | `vpa`, `transactionRef`, `upiTransactionId` |
| `mpesa` | `phoneNumber`, `checkoutRequestId`, `transactionId` |
| `pix` | `pixKey`, `e2eID`, `txID` |
| `x402` | `signature`, `authorization` (EIP-3009) |
| `sepa` | `iban`, `bic`, `reference`, `endToEndID` |
| `openbanking` | `consentID`, `paymentID`, `provider` |
| `alipay` | `tradeNo`, `outTradeNo`, `buyerID` |

---

## Architecture

ACP separates **what** kind of payment (Intents) from **how** it's executed (Methods). This is the key design decision that makes it rail-agnostic.

```
                    ┌─────────────────────────────────┐
                    │           ACP Core               │
                    │  ┌───────────┐  ┌─────────────┐ │
                    │  │  Intents  │  │  Discovery   │ │
                    │  │ ─────────  │  │             │ │
                    │  │ Charge    │  │ Method       │ │
                    │  │ Authorize │  │ Registry     │ │
                    │  │ Subscribe │  │              │ │
                    │  │ Mandate   │  │ Facilitator  │ │
                    │  │           │  │ Resolution   │ │
                    │  └───────────┘  └─────────────┘ │
                    └──────────┬──────────────────────┘
                               │
              ┌────────────────┼────────────────────┐
              │                │                    │
     ┌────────▼──────┐ ┌──────▼───────┐  ┌────────▼────────┐
     │  Transport    │ │   Methods    │  │  Facilitators   │
     │ ────────────  │ │ ──────────── │  │ ──────────────  │
     │ HTTP (402)    │ │ UPI          │  │ Self-hosted     │
     │ MCP           │ │ PIX          │  │ Hosted service  │
     │ A2A           │ │ M-Pesa       │  │ x402 bridge     │
     │ gRPC          │ │ SEPA         │  │                 │
     │               │ │ Cards/Stripe │  │                 │
     │               │ │ x402/Crypto  │  │                 │
     │               │ │ Alipay       │  │                 │
     │               │ │ Open Banking │  │                 │
     └───────────────┘ └──────────────┘  └─────────────────┘
```

### Intents

An Intent describes **what kind of payment** the service wants. Intents are independent of payment rails.

| Intent | Description | Use Case |
|---|---|---|
| `charge` | Fixed amount, one-time payment | API call, file download |
| `authorize` | Pre-authorize up to a limit | Session-based usage |
| `subscribe` | Recurring payment | Ongoing API access |
| `mandate` | Delegated spending authority | Autonomous agent budgets |

### Methods

A Method is a **concrete payment rail implementation**. Each method knows how to create payment payloads, verify them, and settle on its specific rail.

| Method | Rail | Region | Settlement | Currency |
|---|---|---|---|---|
| `card` | Visa/Mastercard/Amex | Global | 1-3 days | USD, EUR, GBP, +12 more |
| `upi` | Unified Payments Interface | India | Instant | INR |
| `pix` | PIX | Brazil | Instant | BRL |
| `mpesa` | M-Pesa (Daraja API) | East Africa | Instant | KES |
| `sepa` | SEPA Instant/Credit Transfer | Europe | Instant - 1 day | EUR |
| `x402` | x402 (EVM USDC) | Global | ~seconds | USDC |
| `alipay` | Alipay | China | Instant | CNY |
| `openbanking` | PSD2/Open Banking | Europe/UK | Instant - 1 day | EUR, GBP |
| `mock` | Testing only | Global | Instant | All |

New methods can be added without modifying the core protocol — just implement the `core.Method` interface.

### Facilitators

A Facilitator is a service that **verifies and settles payments** on behalf of the resource server:

```go
type Facilitator interface {
    Verify(ctx context.Context, payload PaymentPayload, requirements PaymentRequired) (*VerifyResponse, error)
    Settle(ctx context.Context, payload PaymentPayload, requirements PaymentRequired) (*SettleResponse, error)
    Supported(ctx context.Context) (*SupportedResponse, error)
}
```

### Transport Bindings

ACP is transport-agnostic. The protocol defines payment negotiation; transport bindings carry it over different channels.

| Transport | Mechanism | Package |
|---|---|---|
| **HTTP** | 402 status + headers | `transport/acphttp` |
| **MCP** | Tool result `_meta` | `transport/acpmcp` |
| **A2A** | Task metadata | `transport/acpa2a` |
| **gRPC** | Metadata + interceptors | `transport/acpgrpc` |

---

## Go SDK

### Server-Side: Protect an Endpoint

```go
package main

import (
    "net/http"
    "os"

    "github.com/paideia-ai/acp"
    "github.com/paideia-ai/acp/methods/card"
    "github.com/paideia-ai/acp/methods/mpesa"
    "github.com/paideia-ai/acp/methods/upi"
    "github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
    gateway := acp.NewGateway(
        acp.WithMethod(card.New(card.Config{
            APIKey:        os.Getenv("STRIPE_SECRET_KEY"),
            WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
        })),
        acp.WithMethod(upi.New(upi.Config{
            APIKey:      os.Getenv("RAZORPAY_KEY"),
            APISecret:   os.Getenv("RAZORPAY_SECRET"),
            MerchantVPA: "merchant@upi",
        })),
        acp.WithMethod(mpesa.New(mpesa.Config{
            ConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
            ConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
            ShortCode:      os.Getenv("MPESA_SHORTCODE"),
            PassKey:        os.Getenv("MPESA_PASSKEY"),
            Environment:    "sandbox",
        })),
    )

    mux := http.NewServeMux()
    mux.Handle("/api/data", acphttp.Paywall(gateway, acp.Price{
        Amount: "5.99", Currency: "USD",
    }, http.HandlerFunc(dataHandler)))

    http.ListenAndServe(":8080", mux)
}
```

### Client-Side: Agent Pays Automatically

```go
gateway := acp.NewGateway(
    acp.WithMethod(card.New(card.Config{
        APIKey:        os.Getenv("STRIPE_SECRET_KEY"),
        WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
    })),
)

client := acphttp.NewClient(gateway, acphttp.WithBudget(core.Budget{
    MaxPerRequest: "10.00",
    MaxPerSession: "100.00",
    Currency:      "USD",
}))

resp, err := client.Get("https://api.example.com/api/data")
// Agent receives 402, selects a method, pays, and gets the 200 — all automatic
```

### Framework Middleware

```go
// net/http (stdlib)
mux.Handle("/paid", acphttp.Paywall(gateway, price, handler))

// Chi
r.With(acpchi.Paywall(gateway, price)).Get("/paid", handler)

// Gin
r.GET("/paid", acpgin.Paywall(gateway, price), handler)

// Echo
e.GET("/paid", handler, acpecho.Paywall(gateway, price))
```

### Mandates: Pre-Authorized Agent Spending

```go
m := mandate.NewMandate(
    mandate.WithAgentID("agent-xyz"),
    mandate.WithMaxAmount("500.00"),
    mandate.WithCurrency(core.USD),
    mandate.WithMaxPerRequest("50.00"),
    mandate.WithAllowedMethods("card", "upi"),
    mandate.WithScope("/api/*"),
    mandate.WithExpiry(time.Now().Add(24 * time.Hour)),
)

store := mandate.NewMemoryStore()
store.Save(m)

enforcer := mandate.NewEnforcer()
err := enforcer.Check(m, payload, "/api/data")
```

### Audit Trail

```go
auditLogger := audit.NewMemoryLogger()
auditedGW := audit.NewAuditedGateway(gateway, auditLogger)
// Use auditedGW instead of gateway — all verify/settle calls are logged
```

### Rate Limiting & Anomaly Detection

```go
limiter := ratelimit.NewTokenBucketLimiter(ratelimit.TokenBucketConfig{
    Rate: 10, Burst: 20,
})

detector := ratelimit.NewAnomalyDetector()
result := detector.Check("agent-id", "100.00", core.USD, "card")
// result.IsAnomaly, result.RiskScore, result.Reasons
```

### Payment Orchestration

```go
// Automatically select the cheapest payment rail
orch := orchestration.NewOrchestrator(&orchestration.CheapestStrategy{
    FeeTable: orchestration.NewFeeTable(map[string]orchestration.FeeInfo{
        "card":  {FixedFee: "0.30", PercentFee: 0.029, Currency: core.USD},
        "upi":   {FixedFee: "0.00", PercentFee: 0.005, Currency: core.INR},
        "mpesa": {FixedFee: "0.00", PercentFee: 0.01, Currency: core.KES},
    }),
})
```

### Agent Authentication (JWT)

```go
issuer := auth.NewTokenIssuer(signingKey, "my-app", "my-app")
token, _ := issuer.Issue("agent-id", "user-id", []auth.Permission{
    {Resource: "/api/*", Methods: []string{"card"}, MaxAmount: "100.00", Currency: "USD"},
}, 24 * time.Hour)

// Middleware validates tokens on incoming requests
validator := auth.NewJWTValidator(auth.JWTConfig{
    SigningKey: signingKey, Issuer: "my-app", Audience: "my-app",
})
mux.Handle("/api/", auth.AuthMiddleware(validator)(apiHandler))
```

---

## API Documentation

ACP ships with an embedded OpenAPI 3.1 specification and Swagger UI.

### Facilitator API

Start the reference facilitator and browse the docs:

```bash
go run ./cmd/acp-facilitator
open http://localhost:8181/api/docs/
```

| Endpoint | Method | Purpose |
|---|---|---|
| `POST /verify` | Verify a payment without executing | Returns `{ valid, reason, payer }` |
| `POST /settle` | Execute payment on the rail | Returns `SettleResponse` |
| `GET /supported` | Declare capabilities | Returns supported methods, intents, currencies |
| `GET /health` | Health check | Returns `{ status: "ok" }` |
| `GET /api/docs/` | Swagger UI | Interactive API documentation |
| `GET /api/openapi.yaml` | OpenAPI spec | Raw YAML specification |

### Dashboard API

```bash
go run ./examples/demo
open http://localhost:9090        # Dashboard UI
open http://localhost:9090/docs/  # Swagger UI
```

| Endpoint | Purpose |
|---|---|
| `GET /` | Dashboard HTML page |
| `GET /api/stats` | Aggregate statistics (volume, success rate, by method) |
| `GET /api/transactions` | Transaction list with pagination (`?limit=&offset=`) |
| `GET /api/methods` | Registered methods and status |
| `GET /api/health` | Health check |

### Service Discovery

```bash
curl http://localhost:8080/.well-known/acp-services
```

Returns all active payment services registered in the discovery registry.

---

## Protocol Specification

The full formal specification is at [`spec/acp-spec-v1.md`](spec/acp-spec-v1.md) (IETF-style).

### Wire Format Summary

**PaymentRequired** (402 response, `ACP-Payment-Required` header):
```json
{
  "acpVersion": 1,
  "resource": { "url": "string", "description": "string", "mimeType": "string" },
  "accepts": [{ "intent": "charge", "method": "string", "currency": "string", "amount": "string" }],
  "extensions": {}
}
```

**PaymentPayload** (retry request, `ACP-Payment` header):
```json
{
  "acpVersion": 1,
  "resource": { "url": "string" },
  "accepted": { "intent": "string", "method": "string", "currency": "string", "amount": "string" },
  "payload": { },
  "extensions": {}
}
```

**SettleResponse** (200 response, `ACP-Payment-Response` header):
```json
{
  "acpVersion": 1,
  "success": true,
  "method": "string",
  "transaction": "string",
  "settledAt": "2026-03-27T18:28:06Z",
  "receipt": { }
}
```

### Error Codes

| Code | Meaning |
|---|---|
| `insufficient_funds` | Payer cannot cover the amount |
| `method_unavailable` | Method not supported by facilitator |
| `mandate_exceeded` | Payment exceeds mandate limits |
| `mandate_expired` | Mandate has expired |
| `currency_mismatch` | Currency not supported for this method |
| `amount_too_high` | Amount exceeds method/facilitator limits |
| `verification_failed` | Payment proof could not be verified |
| `settlement_failed` | Payment execution failed on the rail |
| `timeout` | Payment or settlement timed out |
| `invalid_payload` | Malformed payment payload |
| `unsupported_intent` | Method does not support this intent |
| `budget_exceeded` | Client budget limit reached |

---

## Testing

227 tests across 27 packages, all passing.

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific package tests
go test ./methods/card/ -v
go test ./transport/acphttp/ -v
go test ./core/ -v
```

### End-to-End Demo

The demo wires up all subsystems: gateway, audit, rate limiting, anomaly detection, mandates, discovery, orchestration, dashboard, and JWT auth.

```bash
go run ./examples/demo
```

Then in another terminal:

```bash
# Free endpoint
curl http://localhost:8080/api/health

# See 402 response
curl -i http://localhost:8080/api/data

# Auto-pay $1.00
go run ./cmd/acp-pay --url http://localhost:8080/api/data

# Auto-pay $9.99
go run ./cmd/acp-pay --url http://localhost:8080/api/premium

# Multi-currency (EUR)
go run ./cmd/acp-pay --url http://localhost:8080/api/eu-data --currency EUR

# Budget enforcement (fails — $9.99 exceeds $5.00 cap)
go run ./cmd/acp-pay --url http://localhost:8080/api/premium --max-spend 5.00

# Dashboard stats
curl http://localhost:9090/api/stats

# Service discovery
curl http://localhost:8080/.well-known/acp-services
```

### Docker

```bash
docker build -t acp-facilitator .
docker run -p 8181:8181 acp-facilitator
```

---

## Comparison with Existing Protocols

| Feature | ACP | x402 | MPP | AP2 |
|---|---|---|---|---|
| **Payment rails** | Any (cards, mobile money, A2A, crypto) | Crypto only (USDC) | Stripe + Tempo | Cards/bank |
| **Fiat support** | Native | No | Via Stripe | Yes |
| **Mobile money** | Native (M-Pesa, GCash, etc.) | No | No | No |
| **Real-time payments** | Native (UPI, PIX) | No | No | No |
| **Open banking** | Native (PSD2, A2A) | No | No | No |
| **Crypto** | Via x402 bridge | Native | Via Tempo | No |
| **Agent autonomy** | Full (mandates + budgets) | Full | Full (sessions) | Partial |
| **Open source** | Yes (Apache 2.0) | Yes (Apache 2.0) | Spec open | Spec open |
| **Transport** | HTTP, MCP, A2A, gRPC | HTTP, MCP, A2A | HTTP | A2A, MCP |

---

## Project Structure

```
acp/
  core/                       # Protocol types, interfaces, errors, utilities
    types.go                  # PaymentRequired, PaymentPayload, SettleResponse
    method.go                 # Method interface
    gateway.go                # GatewayInterface (shared by audit/ratelimit wrappers)
    facilitator.go            # Facilitator interface
    intent.go                 # Charge, Authorize, Subscribe, Mandate
    currency.go               # ISO 4217 + crypto token handling
    errors.go                 # Typed error codes
    budget.go                 # Budget enforcement
    methodutil.go             # Shared utilities (payload, settle, validation)
    slices.go                 # Shared slice helpers
    idempotency.go            # Idempotency key generation

  methods/                    # Payment rail implementations
    card/                     # Visa/Mastercard via Stripe
    upi/                      # UPI via Razorpay (India)
    pix/                      # PIX (Brazil)
    mpesa/                    # M-Pesa via Daraja (Kenya)
    sepa/                     # SEPA Instant (Europe)
    x402/                     # x402 USDC bridge (EVM)
    alipay/                   # Alipay (China)
    openbanking/              # Open Banking / PSD2 (UK/EU)
    fx/                       # FX rate conversion utility
    mock/                     # Mock method for testing

  transport/                  # Transport bindings
    acphttp/                  # HTTP 402 middleware + auto-paying client
    acpmcp/                   # MCP tool payment handler
    acpa2a/                   # A2A task payment metadata
    acpgrpc/                  # gRPC interceptors

  middleware/                 # Framework-specific adapters
    acpchi/                   # Chi middleware
    acpgin/                   # Gin middleware
    acpecho/                  # Echo middleware

  mandate/                    # Mandate specification and enforcement
  audit/                      # Audit trail and receipt logging
  auth/                       # OAuth 2.0 JWT agent tokens
  ratelimit/                  # Rate limiting + anomaly detection
  discovery/                  # Service discovery + health checking
  orchestration/              # Smart payment rail selection (6 strategies)
  dashboard/                  # Web monitoring dashboard + REST API

  api/                        # OpenAPI spec + Swagger UI
    openapi.yaml              # OpenAPI 3.1 specification
    swagger-ui/               # Embedded Swagger UI

  spec/                       # Formal protocol specification
    acp-spec-v1.md            # IETF-style protocol spec

  cmd/
    acp-pay/                  # CLI tool for testing payments
    acp-facilitator/          # Reference facilitator server

  examples/
    server/                   # Simple example server
    demo/                     # Full-stack demo (all subsystems)

  Dockerfile                  # Multi-stage build for facilitator
  docker-compose.yml          # Facilitator + optional Redis
```

---

## Design Principles

1. **Rail-agnostic** — The protocol never assumes a specific payment rail. New rails are added as `Method` implementations without protocol changes.

2. **Intent over implementation** — Services declare what they need (a charge of $5), not how to get it (a Stripe PaymentIntent). The agent and facilitator negotiate the how.

3. **Progressive complexity** — One line to paywall an endpoint. Add mandates, budgets, orchestration, audit, and auth when you need them.

4. **Interoperable** — Bridges to x402, MPP, and AP2 so ACP works within the existing ecosystem, not against it.

5. **Global by default** — Currency is ISO 4217. Methods span every major payment rail. No region or provider is privileged.

6. **Agent-first, human-compatible** — Designed for machine-to-machine payments, but supports human-in-the-loop confirmation when required (e.g., STK Push, 3DS).

7. **Trust-minimizing** — Facilitators cannot move funds beyond what the payment payload authorizes. Mandates are scoped and time-limited.

---

## Contributing

We're looking for contributors, especially those with experience in:

- Payment provider integrations (Stripe, Razorpay, Daraja, PagSeguro, Adyen, etc.)
- Protocol design and specification writing
- Go middleware and SDK development
- AI agent frameworks (LangChain, AutoGen, CrewAI, etc.)
- MCP and A2A protocol implementations

---

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

---

## Acknowledgments

ACP builds on ideas from:

- [x402](https://x402.org) by Coinbase — pioneered HTTP 402 for machine payments
- [MPP](https://mpp.dev) by Stripe/Tempo — introduced Intent/Method separation
- [AP2](https://github.com/google-agentic-commerce/AP2) by Google — Verifiable Digital Credentials for agent authorization
- [Interledger Protocol](https://interledger.org) — network-agnostic value transfer

ACP is not affiliated with any of these projects.
