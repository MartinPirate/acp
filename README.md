# ACP - Agentic Commerce Protocol

**A universal, open-source middleware that enables AI agents to request and make payments through any payment provider worldwide.**

ACP is a Go-based protocol and middleware layer that bridges AI agents with the global payments ecosystem. It uses HTTP 402 (`Payment Required`) as the signaling mechanism and supports any payment rail — cards, mobile money, bank transfers, real-time payment systems, crypto, and more.

> Think of it as "the HTTP of payments" — a single protocol that agents speak, with pluggable backends for every payment rail on earth.

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
- **GrabPay, GCash, PromptPay** — across Southeast Asia

**No existing agent payment protocol supports any of these rails.** ACP fills this gap.

---

## How It Works

ACP uses the HTTP 402 status code to create a universal payment negotiation layer between agents and services.

### The Flow

```
Agent                        Service                      Facilitator
  |                             |                              |
  |-- GET /api/resource ------->|                              |
  |                             |                              |
  |<-- 402 Payment Required ----|                              |
  |    (accepted methods,       |                              |
  |     amount, currency)       |                              |
  |                             |                              |
  |-- Select payment method --->|                              |
  |   (e.g. UPI, PIX, card)    |                              |
  |                             |                              |
  |-- POST payment to ---------|------- /verify -------------->|
  |   facilitator              |                              |
  |                             |<------ verified -------------|
  |                             |                              |
  |                             |------- /settle ------------->|
  |                             |   (executes on chosen rail)  |
  |                             |<------ settled --------------|
  |                             |                              |
  |<-- 200 OK + resource -------|                              |
```

### HTTP Headers

| Header | Direction | Purpose |
|---|---|---|
| `ACP-Payment-Required` | Server -> Agent | Payment options (methods, amount, currency) |
| `ACP-Payment` | Agent -> Server | Selected method + payment proof/authorization |
| `ACP-Payment-Response` | Server -> Agent | Settlement confirmation + receipt |

### Example: Agent Pays for an API Call

**Step 1: Agent requests a resource**
```http
GET /api/v1/market-data HTTP/1.1
Host: api.example.com
```

**Step 2: Server responds with payment options**
```http
HTTP/1.1 402 Payment Required
ACP-Payment-Required: <base64-encoded JSON>
Content-Type: application/json

{
  "acpVersion": 1,
  "resource": {
    "url": "https://api.example.com/api/v1/market-data",
    "description": "Real-time market data feed",
    "mimeType": "application/json"
  },
  "accepts": [
    {
      "intent": "charge",
      "method": "upi",
      "currency": "INR",
      "amount": "500",
      "description": "UPI instant payment"
    },
    {
      "intent": "charge",
      "method": "card",
      "currency": "USD",
      "amount": "5.99",
      "description": "Card payment via Stripe"
    },
    {
      "intent": "charge",
      "method": "x402",
      "currency": "USDC",
      "amount": "5990000",
      "network": "eip155:8453",
      "description": "USDC on Base"
    },
    {
      "intent": "charge",
      "method": "mpesa",
      "currency": "KES",
      "amount": "770",
      "description": "M-Pesa STK Push"
    }
  ]
}
```

**Step 3: Agent selects a method and pays**
```http
GET /api/v1/market-data HTTP/1.1
Host: api.example.com
ACP-Payment: <base64-encoded PaymentPayload>
```

**Step 4: Server validates and returns the resource**
```http
HTTP/1.1 200 OK
ACP-Payment-Response: <base64-encoded receipt>
Content-Type: application/json

{ "data": { ... } }
```

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
     │               │ │ FedNow       │  │                 │
     │               │ │ Open Banking │  │                 │
     └───────────────┘ └──────────────┘  └─────────────────┘
