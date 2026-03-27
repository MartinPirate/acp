# ACP: Agentic Commerce Protocol Specification v1

## Abstract

The Agentic Commerce Protocol (ACP) defines a standard mechanism for AI agents to discover, negotiate, and execute payments when accessing paid resources. ACP introduces a structured flow built on the HTTP 402 Payment Required status code, enabling resource servers to declare payment requirements and AI agents to fulfill them through pluggable payment methods and facilitators.

This specification defines the data types, transport bindings, intent model, facilitator API, mandate system, and security considerations necessary for interoperable agent-to-service commerce.

## Status of This Memo

This document specifies version 1 of the Agentic Commerce Protocol. It is intended as the authoritative reference for implementors of ACP-compatible agents, resource servers, facilitators, and payment methods.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Protocol Overview](#2-protocol-overview)
3. [Protocol Flow](#3-protocol-flow)
4. [Data Types](#4-data-types)
5. [HTTP Transport Binding](#5-http-transport-binding)
6. [MCP Transport Binding](#6-mcp-transport-binding)
7. [A2A Transport Binding](#7-a2a-transport-binding)
8. [gRPC Transport Binding](#8-grpc-transport-binding)
9. [Intents](#9-intents)
10. [Methods](#10-methods)
11. [Facilitator API](#11-facilitator-api)
12. [Mandates](#12-mandates)
13. [Security Considerations](#13-security-considerations)
14. [Error Handling](#14-error-handling)
15. [Extensibility](#15-extensibility)
16. [IANA Considerations](#16-iana-considerations)
- [Appendix A: JSON Schema for All Types](#appendix-a-json-schema-for-all-types)
- [Appendix B: Example Flows](#appendix-b-example-flows)

---

## 1. Introduction

### 1.1 Purpose

As AI agents become autonomous participants in digital commerce, they require a standardized protocol for discovering payment requirements, selecting payment methods, and executing transactions without human intervention. ACP provides this protocol layer.

### 1.2 Scope

ACP covers:

- The negotiation flow between an agent (client) and a resource server
- The data formats exchanged during payment negotiation and settlement
- The facilitator API for third-party payment processing
- Transport bindings for HTTP, MCP, A2A, and gRPC
- A mandate system for pre-authorized recurring or bounded payments
- Discovery of available payment services

ACP does NOT cover:

- The internal implementation of payment rails (e.g., card network protocols)
- Agent authentication or identity management (these are delegated to existing standards)
- Pricing models or billing logic within resource servers

### 1.3 Terminology

| Term | Definition |
|------|-----------|
| **Agent** | An AI system acting as a client that consumes paid resources |
| **Resource Server** | A service that provides resources gated behind payment |
| **Facilitator** | A remote service that verifies and settles payments on behalf of a resource server |
| **Method** | A concrete payment rail (e.g., card, UPI, PIX, crypto) |
| **Intent** | The type of payment action requested (charge, authorize, subscribe, mandate) |
| **Payment Option** | A specific combination of method, intent, currency, and amount offered by a resource server |
| **Mandate** | A pre-authorized agreement allowing bounded future payments |
| **Settlement** | The final execution of a payment transaction |

### 1.4 Conventions

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

---

## 2. Protocol Overview

### 2.1 Roles

ACP defines three primary roles:

```
+----------+          +-----------------+          +--------------+
|  Agent   |  <--->   | Resource Server |  <--->   | Facilitator  |
| (Client) |          |   (Merchant)    |          |  (Processor) |
+----------+          +-----------------+          +--------------+
```

**Agent (Client):** An AI agent that wants to access a paid resource. The agent inspects payment requirements, selects a payment option, constructs a payment payload, and submits it.

**Resource Server (Merchant):** A service that hosts paid resources. It declares what payment it accepts (via `PaymentRequired`) and processes or delegates payment verification and settlement.

**Facilitator (Processor):** An optional remote service that handles payment verification and settlement on behalf of the resource server. Facilitators enable resource servers to accept payments without implementing payment logic directly.

### 2.2 Design Principles

1. **Method Agnostic:** ACP is not tied to any payment rail. New methods can be added without protocol changes.
2. **Transport Agnostic:** The core data types work across HTTP, MCP, A2A, and gRPC.
3. **Facilitator Optional:** Resource servers MAY process payments directly or delegate to facilitators.
4. **Agent Autonomy:** Agents select from offered payment options based on their own policies, budgets, and preferences.
5. **Extensible:** Every major data type includes an `extensions` field for forward compatibility.

---

## 3. Protocol Flow

### 3.1 Basic Charge Flow

```
Agent                    Resource Server              Facilitator
  |                            |                           |
  |  1. GET /resource          |                           |
  |--------------------------->|                           |
  |                            |                           |
  |  2. 402 Payment Required   |                           |
  |    ACP-Payment-Required:   |                           |
  |    {accepts: [...]}        |                           |
  |<---------------------------|                           |
  |                            |                           |
  |  3. Agent selects option   |                           |
  |     and builds payload     |                           |
  |                            |                           |
  |  4. GET /resource          |                           |
  |    ACP-Payment:            |                           |
  |    {accepted, payload}     |                           |
  |--------------------------->|                           |
  |                            |                           |
  |                            |  5. POST /verify          |
  |                            |     {payload, reqs}       |
  |                            |-------------------------->|
  |                            |                           |
  |                            |  6. {valid: true}         |
  |                            |<--------------------------|
  |                            |                           |
  |                            |  7. POST /settle          |
  |                            |     {payload, reqs}       |
  |                            |-------------------------->|
  |                            |                           |
  |                            |  8. {success, txn, ...}   |
  |                            |<--------------------------|
  |                            |                           |
  |  9. 200 OK                 |                           |
  |    ACP-Payment-Response:   |                           |
  |    {success, transaction}  |                           |
  |    + resource body         |                           |
  |<---------------------------|                           |
  |                            |                           |
```

### 3.2 Step-by-Step

1. **Request:** The agent sends a standard request to the resource server.
2. **Payment Required:** The server responds with HTTP 402 and an `ACP-Payment-Required` header containing a `PaymentRequired` object. This lists all accepted payment options.
3. **Selection:** The agent examines the offered options, checks its budget and method availability, and selects one.
4. **Payment Submission:** The agent re-sends the original request with an `ACP-Payment` header containing a `PaymentPayload` object.
5. **Verification:** The resource server (or its middleware) sends the payload to a facilitator for verification.
6. **Verification Response:** The facilitator confirms the payment is valid.
7. **Settlement:** The resource server requests settlement from the facilitator.
8. **Settlement Response:** The facilitator executes the payment and returns a receipt.
9. **Resource Delivery:** The resource server delivers the resource along with an `ACP-Payment-Response` header containing the settlement receipt.

### 3.3 Direct Processing Flow

When no facilitator is used, steps 5-8 are handled internally by the resource server using a registered `Method` implementation.

---

## 4. Data Types

### 4.1 Resource

Identifies the resource being paid for.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | REQUIRED | Canonical URL of the resource |
| `description` | string | OPTIONAL | Human-readable description |
| `mimeType` | string | OPTIONAL | MIME type of the resource |

### 4.2 PaymentOption

A single accepted payment method with its parameters.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `intent` | Intent | REQUIRED | Payment intent type |
| `method` | string | REQUIRED | Payment method identifier |
| `currency` | Currency | REQUIRED | ISO 4217 currency code or token symbol |
| `amount` | string | REQUIRED | Payment amount as a decimal string |
| `description` | string | OPTIONAL | Human-readable description of this option |
| `extra` | object | OPTIONAL | Method-specific additional data |

Amount MUST be represented as a non-negative decimal string (e.g., `"10.00"`, `"0.50"`). Implementations MUST use arbitrary-precision arithmetic for amount comparisons.

### 4.3 PaymentRequired

The 402 response payload.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `acpVersion` | integer | REQUIRED | Protocol version (currently `1`) |
| `resource` | Resource | REQUIRED | The resource requiring payment |
| `accepts` | PaymentOption[] | REQUIRED | List of accepted payment options |
| `extensions` | object | OPTIONAL | Protocol extensions |

### 4.4 PaymentPayload

The agent's payment submission.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `acpVersion` | integer | REQUIRED | Protocol version |
| `resource` | Resource | REQUIRED | The resource being paid for |
| `accepted` | PaymentOption | REQUIRED | The selected payment option |
| `payload` | object | REQUIRED | Method-specific payment data |
| `extensions` | object | OPTIONAL | Protocol extensions |

### 4.5 VerifyResponse

Facilitator verification result.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `valid` | boolean | REQUIRED | Whether the payment is valid |
| `reason` | string | OPTIONAL | Explanation if invalid |
| `payer` | string | OPTIONAL | Identified payer (agent ID, wallet, etc.) |

### 4.6 SettleResponse

Settlement receipt.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `acpVersion` | integer | REQUIRED | Protocol version |
| `success` | boolean | REQUIRED | Whether settlement succeeded |
| `method` | string | REQUIRED | Method used for settlement |
| `transaction` | string | REQUIRED | Unique transaction identifier |
| `settledAt` | string | REQUIRED | ISO 8601 timestamp of settlement |
| `receipt` | object | OPTIONAL | Method-specific receipt data |
| `extensions` | object | OPTIONAL | Protocol extensions |

### 4.7 Price

Server-side price declaration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount` | string | REQUIRED | Price amount as a decimal string |
| `currency` | Currency | REQUIRED | Currency code |

### 4.8 PaymentError

Structured error response.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | ErrorCode | REQUIRED | Machine-readable error code |
| `message` | string | REQUIRED | Human-readable error message |
| `method` | string | OPTIONAL | Method that caused the error |

---

## 5. HTTP Transport Binding

### 5.1 Headers

ACP uses three custom HTTP headers:

| Header | Direction | Content |
|--------|-----------|---------|
| `ACP-Payment-Required` | Response (402) | JSON-encoded `PaymentRequired` |
| `ACP-Payment` | Request | JSON-encoded `PaymentPayload` |
| `ACP-Payment-Response` | Response (200) | JSON-encoded `SettleResponse` |

### 5.2 Status Codes

| Code | Usage |
|------|-------|
| `402 Payment Required` | Resource requires payment; body and header contain `PaymentRequired` |
| `200 OK` | Payment accepted, resource delivered; header contains `SettleResponse` |
| `400 Bad Request` | Malformed payment payload |
| `422 Unprocessable Entity` | Payment verification or settlement failed |

### 5.3 Content Types

- Headers carry JSON-encoded ACP objects.
- The response body for 402 SHOULD also include the `PaymentRequired` object as JSON with `Content-Type: application/json`.
- The response body for 200 contains the requested resource in its native content type.

### 5.4 Request Flow

1. Client sends request without `ACP-Payment` header.
2. Server returns `402` with `ACP-Payment-Required` header.
3. Client sends same request with `ACP-Payment` header.
4. Server processes payment and returns `200` with `ACP-Payment-Response` header.

### 5.5 Idempotency

Clients SHOULD include an `Idempotency-Key` header (RFC draft) with payment requests. Servers MUST NOT charge the same idempotency key twice within a reasonable window (RECOMMENDED: 24 hours).

---

## 6. MCP Transport Binding

### 6.1 Overview

When ACP operates within the Model Context Protocol (MCP), payment negotiation occurs through MCP tool calls and responses rather than HTTP headers.

### 6.2 Tool Definition

An MCP server MAY expose a tool that returns a `PaymentRequired` object when the resource requires payment:

```json
{
  "name": "access_resource",
  "description": "Access a paid resource",
  "inputSchema": {
    "type": "object",
    "properties": {
      "resource_url": { "type": "string" },
      "acp_payment": { "$ref": "#/definitions/PaymentPayload" }
    }
  }
}
```

### 6.3 Flow

1. Agent calls the MCP tool without `acp_payment`.
2. Tool returns a result containing `PaymentRequired` with `isError: false` and a structured content block.
3. Agent constructs a `PaymentPayload` and re-invokes the tool with `acp_payment` populated.
4. Tool processes payment and returns the resource content along with a `SettleResponse`.

### 6.4 Content Blocks

Payment data SHOULD be conveyed in MCP content blocks with:
- `type: "resource"` and `mimeType: "application/acp+json"`

---

## 7. A2A Transport Binding

### 7.1 Overview

When two agents negotiate payment via the Agent-to-Agent (A2A) protocol, ACP messages are embedded within A2A task artifacts.

### 7.2 Task Parts

ACP data is carried as A2A `DataPart` objects:

```json
{
  "type": "data",
  "mimeType": "application/acp+json",
  "data": { ... }
}
```

### 7.3 Flow

1. Client agent sends an A2A task requesting a resource.
2. Server agent responds with a task update containing a `PaymentRequired` data part and status `input-required`.
3. Client agent sends a task update with a `PaymentPayload` data part.
4. Server agent processes payment and completes the task with the resource and a `SettleResponse` data part.

### 7.4 Agent Cards

ACP-capable agents SHOULD advertise their payment capabilities in their A2A Agent Card under the `skills` section, including supported methods and currencies.

---

## 8. gRPC Transport Binding

### 8.1 Overview

For high-throughput agent-to-service communication, ACP defines a gRPC service binding.

### 8.2 Service Definition

```protobuf
syntax = "proto3";
package acp.v1;

service ACPService {
  rpc RequestResource(ResourceRequest) returns (ResourceResponse);
  rpc SubmitPayment(PaymentSubmission) returns (PaymentResult);
}

message ResourceRequest {
  string url = 1;
  map<string, string> metadata = 2;
}

message ResourceResponse {
  oneof result {
    PaymentRequired payment_required = 1;
    ResourceContent content = 2;
  }
}

message PaymentSubmission {
  string url = 1;
  PaymentPayload payment = 2;
}

message PaymentResult {
  bool success = 1;
  SettleResponse settlement = 2;
  bytes content = 3;
  string content_type = 4;
}
```

### 8.3 Metadata

ACP version and idempotency keys SHOULD be conveyed via gRPC metadata headers:
- `acp-version`: Protocol version
- `idempotency-key`: Unique request identifier

---

## 9. Intents

### 9.1 charge

**Purpose:** Immediate one-time payment.

The resource server requests a fixed amount. The agent pays and receives the resource. Settlement is expected to complete synchronously (or near-synchronously).

| Field | Constraint |
|-------|-----------|
| `amount` | REQUIRED, MUST be > 0 |
| `currency` | REQUIRED |

### 9.2 authorize

**Purpose:** Pre-authorization without immediate capture.

The agent authorizes a maximum amount. The resource server may capture up to the authorized amount later. Useful for metered or usage-based pricing.

| Field | Constraint |
|-------|-----------|
| `amount` | REQUIRED, represents the maximum capturable amount |
| `currency` | REQUIRED |

Authorization SHOULD have a finite validity window (RECOMMENDED: 7 days).

### 9.3 subscribe

**Purpose:** Recurring payment enrollment.

The agent agrees to a recurring payment schedule. The `extra` field MUST contain subscription parameters:

```json
{
  "interval": "monthly",
  "periods": 12,
  "trialDays": 0
}
```

| Extra Field | Type | Description |
|-------------|------|-------------|
| `interval` | string | Billing interval: "daily", "weekly", "monthly", "yearly" |
| `periods` | integer | Number of billing periods (0 = indefinite) |
| `trialDays` | integer | Free trial period in days |

### 9.4 mandate

**Purpose:** Pre-authorized bounded future payments.

The agent grants a mandate allowing the resource server to initiate payments within defined bounds. See Section 12 for mandate specification.

---

## 10. Methods

### 10.1 Method Interface

A payment method MUST implement the following operations:

| Operation | Description |
|-----------|-------------|
| `Name()` | Returns the unique method identifier string |
| `SupportedIntents()` | Returns the list of intents this method handles |
| `SupportedCurrencies()` | Returns the list of currencies this method supports |
| `BuildOption(intent, price)` | Constructs a `PaymentOption` for the 402 response |
| `CreatePayload(ctx, option)` | Builds method-specific payment data |
| `Verify(ctx, payload, option)` | Validates a payment without settling |
| `Settle(ctx, payload, option)` | Executes the payment on this rail |

### 10.2 Method Naming

Method names MUST be lowercase alphanumeric strings with optional hyphens. Examples: `card`, `upi`, `pix`, `usdc-base`, `mpesa`.

Method names SHOULD follow the convention `{rail}` or `{rail}-{network}` for specificity.

### 10.3 Method Discovery

Methods are discovered through:

1. **Static registration:** Methods registered programmatically with a Gateway.
2. **Facilitator query:** Calling `GET /supported` on a facilitator to learn its capabilities.
3. **Well-known endpoint:** `GET /.well-known/acp-services` returns a list of available services including their supported methods.

### 10.4 Standard Methods

The following method identifiers are reserved:

| Identifier | Description |
|------------|-------------|
| `card` | Credit/debit card payments |
| `upi` | Unified Payments Interface (India) |
| `pix` | PIX instant payment (Brazil) |
| `mpesa` | M-Pesa mobile money (East Africa) |
| `usdc` | USDC stablecoin |
| `mock` | Test/development method (MUST NOT be used in production) |

---

## 11. Facilitator API

### 11.1 Overview

A facilitator exposes three HTTP endpoints for remote payment processing.

### 11.2 POST /verify

Verifies that a payment payload is valid without executing settlement.

**Request:**

```json
{
  "payload": { PaymentPayload },
  "requirements": { PaymentRequired }
}
```

**Response (200):**

```json
{
  "valid": true,
  "payer": "agent-xyz"
}
```

**Response (200, invalid):**

```json
{
  "valid": false,
  "reason": "insufficient funds"
}
```

### 11.3 POST /settle

Executes the payment and returns a settlement receipt.

**Request:**

```json
{
  "payload": { PaymentPayload },
  "requirements": { PaymentRequired }
}
```

**Response (200):**

```json
{
  "acpVersion": 1,
  "success": true,
  "method": "card",
  "transaction": "txn_abc123",
  "settledAt": "2026-03-27T10:00:00Z",
  "receipt": { ... }
}
```

**Response (422, failure):**

```json
{
  "error": "settlement_failed: card declined"
}
```

### 11.4 GET /supported

Returns the methods, intents, and currencies the facilitator handles.

**Response (200):**

```json
{
  "methods": ["card", "upi", "pix"],
  "intents": ["charge", "authorize", "subscribe", "mandate"],
  "currencies": ["USD", "EUR", "INR", "BRL"]
}
```

### 11.5 GET /health

Returns the operational status of the facilitator.

**Response (200):**

```json
{
  "status": "ok"
}
```

---

## 12. Mandates

### 12.1 Overview

A mandate is a pre-authorized agreement that allows a resource server to initiate future payments within defined constraints. Mandates enable autonomous agent operation without per-transaction approval.

### 12.2 Mandate Specification

A mandate is defined within a `PaymentOption` with `intent: "mandate"`. The `extra` field MUST contain:

```json
{
  "mandateId": "mnd_unique_id",
  "maxAmount": "100.00",
  "maxTotal": "1000.00",
  "maxCount": 50,
  "validFrom": "2026-03-27T00:00:00Z",
  "validUntil": "2027-03-27T00:00:00Z",
  "cooldownSeconds": 3600,
  "allowedMethods": ["card", "usdc"],
  "description": "API access up to $100 per request"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mandateId` | string | REQUIRED | Unique mandate identifier |
| `maxAmount` | string | REQUIRED | Maximum amount per individual charge |
| `maxTotal` | string | OPTIONAL | Maximum cumulative amount over mandate lifetime |
| `maxCount` | integer | OPTIONAL | Maximum number of charges |
| `validFrom` | string | OPTIONAL | ISO 8601 start of validity (default: now) |
| `validUntil` | string | REQUIRED | ISO 8601 end of validity |
| `cooldownSeconds` | integer | OPTIONAL | Minimum seconds between charges |
| `allowedMethods` | string[] | OPTIONAL | Restrict to specific methods |
| `description` | string | OPTIONAL | Human-readable purpose |

### 12.3 Enforcement Rules

1. A payment MUST be rejected if `amount > maxAmount`.
2. A payment MUST be rejected if cumulative total would exceed `maxTotal`.
3. A payment MUST be rejected if total charge count would exceed `maxCount`.
4. A payment MUST be rejected if the current time is outside `[validFrom, validUntil]`.
5. A payment MUST be rejected if less than `cooldownSeconds` have elapsed since the last charge.
6. A payment MUST be rejected if the method is not in `allowedMethods` (when specified).

### 12.4 Mandate Lifecycle

1. **Creation:** Agent approves a mandate by submitting a `PaymentPayload` with `intent: "mandate"`.
2. **Active:** Resource server can initiate charges within mandate bounds.
3. **Exhausted:** Mandate reaches `maxTotal`, `maxCount`, or `validUntil`.
4. **Revoked:** Agent explicitly revokes the mandate.

---

## 13. Security Considerations

### 13.1 Authentication

ACP does not define its own authentication mechanism. Implementations SHOULD use existing standards:

- **HTTP:** Bearer tokens (RFC 6750), mutual TLS, or API keys
- **MCP:** MCP's built-in authentication
- **A2A:** A2A's agent authentication
- **gRPC:** TLS client certificates or token-based auth via metadata

### 13.2 Authorization

Resource servers MUST verify that the agent submitting a payment is authorized to do so. For mandates, the server MUST verify that the mandate was granted by the agent claiming to pay.

### 13.3 Replay Prevention

1. Implementations MUST support idempotency keys to prevent duplicate charges.
2. Settlement responses MUST include unique transaction identifiers.
3. Facilitators SHOULD reject payment payloads that have already been settled.

### 13.4 Amount Validation

1. Resource servers MUST verify that the amount in the `PaymentPayload` matches the amount in the original `PaymentRequired` response.
2. Amounts MUST be compared using arbitrary-precision arithmetic, not floating-point.
3. Currency codes MUST match exactly between the offered option and the submitted payment.

### 13.5 Transport Security

1. All ACP communication MUST use TLS 1.2 or later in production.
2. Facilitator URLs MUST use HTTPS.
3. Well-known endpoints MUST be served over HTTPS.

### 13.6 Payload Integrity

Implementations SHOULD sign payment payloads to prevent tampering. The signing mechanism is method-specific and defined by the individual method specification.

### 13.7 Rate Limiting

Facilitators and resource servers SHOULD implement rate limiting to prevent abuse. Rate limit status SHOULD be communicated via standard `Retry-After` headers.

---

## 14. Error Handling

### 14.1 Error Codes

| Code | Description | Retryable |
|------|-------------|-----------|
| `insufficient_funds` | Payer has insufficient funds | No |
| `method_unavailable` | Requested method is not available | Yes (with different method) |
| `mandate_exceeded` | Payment exceeds mandate limits | No |
| `mandate_expired` | Mandate has expired | No |
| `currency_mismatch` | Currency does not match | No |
| `amount_too_high` | Amount exceeds maximum | No |
| `verification_failed` | Payment verification failed | Yes (with new payload) |
| `settlement_failed` | Settlement could not be completed | Yes (after delay) |
| `timeout` | Operation timed out | Yes |
| `invalid_payload` | Malformed payment data | No (fix payload) |
| `unsupported_intent` | Method does not support this intent | No |
| `budget_exceeded` | Agent budget limit reached | No |

### 14.2 Error Response Format

```json
{
  "code": "settlement_failed",
  "message": "Card issuer declined the transaction",
  "method": "card"
}
```

### 14.3 Retry Semantics

1. Clients SHOULD implement exponential backoff for retryable errors.
2. The initial retry delay SHOULD be at least 1 second.
3. Clients MUST NOT retry non-retryable errors.
4. Clients MUST use the same idempotency key when retrying.
5. After 3 consecutive failures on one method, clients SHOULD try an alternative method.

---

## 15. Extensibility

### 15.1 Extension Fields

All major data types (`PaymentRequired`, `PaymentPayload`, `SettleResponse`) include an `extensions` field (JSON object). This allows protocol extensions without version bumps.

### 15.2 Adding New Methods

1. Choose a unique method name following the naming convention in Section 10.2.
2. Implement the Method interface (Section 10.1).
3. Register the method with a Gateway or Facilitator.
4. Document the method-specific `extra` and `payload` formats.
5. Register with the ACP method registry (when established).

### 15.3 Adding New Transports

1. Define how `PaymentRequired`, `PaymentPayload`, and `SettleResponse` are conveyed in the transport.
2. Define the negotiation flow (equivalent to the 402 handshake).
3. Specify idempotency and replay prevention mechanisms.
4. Document transport-specific security requirements.

### 15.4 Adding New Intents

1. Define the intent semantics (what the payment action means).
2. Specify required and optional fields in `PaymentOption.extra`.
3. Define the settlement behavior.
4. Update the `Intent` type and validation logic.

### 15.5 Version Negotiation

The `acpVersion` field enables version negotiation. Servers MUST include their supported version. Clients SHOULD reject responses with an unrecognized version. Future versions SHOULD maintain backward compatibility where possible.

---

## 16. IANA Considerations

### 16.1 HTTP Header Registration

This specification defines three HTTP headers for registration with IANA:

| Header | Status | Reference |
|--------|--------|-----------|
| `ACP-Payment-Required` | Provisional | This specification, Section 5.1 |
| `ACP-Payment` | Provisional | This specification, Section 5.1 |
| `ACP-Payment-Response` | Provisional | This specification, Section 5.1 |

### 16.2 HTTP 402 Status Code

ACP formally uses the HTTP 402 Payment Required status code (RFC 9110, Section 15.5.3) for its intended purpose: indicating that payment is required to access the resource.

### 16.3 Media Type

This specification defines the media type `application/acp+json` for ACP data objects when transported outside of HTTP headers.

### 16.4 Well-Known URI

This specification registers the well-known URI `/.well-known/acp-services` for service discovery (RFC 8615).

---

## Appendix A: JSON Schema for All Types

### A.1 Resource

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Resource",
  "type": "object",
  "required": ["url"],
  "properties": {
    "url": { "type": "string", "format": "uri" },
    "description": { "type": "string" },
    "mimeType": { "type": "string" }
  }
}
```

### A.2 PaymentOption

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PaymentOption",
  "type": "object",
  "required": ["intent", "method", "currency", "amount"],
  "properties": {
    "intent": { "type": "string", "enum": ["charge", "authorize", "subscribe", "mandate"] },
    "method": { "type": "string", "pattern": "^[a-z0-9-]+$" },
    "currency": { "type": "string", "minLength": 3, "maxLength": 6 },
    "amount": { "type": "string", "pattern": "^[0-9]+(\\.[0-9]+)?$" },
    "description": { "type": "string" },
    "extra": { "type": "object" }
  }
}
```

### A.3 PaymentRequired

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PaymentRequired",
  "type": "object",
  "required": ["acpVersion", "resource", "accepts"],
  "properties": {
    "acpVersion": { "type": "integer", "const": 1 },
    "resource": { "$ref": "#/$defs/Resource" },
    "accepts": {
      "type": "array",
      "items": { "$ref": "#/$defs/PaymentOption" },
      "minItems": 1
    },
    "extensions": { "type": "object" }
  }
}
```

### A.4 PaymentPayload

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PaymentPayload",
  "type": "object",
  "required": ["acpVersion", "resource", "accepted", "payload"],
  "properties": {
    "acpVersion": { "type": "integer", "const": 1 },
    "resource": { "$ref": "#/$defs/Resource" },
    "accepted": { "$ref": "#/$defs/PaymentOption" },
    "payload": { "type": "object" },
    "extensions": { "type": "object" }
  }
}
```

### A.5 VerifyResponse

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "VerifyResponse",
  "type": "object",
  "required": ["valid"],
  "properties": {
    "valid": { "type": "boolean" },
    "reason": { "type": "string" },
    "payer": { "type": "string" }
  }
}
```

### A.6 SettleResponse

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "SettleResponse",
  "type": "object",
  "required": ["acpVersion", "success", "method", "transaction", "settledAt"],
  "properties": {
    "acpVersion": { "type": "integer" },
    "success": { "type": "boolean" },
    "method": { "type": "string" },
    "transaction": { "type": "string" },
    "settledAt": { "type": "string", "format": "date-time" },
    "receipt": { "type": "object" },
    "extensions": { "type": "object" }
  }
}
```

### A.7 PaymentError

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PaymentError",
  "type": "object",
  "required": ["code", "message"],
  "properties": {
    "code": {
      "type": "string",
      "enum": [
        "insufficient_funds", "method_unavailable", "mandate_exceeded",
        "mandate_expired", "currency_mismatch", "amount_too_high",
        "verification_failed", "settlement_failed", "timeout",
        "invalid_payload", "unsupported_intent", "budget_exceeded"
      ]
    },
    "message": { "type": "string" },
    "method": { "type": "string" }
  }
}
```

---

## Appendix B: Example Flows

### B.1 Simple Charge (HTTP)

**Step 1: Agent requests resource**

```http
GET /api/premium-data HTTP/1.1
Host: data-service.example.com
Authorization: Bearer agent_token_xyz
```

**Step 2: Server responds with 402**

```http
HTTP/1.1 402 Payment Required
Content-Type: application/json
ACP-Payment-Required: {"acpVersion":1,"resource":{"url":"https://data-service.example.com/api/premium-data","description":"Premium dataset"},"accepts":[{"intent":"charge","method":"card","currency":"USD","amount":"5.00"},{"intent":"charge","method":"usdc","currency":"USDC","amount":"5.00"}]}

{
  "acpVersion": 1,
  "resource": {
    "url": "https://data-service.example.com/api/premium-data",
    "description": "Premium dataset"
  },
  "accepts": [
    {
      "intent": "charge",
      "method": "card",
      "currency": "USD",
      "amount": "5.00"
    },
    {
      "intent": "charge",
      "method": "usdc",
      "currency": "USDC",
      "amount": "5.00"
    }
  ]
}
```

**Step 3: Agent selects USDC and submits payment**

```http
GET /api/premium-data HTTP/1.1
Host: data-service.example.com
Authorization: Bearer agent_token_xyz
Idempotency-Key: idem_abc123
ACP-Payment: {"acpVersion":1,"resource":{"url":"https://data-service.example.com/api/premium-data"},"accepted":{"intent":"charge","method":"usdc","currency":"USDC","amount":"5.00"},"payload":{"txHash":"0xabc...","chain":"base","sender":"0x123..."}}
```

**Step 4: Server processes payment and delivers resource**

```http
HTTP/1.1 200 OK
Content-Type: application/json
ACP-Payment-Response: {"acpVersion":1,"success":true,"method":"usdc","transaction":"txn_def456","settledAt":"2026-03-27T10:30:00Z"}

{
  "data": [ ... ]
}
```

### B.2 Mandate Setup (HTTP)

**Step 1: Server offers mandate option**

```http
HTTP/1.1 402 Payment Required
ACP-Payment-Required: {"acpVersion":1,"resource":{"url":"https://api.example.com/v1"},"accepts":[{"intent":"mandate","method":"card","currency":"USD","amount":"0.00","extra":{"mandateId":"mnd_001","maxAmount":"10.00","maxTotal":"500.00","validUntil":"2027-03-27T00:00:00Z"}}]}
```

**Step 2: Agent approves mandate**

```http
POST /api/v1/mandate HTTP/1.1
ACP-Payment: {"acpVersion":1,"resource":{"url":"https://api.example.com/v1"},"accepted":{"intent":"mandate","method":"card","currency":"USD","amount":"0.00","extra":{"mandateId":"mnd_001","maxAmount":"10.00","maxTotal":"500.00","validUntil":"2027-03-27T00:00:00Z"}},"payload":{"cardToken":"tok_xyz","approval":"granted"}}
```

### B.3 MCP Tool Flow

**Step 1: Agent calls MCP tool**

```json
{
  "method": "tools/call",
  "params": {
    "name": "access_resource",
    "arguments": {
      "resource_url": "https://data.example.com/report"
    }
  }
}
```

**Step 2: Tool returns payment requirement**

```json
{
  "content": [
    {
      "type": "resource",
      "mimeType": "application/acp+json",
      "text": "{\"acpVersion\":1,\"resource\":{\"url\":\"https://data.example.com/report\"},\"accepts\":[{\"intent\":\"charge\",\"method\":\"usdc\",\"currency\":\"USDC\",\"amount\":\"1.00\"}]}"
    }
  ]
}
```

**Step 3: Agent re-invokes with payment**

```json
{
  "method": "tools/call",
  "params": {
    "name": "access_resource",
    "arguments": {
      "resource_url": "https://data.example.com/report",
      "acp_payment": {
        "acpVersion": 1,
        "resource": { "url": "https://data.example.com/report" },
        "accepted": { "intent": "charge", "method": "usdc", "currency": "USDC", "amount": "1.00" },
        "payload": { "txHash": "0xdef..." }
      }
    }
  }
}
```

---

*End of ACP Specification v1*