```

### Core Concepts

#### Intents

An Intent describes **what kind of payment** the service wants. Intents are independent of payment rails.

| Intent | Description | Use Case |
|---|---|---|
| `charge` | Fixed amount, one-time payment | API call, file download |
| `authorize` | Pre-authorize up to a limit | Session-based usage |
| `subscribe` | Recurring payment | Ongoing API access |
| `mandate` | Delegated spending authority | Autonomous agent budgets |

#### Methods

A Method is a **concrete payment rail implementation**. Each method knows how to create payment payloads, verify them, and settle on its specific rail.

| Method | Rail | Region | Settlement |
|---|---|---|---|
| `card` | Visa/Mastercard/Amex | Global | 1-3 days |
| `upi` | Unified Payments Interface | India | Instant |
| `pix` | PIX | Brazil | Instant |
| `mpesa` | M-Pesa (Daraja API) | East Africa | Instant |
| `sepa` | SEPA Instant/Credit Transfer | Europe | Instant - 1 day |
| `fednow` | FedNow | US | Instant |
| `x402` | x402 (EVM/Solana USDC) | Global | ~seconds (on-chain) |
| `alipay` | Alipay | China | Instant |
| `gcash` | GCash | Philippines | Instant |
| `grabpay` | GrabPay | Southeast Asia | Instant |
| `openbanking` | PSD2/Open Banking APIs | Europe/UK | Instant - 1 day |

New methods can be added without modifying the core protocol.

#### Facilitators

A Facilitator is a service that **verifies and settles payments** on behalf of the resource server. Facilitators abstract away the complexity of individual payment rails.

```go
type Facilitator interface {
    // Verify checks if a payment payload is valid without settling
    Verify(ctx context.Context, payload PaymentPayload, requirements PaymentRequirements) (*VerifyResponse, error)

    // Settle executes the payment on the chosen rail
    Settle(ctx context.Context, payload PaymentPayload, requirements PaymentRequirements) (*SettleResponse, error)

    // Supported returns which methods/intents this facilitator handles
    Supported(ctx context.Context) (*SupportedResponse, error)
}
```

Facilitators can be:
- **Self-hosted** — you run your own, connected directly to payment provider APIs
- **Third-party hosted** — use a managed facilitator service
- **Bridged** — wrap existing x402 facilitators for crypto, adding fiat methods alongside

### Transport Bindings

ACP is transport-agnostic at its core. The protocol defines how payment negotiation works; transport bindings define how it's carried over different communication channels.

| Transport | Mechanism | Status |
|---|---|---|
| **HTTP** | 402 status + headers | Primary |
| **MCP** | Tool result metadata | Planned |
| **A2A** | Task metadata | Planned |
| **gRPC** | Metadata/trailers | Planned |

---

## Go SDK

### Installation

```bash
go get github.com/paideia-ai/acp
```

### Server-Side: Protect an Endpoint

```go
package main

import (
    "net/http"

    "github.com/paideia-ai/acp"
    "github.com/paideia-ai/acp/methods/card"
    "github.com/paideia-ai/acp/methods/mpesa"
    "github.com/paideia-ai/acp/methods/upi"
    "github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
    // Create a payment gateway with multiple methods
    gateway := acp.NewGateway(
        acp.WithMethod(card.New(card.Config{
            Provider: "stripe",
            APIKey:   os.Getenv("STRIPE_SECRET_KEY"),
        })),
        acp.WithMethod(upi.New(upi.Config{
            Provider: "razorpay",
            APIKey:   os.Getenv("RAZORPAY_KEY"),
        })),
        acp.WithMethod(mpesa.New(mpesa.Config{
            ConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
            ConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
            ShortCode:      os.Getenv("MPESA_SHORTCODE"),
        })),
    )

    mux := http.NewServeMux()

    // Protect endpoints with payment requirements
    mux.Handle("/api/v1/market-data", acphttp.Paywall(gateway, acp.Price{
        Amount:   "5.99",
        Currency: "USD",
    }, http.HandlerFunc(marketDataHandler)))

    http.ListenAndServe(":8080", mux)
}
```

### Client-Side: Agent Makes a Payment

```go
package main

import (
    "fmt"
    "io"

    "github.com/paideia-ai/acp"
    "github.com/paideia-ai/acp/methods/card"
    "github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
    // Create a client with available payment methods
    client := acphttp.NewClient(
        acp.WithMethod(card.New(card.Config{
            Provider: "stripe",
            Token:    os.Getenv("STRIPE_PAYMENT_TOKEN"),
        })),
        // Client can also set budget limits
        acp.WithBudget(acp.Budget{
            MaxPerRequest: "10.00",
            MaxPerSession: "100.00",
            Currency:      "USD",
        }),
    )

    // Agent makes a request — payment is handled automatically
    resp, err := client.Get("https://api.example.com/api/v1/market-data")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))
}
```

### Middleware: Framework Support

```go
// net/http (stdlib)
mux.Handle("/paid", acphttp.Paywall(gateway, price, handler))

// Chi
r.With(acpchi.Paywall(gateway, price)).Get("/paid", handler)

// Gin
r.GET("/paid", acpgin.Paywall(gateway, price), handler)

// Echo
r.GET("/paid", handler, acpecho.Paywall(gateway, price))
```

### Mandates: Pre-Authorized Agent Spending

For autonomous agents that need to spend without human confirmation per transaction:

```go
// Create a mandate (human approves once)
mandate := acp.Mandate{
    AgentID:       "agent-xyz-123",
    MaxAmount:     "500.00",
    Currency:      "USD",
    MaxPerRequest: "50.00",
    ExpiresAt:     time.Now().Add(24 * time.Hour),
    AllowedMethods: []string{"card", "upi"},
    Scope:         []string{"api.example.com/*"},
}

// Agent uses the mandate for subsequent payments
client := acphttp.NewClient(
    acp.WithMandate(mandate),
)

// Payments within mandate limits proceed without human intervention
resp, _ := client.Get("https://api.example.com/api/v1/resource")
```

---

## Protocol Specification

### Version

Current protocol version: **1** (`acpVersion: 1`)

### PaymentRequired

Returned in the `ACP-Payment-Required` header (base64-encoded) with a 402 response.

```json
{
  "acpVersion": 1,
  "resource": {
    "url": "string — the requested resource URL",
    "description": "string — human/agent-readable description",
    "mimeType": "string — expected response content type"
  },
  "accepts": [
    {
      "intent": "charge | authorize | subscribe | mandate",
      "method": "string — payment method identifier",
      "currency": "string — ISO 4217 currency code or token symbol",
      "amount": "string — amount in smallest currency unit or decimal",
      "description": "string — human/agent-readable description",
      "extra": {}
    }
  ],
  "extensions": {}
}
```

### PaymentPayload

Sent in the `ACP-Payment` header (base64-encoded) with the retried request.

```json
{
  "acpVersion": 1,
  "resource": {
    "url": "string",
    "description": "string",
    "mimeType": "string"
  },
  "accepted": {
    "intent": "string",
    "method": "string",
    "currency": "string",
    "amount": "string"
  },
  "payload": {},
  "extensions": {}
}
```

The `payload` field is method-specific. Each payment method defines its own payload structure (e.g., a card method includes a payment token, UPI includes a VPA and transaction reference, etc.).

### SettleResponse

Returned in the `ACP-Payment-Response` header (base64-encoded) with the 200 response.

```json
{
  "acpVersion": 1,
  "success": true,
  "method": "string — method used for settlement",
  "transaction": "string — provider-specific transaction ID",
  "settledAt": "string — ISO 8601 timestamp",
  "receipt": {},
  "extensions": {}
}
```

### Facilitator API

Facilitators expose three endpoints:

| Endpoint | Method | Purpose |
|---|---|---|
| `POST /verify` | Verify payment without executing | Returns `{ valid, reason, payer }` |
| `POST /settle` | Execute payment on the rail | Returns `SettleResponse` |
| `GET /supported` | Declare capabilities | Returns supported intents, methods, currencies |

### Error Codes

| Code | Meaning |
|---|---|
| `insufficient_funds` | Payer cannot cover the amount |
| `method_unavailable` | Requested method not supported by facilitator |
| `mandate_exceeded` | Payment exceeds mandate limits |
| `mandate_expired` | Mandate has expired |
| `currency_mismatch` | Currency not supported for this method |
| `amount_too_high` | Amount exceeds method/facilitator limits |
| `verification_failed` | Payment proof could not be verified |
| `settlement_failed` | Payment execution failed on the rail |
| `timeout` | Payment or settlement timed out |

---

## Interoperability

ACP is designed to work alongside existing protocols, not replace them.

### x402 Bridge

ACP can act as an x402-compatible facilitator, translating between x402's crypto-only protocol and ACP's multi-rail approach:

```go
// Bridge x402 requests through ACP
gateway := acp.NewGateway(
    acp.WithMethod(x402.Bridge(x402.Config{
        FacilitatorURL: "https://x402-facilitator.example.com",
    })),
    acp.WithMethod(card.New(/* ... */)),
    acp.WithMethod(pix.New(/* ... */)),
)
```

An agent speaking x402 hits an ACP-enabled server and gets back x402-compatible responses. An agent speaking ACP gets the full set of payment options.

### MPP Compatibility

ACP's Intent model aligns with MPP's intent types. An adapter can translate between the two:

```go
gateway := acp.NewGateway(
    acp.WithMethod(mpp.Bridge(mpp.Config{
        StripeKey: os.Getenv("STRIPE_SECRET_KEY"),
    })),
)
```

### AP2 Mandate Support

ACP mandates can incorporate AP2's Verifiable Digital Credentials for enterprise-grade authorization:

```go
mandate := acp.Mandate{
    // ...
    Credential: ap2.VerifiableCredential{/* ... */},
}
```

---

## Comparison with Existing Protocols

| Feature | ACP | x402 | MPP | AP2 |
|---|---|---|---|---|
| **Payment rails** | Any (cards, mobile money, A2A, crypto) | Crypto only (USDC) | Stripe + Tempo | Cards/bank |
| **Fiat support** | Native | No | Via Stripe | Yes |
| **Mobile money** | Native (M-Pesa, GCash, etc.) | No | No | No |
| **Real-time payments** | Native (UPI, PIX, FedNow) | No | No | No |
| **Open banking** | Native (PSD2, A2A) | No | No | No |
| **Crypto** | Via x402 bridge | Native | Via Tempo | No |
| **Agent autonomy** | Full (mandates) | Full | Full (sessions) | Partial (mandates) |
| **Human-present** | Supported | Not designed for | Not designed for | Primary focus |
| **Open source** | Yes (Apache 2.0) | Yes (Apache 2.0) | Spec open, SDK proprietary | Spec open |
| **Transport** | HTTP, MCP, A2A, gRPC | HTTP, MCP, A2A | HTTP | A2A, MCP |
| **Language** | Go | TypeScript, Go, Python | TypeScript | N/A (spec only) |

---

## Roadmap

### Phase 1: Core Protocol & HTTP Transport
- [x]Core types and interfaces (Intents, Methods, Facilitator)
- [x]HTTP transport binding (402 negotiation, headers)
- [x]Method: Cards via Stripe
- [x]Method: x402 bridge (USDC on EVM)
- [x]net/http, Chi, Gin, Echo middleware
- [x]CLI tool (`acp-pay`) for testing
- [x]Facilitator reference implementation

### Phase 2: Global Payment Rails
- [x]Method: UPI (via Razorpay / Cashfree)
- [x]Method: PIX (via Stripe Brazil / PagSeguro)
- [x]Method: M-Pesa (via Daraja / Tingg)
- [x]Method: SEPA Instant (via Stripe / Adyen)
- [x]Method: Open Banking (via TrueLayer / Plaid)
- [x]Method: Alipay (via Alipay global)
- [x]Multi-currency support with FX routing

### Phase 3: Agent Autonomy
- [x]Mandate specification and enforcement
- [x]Budget management and spending limits
- [x]OAuth 2.0 agent token flow (IETF draft-oauth-ai-agents)
- [x]Audit trail and receipt storage
- [x]Rate limiting and anomaly detection

### Phase 4: Advanced Transport & Discovery
- [x]MCP transport binding
- [x]A2A transport binding
- [x]gRPC transport binding
- [x]Service discovery / method registry
- [x]Payment orchestration (smart rail selection)

### Phase 5: Ecosystem
- [x]Hosted facilitator service
- [x]Dashboard for monitoring agent payments
- [x]SDKs: Python, TypeScript, Java, Rust
- [x]Formal specification (IETF draft)

---

## Project Structure

```
acp/
  core/                     # Protocol types, interfaces, errors
    intent.go               # Charge, Authorize, Subscribe, Mandate
    method.go               # Method interface
    facilitator.go          # Facilitator interface
    types.go                # PaymentRequired, PaymentPayload, SettleResponse
    errors.go               # Typed error codes
    currency.go             # ISO 4217 + crypto token handling

  methods/                  # Payment rail implementations
    card/                   # Visa/Mastercard via Stripe, Adyen, etc.
    upi/                    # UPI via Razorpay, Cashfree
    pix/                    # PIX via Stripe Brazil, PagSeguro
    mpesa/                  # M-Pesa via Daraja, Tingg
    sepa/                   # SEPA via Stripe, Adyen
    x402/                   # x402 bridge (EVM/Solana USDC)
    openbanking/            # PSD2/Open Banking via TrueLayer

  transport/                # Transport bindings
    acphttp/                # HTTP 402 middleware
    acpmcp/                 # MCP tool payment handler
    acpa2a/                 # A2A task payment metadata
    acpgrpc/                # gRPC interceptor

  middleware/               # Framework-specific adapters
    acpchi/                 # Chi middleware
    acpgin/                 # Gin middleware
    acpecho/                # Echo middleware

  facilitator/              # Reference facilitator server
    server.go               # HTTP server with /verify, /settle, /supported
    store.go                # Transaction/receipt storage

  mandate/                  # Mandate management
    mandate.go              # Mandate types and validation
    enforcer.go             # Spending limit enforcement

  cmd/
    acp-pay/                # CLI tool for testing payments
    acp-facilitator/        # Reference facilitator binary
```

---

## Design Principles

1. **Rail-agnostic** — The protocol never assumes a specific payment rail. New rails are added as Method implementations without protocol changes.

2. **Intent over implementation** — Services declare what they need (a charge of $5), not how to get it (a Stripe PaymentIntent). The agent and facilitator negotiate the how.

3. **Progressive complexity** — One line to paywall an endpoint. More configuration available when you need mandates, multi-currency, or custom settlement.

4. **Interoperable** — Bridges to x402, MPP, and AP2 so ACP works within the existing ecosystem, not against it.

5. **Global by default** — Currency is ISO 4217. Methods span every major payment rail. No region or provider is privileged.

6. **Agent-first, human-compatible** — Designed for machine-to-machine payments, but supports human-in-the-loop confirmation when required (e.g., STK Push, 3DS).

7. **Trust-minimizing** — Facilitators cannot move funds beyond what the payment payload authorizes. Mandates are scoped and time-limited.

---

## Contributing

ACP is in its early stages. We're looking for contributors, especially those with experience in:

- Payment provider integrations (Stripe, Razorpay, Daraja, PagSeguro, Adyen, etc.)
- Protocol design and specification writing
- Go middleware and SDK development
- AI agent frameworks (LangChain, AutoGen, CrewAI, etc.)
- MCP and A2A protocol implementations

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

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
