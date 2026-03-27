# ACP (Agentic Commerce Protocol) - Global Payment Research

> Compiled: March 2026
> Purpose: Inform the architecture of an open, payment-rail-agnostic middleware protocol for AI agent payments

---

## Table of Contents

1. [Global Payment Rails & Aggregators](#1-global-payment-rails--aggregators)
2. [Open Banking / PSD2 / A2A Payments](#2-open-banking--psd2--a2a-payments)
3. [Real-Time Payment Systems by Country](#3-real-time-payment-systems-by-country)
4. [Payment Orchestration Platforms](#4-payment-orchestration-platforms)
5. [Existing Agent Payment Research](#5-existing-agent-payment-research)
6. [Authentication & Authorization for Agent Payments](#6-authentication--authorization-for-agent-payments)
7. [Key Technical Considerations](#7-key-technical-considerations)

---

## 1. Global Payment Rails & Aggregators

### 1.1 India

**UPI (Unified Payments Interface)**
- Operated by NPCI (National Payments Corporation of India), launched August 2016
- Processed over **228 billion transactions** worth Rs 300 trillion in 2025
- Global leader in real-time digital payments by volume
- Developer-accessible via payment gateway intermediaries (not direct NPCI API access for most developers)

**Key Providers:**

| Provider | Notes |
|----------|-------|
| **Razorpay** | Dominant developer ecosystem. Clean REST APIs, UPI AutoPay, plug-and-play SDKs. Supports recurring mandates. |
| **Cashfree** | Fast settlement, strong API docs, growing developer community |
| **PhonePe** | One of India's largest UPI apps, offers PG APIs for merchants |
| **Paytm PG** | Large user base, full-stack payment gateway |
| **Decentro** | Banking-as-a-service APIs including UPI APIs for account-level integration |
| **Juspay** | Payment orchestration layer (also builds Hyperswitch) |

**Key Features for ACP:**
- **UPI AutoPay** (UPI 2.0): Recurring e-mandate system. Customer authorizes once, future debits happen automatically. Supports subscriptions, EMIs, bill payments. Max limit INR 1,00,000/txn for auto-debit.
- **UPI Lite**: Small-value offline transactions (no PIN needed under INR 500)
- **Tokenized payments**: Card-on-file tokenization per RBI mandate
- **Integration cost**: INR 40,000 (basic) to INR 8,00,000 (enterprise direct API)

**Relevance to ACP:** UPI's mandate system is a natural fit for agent-initiated recurring payments. The e-mandate concept (authorize once, debit later within constraints) maps directly to the "Intent Mandate" pattern used in AP2.

---

### 1.2 China

**Alipay**
- Operated by Ant Group
- **Alipay+**: Cross-border payment solution enabling international acceptance without local Chinese registration
- **MCP Server** (launched April 2025): Enables AI developers to integrate Alipay payment services into AI agents via the Model Context Protocol
- **AI Pay Performance**: Surpassed **120 million transactions in one week** (Feb 2026) executed by AI agents
- **Agentic Commerce Trust Protocol** (Jan 2026): Standardized framework for AI-led financial interactions, developed with Taobao, Alibaba Qwen

**Developer Integration Options:**
- Direct Alipay API (requires Chinese entity or cross-border merchant agreement)
- Via Alipay+ for international merchants
- Via aggregators: AlphaPay, Adyen, Stripe (as local payment method)
- **MCP Server** (GitHub: `alipay/global-alipayplus-mcp`): AI agents can process payments, check status, initiate refunds via natural language

**WeChat Pay**
- Operated by Tencent (Tenpay Global for cross-border)
- International merchants use Tenpay Global without needing Chinese company
- Integration via QR code, in-app, H5, and mini-program payments
- Settlement in major currencies: USD, EUR, GBP, HKD
- Transaction fees ~2% (varies by volume)

**UnionPay**
- Card network with global acceptance
- Available through most global PSPs

**Relevance to ACP:** Alipay's MCP server is the most advanced production deployment of agent-initiated payments globally. Their Agentic Commerce Trust Protocol is a direct competitor/complement. The 120M weekly AI transactions prove market demand.

---

### 1.3 Southeast Asia

| Country | Primary Methods | Developer Access |
|---------|----------------|-----------------|
| **Singapore** | GrabPay, PayNow | Via Adyen, Stripe, 2C2P, ECOMMPAY |
| **Philippines** | GCash, GrabPay, DragonPay | Via Xendit, PayMongo, 2C2P |
| **Indonesia** | OVO, GoPay (Gojek, 190M+ downloads), DANA, DOKU | Via Xendit, Midtrans (Goto), 2C2P |
| **Thailand** | PromptPay (bank-linked instant), TrueMoney | Via 2C2P, Omise (Opn) |
| **Vietnam** | MoMo, ZaloPay, VNPay | Via 2C2P, PayOS |
| **Malaysia** | Touch 'n Go, DuitNow, Boost | Via 2C2P, iPay88 |

**Key Aggregators for SEA:**
- **2C2P**: Regional PSP covering 9 SEA countries, supports GrabPay, TrueMoney, and local methods
- **Xendit**: Indonesia/Philippines focused, strong developer APIs
- **Adyen/Stripe**: Global PSPs with SEA local method support

**GoPay (Indonesia):**
- REST API with clear documentation, GitHub examples, SDKs
- Payment gateway integration available at gopay.com/en/integration/

**Relevance to ACP:** SEA is highly fragmented -- each country has its own dominant wallet. A protocol must support wallet-based payments where the user confirms in their app (redirect/deep-link flow), not just card-based flows.

---

### 1.4 Japan

| Provider | Users | Notes |
|----------|-------|-------|
| **PayPay** | 68M+ users | Largest QR-code payment. Open Payment API with sandbox (PayPay Lab). Supports Web Payment and Native Payment flows. Settlement in 4 business days. |
| **LINE Pay** | Significant | Credit card integration, invoice support |
| **Rakuten Pay** | Large | Tied to Rakuten ecosystem |
| **Merpay** | Growing | Tied to Mercari marketplace |

**Developer Integration:**
- PayPay offers continuous payments API for recurring/subscription use cases
- Integration via Stripe (since 2025), Square, or direct PayPay API
- Sandbox environment for testing full payment flows

---

### 1.5 Latin America

**Brazil:**

| System | Type | Notes |
|--------|------|-------|
| **PIX** | Instant payment (Central Bank) | Launched Nov 2020. Instant 24/7 transfers. QR code + copy-paste code. **Pix Automatico** (June 2025): recurring payments without per-transaction approval. **Pix por Aproximacao** (Feb 2025): NFC contactless via Open Finance API. |
| **MercadoPago** | PSP/Wallet | PIX integration via POST /v1/payments. SDKs available. Idempotency via X-Idempotency-Key header. |
| **PagSeguro** | PSP | PIX payouts API for mass disbursements |
| **PagBrasil** | PSP | "Automatic Pix" with developer integration guide |
| **Boleto Bancario** | Bill payment slip | Cash-based, settles in 1-3 days |

**Mexico:**

| System | Type | Notes |
|--------|------|-------|
| **SPEI** | Real-time interbank (Banxico) | 3M+ daily transactions. Instant transfers via CLABE. |
| **CoDi** | QR-based (on SPEI) | Launched 2019. Government-promoted for financial inclusion. |
| **OXXO** | Cash payment | Pay at 20K+ convenience stores |
| **MercadoPago** | PSP | SPEI integration via Checkout API |
| **Clip** | POS/Online | Growing Mexican fintech |
| **Prometeo** | Banking API | SPEI send/receive + CoDi one-click payments |

**Relevance to ACP:** PIX's Automatico feature (recurring without per-transaction approval) is directly relevant to agent-initiated payments. The mandate-based authorization mirrors what AP2 proposes at protocol level.

---

### 1.6 Europe

**Pan-European Infrastructure:**

| System | Coverage | Notes |
|--------|----------|-------|
| **SEPA SCT Inst** | Eurozone (36 countries) | Instant credit transfers, 24/7/365. Mandatory for all euro-area banks by Oct 2025. Max EUR 100,000/txn (increasing to 200K). |
| **Wero** (EPI) | EU-wide | New pan-European instant payment wallet unifying iDEAL, Bancontact, Paylib into one interoperable network. Cross-border instant payments. |
| **SEPA Direct Debit** | Eurozone | Mandate-based recurring debits. Customer signs mandate, merchant initiates debits. |

**Country-Specific Methods:**

| Country | Method | Notes |
|---------|--------|-------|
| Netherlands | **iDEAL** | Bank redirect payment. Being absorbed into Wero. |
| Belgium | **Bancontact** | Debit card + app payment. Being absorbed into Wero. |
| Sweden | **Swish** | Mobile instant payment (10M+ users in 10M pop). API via Checkout.com, Stripe. |
| Norway | **Vipps** | Mobile payment (4M+ users). Merged with MobilePay (Denmark). |
| Germany | **Giropay** | Online banking redirect. |
| France | **Cartes Bancaires** | Dominant card scheme. |
| Spain | **Bizum** | Mobile P2P/merchant payment. Interoperable with BANCOMAT (Italy), MB WAY (Portugal). |
| UK | **Faster Payments** | Instant payment rail. Open Banking APIs via PSD2. |

**Relevance to ACP:** Europe's Wero initiative proves the market is moving toward unified payment surfaces across local methods. ACP should learn from EPI's interoperability model. SEPA Direct Debit mandates are the template for agent-authorized recurring payments.

---

### 1.7 Middle East

| Country | Provider | Notes |
|---------|----------|-------|
| Saudi Arabia | **STC Pay** | 8M+ users. SAR only. No recurring payments support. API via Tap, Amazon Payment Services, Checkout.com, Stripe. |
| Egypt | **Fawry** | 55% market share in cash payments. 20M users, 200K+ agent locations. Bill reference code system. Fawry Plus for online. |
| Egypt | **Paymob** | Developer-focused PSP for Egypt/MENA |
| Bahrain | **benefit** (BenefitPay) | National debit network + mobile wallet |
| UAE | **Apple Pay / Samsung Pay** | Growing, plus local: Payit (FAB), e-Dirham |
| Regional | **Tap Payments** | MENA PSP aggregator covering STC Pay, Fawry, cards |
| Regional | **Moyasar** | Saudi-focused developer payment infrastructure |

**Regulatory Context:**
- SAMA (Saudi Arabian Monetary Authority) compliance required
- UAE Central Bank regulations for stored value facilities
- Arabic interface support typically required

---

### 1.8 Africa

| Provider | Coverage | Focus |
|----------|----------|-------|
| **M-Pesa** (Safaricom) | Kenya, Tanzania, DRC, Mozambique, others | Pioneer mobile money. Daraja API for developers. |
| **Flutterwave** | 34 African countries | Unified API for cards, mobile money, bank transfers. Developer hub + community Slack. |
| **Paystack** (Stripe-owned) | Nigeria, Ghana, South Africa, Kenya | Developer-first APIs. Strong documentation. |
| **pawaPay** | 19 countries | Mobile money specialist. Covers 85% of African mobile money transactions. Single API + unified dashboard. |
| **Tingg** (Cellulant) | Pan-African | Collections and disbursements |
| **Kora** | Nigeria, Kenya, others | M-Pesa, Airtel Money, bank transfers |
| **Interswitch** | Nigeria | Cards, USSD, QR payments |
| **DPO Group** (Network International) | Pan-African | Multi-channel payments |

**Mobile Money Architecture:**
- Customer initiates payment via USSD or app
- Telco confirms via STK push (SIM Toolkit) or app prompt
- Settlement to merchant M-Money or bank account
- No card infrastructure needed -- works on feature phones

**Relevance to ACP:** Mobile money is the dominant payment method in most of Africa. ACP must support flows where the payer confirms on their phone (STK push/USSD), not via card tokenization. pawaPay's single API across 19 countries is a model for how ACP could abstract mobile money.

---

### 1.9 Global Aggregators

| Aggregator | Countries | Local Methods | Approach |
|------------|-----------|---------------|----------|
| **Adyen** | 200+ countries | Hundreds of payment methods | Single API, drop-in UI. `/paymentMethods` endpoint returns available methods per country. Full REST API + SDKs. |
| **Stripe** | 47+ countries | 250+ payment methods | PaymentIntents API abstracts all methods. Launched MPP for agent payments (Mar 2026). |
| **dLocal** | 40 countries (emerging markets) | 1000+ methods | "One dLocal" concept: one API, one platform, one contract. Specializes in LatAm, Africa, Asia emerging markets. Supports PIX, Boleto, OXXO, SPEI, M-Pesa, mobile money. |
| **Rapyd** | 100+ countries | Cards, wallets, bank transfers | Fintech-as-a-service. Collect, disburse, issue cards via single API. |
| **Checkout.com** | 150+ countries | Cards, local methods, crypto | Enterprise-focused. STC Pay, Swish, iDEAL support. |
| **Worldpay** (FIS) | Global | Comprehensive | Largest payment processor by volume. |
| **PayPal/Braintree** | 200+ countries | Cards, PayPal, Venmo, local methods | Braintree SDK supports local payment methods via unified API. |

**How They Handle Local Methods:**
1. **API Abstraction**: Single `POST /payments` with a `payment_method_type` field. The aggregator handles the provider-specific integration.
2. **Dynamic Method Discovery**: Adyen's `/paymentMethods` returns available methods for a given country/amount. This is critical for ACP -- agents need to discover what methods a user can pay with.
3. **Redirect/App-Switch Flows**: For wallets (GrabPay, iDEAL, etc.), the aggregator returns a redirect URL. The user completes payment in their bank/wallet app, then returns. For agents, this means async confirmation.
4. **Webhook Confirmation**: Payment confirmation is async via webhooks for most local methods (not instant like card auth).

**Relevance to ACP:** Global aggregators already solve the "one API, many rails" problem for merchants. ACP should either sit on top of these (orchestration layer) or define a standard that these aggregators can implement as providers.

---

## 2. Open Banking / PSD2 / A2A Payments

### 2.1 How Open Banking APIs Work

Open banking enables third parties (TPPs - Third Party Providers) to access bank account data and initiate payments directly from bank accounts, with customer consent.

**Two Core Services:**
1. **AIS (Account Information Services)**: Read account balances, transaction history
2. **PIS (Payment Initiation Services)**: Initiate a payment from the user's bank account to a merchant

**Payment Flow (A2A via Open Banking):**
1. User selects "Pay by Bank" at checkout
2. Merchant's TPP creates a payment initiation request
3. User is redirected to their bank's authorization page
4. User authenticates (biometrics/PIN) and approves the payment
5. Bank executes the transfer directly to merchant's account
6. TPP receives confirmation webhook

**Advantages over Card Payments:**
- Lower fees (no interchange, no card network fees)
- Instant settlement (via real-time rails like Faster Payments, SEPA Inst)
- No chargebacks (push payment, not pull)
- Strong Customer Authentication built-in

### 2.2 PSD2 in Europe

**Payment Services Directive 2** (EU regulation, effective 2018):
- Mandates banks to open APIs to licensed TPPs
- Requires Strong Customer Authentication (SCA) for electronic payments
- Two types of TPP licenses: AISP (data) and PISP (payments)
- Banks must provide free API access to licensed TPPs
- **PSD3** (proposed): Will further standardize APIs and extend to non-bank payment accounts

### 2.3 Similar Regulations Globally

| Region | Regulation/Initiative | Status |
|--------|-----------------------|--------|
| UK | Open Banking (CMA Order 2018) | Mature. 98% bank coverage. |
| Brazil | Open Finance (BCB) | Mandatory since 2021. Tied to PIX. |
| India | Account Aggregator Framework (RBI) | Live since 2021. Setu, Finvu as AAs. |
| Australia | Consumer Data Right (CDR) | Banking phase live since 2020. |
| Saudi Arabia | Saudi Open Banking (SAMA) | Framework launched 2022. |
| Nigeria | Open Banking Framework (CBN) | Guidelines issued 2021. |
| Singapore | SGFinDex | Government-backed data exchange. |
| Japan | Banking Act amendment (2017) | Voluntary API opening. |
| Mexico | Fintech Law (2018) | Open Finance provisions. |
| Canada | Consumer-Driven Banking | Framework in development. |

### 2.4 Key Open Banking API Providers

| Provider | Coverage | Notes |
|----------|----------|-------|
| **TrueLayer** | EU/UK | Connects to 95% of European bank accounts, 98% UK banks. Payment initiation + data. |
| **Tink** (Visa-owned) | EU | Access to 6,000+ banks across Europe. PIS + AIS. |
| **Yapily** | 19 countries, 2,000 banks | UK, Germany, France, Netherlands focus. |
| **Plaid** | US, UK, EU, Canada | Strongest in US (bank linking). Growing payment initiation. |
| **Finexer** | UK | UK-focused open banking payments. |
| **Noda** | EU | Open banking payments with smart routing. |
| **Token.io** | EU/UK | Enterprise open banking infrastructure. |

### 2.5 Could Agents Use Open Banking?

**Yes, with constraints:**
- **SCA Requirement**: User must authenticate with their bank for each payment. This is the main friction point for agent-initiated payments.
- **VRP (Variable Recurring Payments)**: UK is pioneering VRPs where the user grants a mandate for recurring payments within defined limits (amount, frequency). The agent could initiate payments within pre-authorized VRP mandates without per-transaction SCA.
- **SEPA Direct Debit mandates**: Once signed, merchants can pull payments without per-transaction auth -- similar to agent mandate concept.
- **Pre-authorized Payment Limits**: Some open banking frameworks allow setting spending limits that don't require per-transaction auth.

**Relevance to ACP:** Open Banking PIS + VRP mandates are the most natural fit for agent-initiated A2A payments. ACP should define a mandate format that maps to VRP parameters (max amount, frequency, merchant scope, expiry).

---

## 3. Real-Time Payment Systems by Country

### 3.1 Global RTP Landscape

| Country/Region | System | Launched | Operator | Speed | Developer Access |
|----------------|--------|----------|----------|-------|-----------------|
| India | **UPI** | 2016 | NPCI | Instant (<5s) | Via PSP APIs (Razorpay, etc.) |
| Brazil | **PIX** | 2020 | BCB (Central Bank) | Instant (<10s) | Via PSP APIs + direct participant APIs |
| USA | **FedNow** | Jul 2023 | Federal Reserve | Instant | Via participating bank APIs |
| USA | **RTP** | 2017 | The Clearing House | Instant (<15s) | Via participating bank APIs |
| UK | **Faster Payments** | 2008 | Pay.UK | Near-instant (<2hrs, typically seconds) | Via open banking APIs |
| EU | **SEPA Inst (SCT Inst)** | 2017 | EPC | <10 seconds | Via bank APIs, mandatory Oct 2025 |
| Mexico | **SPEI** | 2004 | Banxico | Instant (seconds) | Via bank APIs, Prometeo |
| Australia | **NPP (New Payments Platform)** | 2018 | NPPA | Instant | Via participating bank APIs |
| Singapore | **FAST** | 2014 | ABS | Instant | Via bank APIs |
| Thailand | **PromptPay** | 2017 | BOT | Instant | Via bank APIs |
| South Korea | **HOFINET** | 2001 | BOK | Instant | Via bank/fintech APIs |
| Japan | **Zengin** | 1973 (modernized) | JBA | Near-instant | Limited direct access |
| Nigeria | **NIP (NIBSS)** | 2011 | NIBSS | Instant | Via bank/fintech APIs |
| Saudi Arabia | **sarie** | 2021 | SAMA | Instant | Via bank APIs |
| Sweden | **BiR (Payments in Real-time)** | 2012 | Bankgirot | Instant | Via Swish APIs |

### 3.2 Developer Accessibility Comparison

**Highly Accessible (PSP/fintech API layer):**
- UPI (India) -- via Razorpay, Cashfree, etc.
- PIX (Brazil) -- via MercadoPago, PagBrasil, etc.
- Faster Payments (UK) -- via open banking TPPs
- SEPA Inst (EU) -- via TrueLayer, Tink, bank APIs

**Moderately Accessible (bank partnership required):**
- FedNow (US) -- only via participating banks, no direct developer API
- RTP (US) -- via bank partners like JP Morgan, Cross River
- SPEI (Mexico) -- via licensed institutions, Prometeo

**Limited Direct Access:**
- Alipay/WeChat Pay rails (China) -- requires Chinese entity or aggregator
- Zengin (Japan) -- via banks only

### 3.3 Settlement Comparison by Rail Type

| Rail Type | Settlement Time | Cost to Merchant | Chargeback Risk |
|-----------|----------------|------------------|-----------------|
| Card (Visa/MC) | T+1 to T+3 | 1.5-3.5% | High (180 days) |
| Real-time (UPI, PIX, FedNow) | Instant | 0-0.5% | None (push payment) |
| SEPA Credit Transfer | T+0 to T+1 | EUR 0.20-0.50 flat | None |
| SEPA Direct Debit | T+2 to T+5 | EUR 0.20-0.35 | Moderate (8 weeks/13 months) |
| Mobile Money (M-Pesa) | Instant to T+1 | 0.5-1.5% | None |
| Open Banking (A2A) | Instant (via RTP) | 0.1-0.5% | None |
| Wire/SWIFT | T+0 to T+3 | $15-50 flat | None |
| ACH (US) | T+1 to T+3 | $0.20-0.50 | Moderate (60 days) |

**Relevance to ACP:** The protocol should expose settlement time expectations to agents so they can choose appropriate rails for time-sensitive purchases. Push payments (RTP, UPI, PIX) have no chargeback risk, which simplifies the agent liability model.

---

## 4. Payment Orchestration Platforms

### 4.1 What Are Payment Orchestration Platforms?

Payment orchestration platforms (POPs) sit between merchants and multiple PSPs, providing:
- **Single API integration** to access dozens of payment providers
- **Smart routing** -- route each transaction to the PSP with highest predicted authorization rate
- **Failover** -- automatic retry with fallback PSP if primary fails
- **Cost optimization** -- route based on fee minimization
- **Unified reporting** across all providers
- **PCI-compliant vault** for storing payment credentials

### 4.2 Key Platforms

| Platform | Approach | Scale |
|----------|----------|-------|
| **Spreedly** | 120+ gateway integrations. Card vaulting + transaction routing. API-first. Supports Stripe, Adyen, Braintree, Worldpay, PayU, MercadoPago, Razorpay, iPay88. | 10+ years operating |
| **Primer** | Single integration point to dozens of PSPs. Drop-in checkout + workflow builder. Founded by ex-Braintree team. | Raised significant funding |
| **CellPoint Digital** | Travel-sector focused. 7.9M txns/hour, 99.999% uptime. USD 8B annual volume. | Enterprise/airline focus |
| **Pagos** | Payment data analytics + intelligence platform | Analytics-focused |
| **APEXX Fintech** | PSP aggregation with smart routing | Enterprise |
| **Rebilly** | Subscription + recurring payment orchestration | Recurring focus |
| **Hyperswitch** (Juspay) | **Open-source** payment orchestration in Rust. 20,000 TPS, 99.999% uptime. Supports cards, wallets, BNPL, UPI, Pay by Bank. Intelligent routing + reconciliation. Backed by $900B annual volume at Juspay. | Open-source, Apache 2.0 |

### 4.3 How Orchestration Abstraction Works

```
[Merchant/Agent]
    |
    v
[Orchestration Layer]  <-- Single API
    |
    +---> [Stripe]      (cards, wallets)
    +---> [Adyen]       (local methods EU)
    +---> [dLocal]      (LatAm, Africa)
    +---> [Razorpay]    (India)
    +---> [Flutterwave] (Africa)
    +---> [2C2P]        (Southeast Asia)
```

**Key API Pattern:**
1. `POST /payments` with amount, currency, country, preferred method
2. Orchestrator determines best PSP based on routing rules
3. PSP processes payment, returns result
4. Orchestrator normalizes response, fires webhooks

### 4.4 Could ACP Sit On Top of Orchestration?

**Yes -- this is the recommended architecture.** ACP would be the agent-facing protocol layer, and orchestration platforms would be the payment execution layer. The stack would be:

```
[AI Agent] <--> [ACP Protocol] <--> [Payment Orchestration] <--> [PSPs] <--> [Payment Rails]
```

**Hyperswitch is particularly relevant** as it's open-source (Apache 2.0), written in Rust for performance, and already supports the payment method diversity ACP needs. ACP could either:
1. Define a standard that orchestration platforms implement
2. Build on top of Hyperswitch as the reference implementation
3. Define the agent-to-orchestrator interface, letting merchants choose their orchestration layer

---

## 5. Existing Agent Payment Research

### 5.1 Active Protocols (as of March 2026)

#### 5.1.1 ACP - Agentic Commerce Protocol (OpenAI + Stripe)

- **Status**: Live in production (ChatGPT Instant Checkout)
- **License**: Apache 2.0
- **GitHub**: `agentic-commerce-protocol/agentic-commerce-protocol`
- **Governance**: OpenAI and Stripe as Founding Maintainers
- **Focus**: Checkout integration -- how agents surface products and complete purchases

**Architecture:**
- Merchant exposes structured catalog data to ChatGPT
- Agent surfaces products in conversation context
- Checkout triggers delegated payment flow

**Delegated Payment Spec:**
- OpenAI sends payment details to merchant's PSP
- PSP returns single-use payment token (scoped by max amount, expiry, merchant ID)
- Token forwarded during `complete-checkout` call
- Currently **card only** (FPAN or network token)
- Stripe's Shared Payment Token (SPT) is first compatible implementation

**Version History:**
- v1.0 (Sep 2025): Initial release
- v1.1 (Dec 2025): Fulfillment enhancements
- v1.2 (Jan 2026): Capability negotiation
- v1.3 (Jan 2026): Extensions, discounts, payment handlers

**Limitations for ACP:**
- Tightly coupled to OpenAI/ChatGPT ecosystem
- Card-only payment support (no A2A, mobile money, wallets)
- No multi-agent support
- No cross-border/multi-currency considerations
- US-only merchants initially

---

#### 5.1.2 AP2 - Agent Payments Protocol (Google)

- **Status**: Specification published, partner ecosystem building
- **License**: Apache 2.0
- **GitHub**: `google-agentic-commerce/AP2`
- **Website**: ap2-protocol.org
- **Partners**: 60+ including Mastercard, Adyen, PayPal, Coinbase, American Express, Salesforce, ServiceNow
- **Focus**: Authorization and traceability layer for agent payments

**Architecture (Role-Based):**
- **User**: The human who authorizes payments
- **User Agent / Shopping Agent (UA/SA)**: AI agent acting on behalf of user
- **Credentials Provider (CP)**: Manages payment credentials (e.g., Google Pay)
- **Merchant Endpoint (ME)**: Merchant's commerce system
- **Merchant Payment Processor (MPP)**: PSP processing the payment
- **Payment Networks/Issuers**: Card networks, banks

**Core Mandate Types:**

1. **Cart Mandate** (human-present):
   - Payer/payee identities
   - Tokenized payment method
   - Risk payload
   - Transaction details (products, amount, currency)
   - Refund conditions
   - Digitally signed by user on hardware-backed device

2. **Intent Mandate** (human-absent):
   - Pre-authorized rules: price limits, timing, conditions
   - Natural language prompt playback
   - TTL expiration
   - Allows agent to auto-generate Cart Mandates when conditions met

3. **Payment Mandate** (ecosystem visibility):
   - AI agent presence signals
   - Transaction modality (human-present vs absent)
   - Fraud prevention data

**Payment Flow (32-step sequence for human-present):**
1. User prompt -> Agent confirms intent
2. Agent discovers products via merchant
3. Merchant creates/signs Cart Mandate
4. Agent requests payment methods from Credentials Provider
5. User selects method on trusted device
6. Agent creates PaymentMandate
7. Tokenization (if needed)
8. Transaction routed: Merchant -> PSP -> Issuer
9. 3DS2/OTP challenge if needed (on trusted surface)
10. Authorization confirmed

**Key Design Principle:** "Verifiable Intent, Not Inferred Action" -- hardware-backed cryptographic signing ensures non-repudiable proof of user intent.

**V0.1 Scope:** Pull payments (cards) via tokenization
**V1.x+ Roadmap:** Push payments (real-time bank transfers, e-wallets), all payment rails

**Relevance to ACP:** AP2's mandate system is the most comprehensive authorization framework for agent payments. The Intent Mandate (human-absent, pre-authorized with constraints) is exactly what a global protocol needs. However, AP2 is currently card-focused and Google-led.

---

#### 5.1.3 MPP - Machine Payments Protocol (Stripe + Tempo)

- **Status**: Launched March 18, 2026
- **License**: Open specification (proposed to IETF)
- **Website**: mpp.dev
- **Focus**: HTTP-native machine-to-machine payments

**How It Works:**
1. Client requests a paid resource (e.g., `GET /api/data`)
2. Server returns HTTP `402 Payment Required` with payment details (RFC 7807 format)
3. Client authorizes payment, retries with payment credentials in header
4. Server verifies, processes payment, returns resource + receipt

**Payment Methods:**
- **Crypto deposits**: Direct on-chain payments via deposit addresses (Base, Polygon, Solana). Stripe handles deposit address + automatic capture on settlement.
- **SPT (Shared Payment Tokens)**: Cards, wallets, BNPL via Stripe's payment infrastructure

**Technical Details:**
- API version: `2026-03-04.preview`
- Challenge binding via secret keys (`crypto.randomBytes(32).toString('base64')`)
- Deposit address caching with 5-minute TTL
- Amount precision: 6 decimals for crypto, 2 for fiat (cents)
- Error format: RFC 7807 problem details

**Visa Partnership:** Visa supports MPP by enabling card-based payments for trusted autonomous agent payments via Visa Acceptance Platform.

**Relevance to ACP:** MPP is the simplest protocol (HTTP 402 flow) but currently tied to Stripe + crypto rails. Its IETF submission signals intent to standardize. The HTTP 402 pattern is elegant but only works for API/resource access, not physical commerce or complex checkout flows.

---

#### 5.1.4 x402 Protocol (Coinbase + Cloudflare)

- **Status**: Live, x402 Foundation established (Sep 2025)
- **License**: Open source
- **GitHub**: `coinbase/x402`
- **Website**: x402.org
- **Whitepaper**: x402.org/x402-whitepaper.pdf
- **Focus**: HTTP-native micropayments using stablecoins

**How It Works:**
1. Client requests x402-enabled resource
2. Server returns `402 Payment Required` with price, acceptable tokens
3. Client sends signed payment payload (e.g., USDC) via HTTP header
4. Server verifies via local check or facilitator `/verify` endpoint
5. Resource returned

**Infrastructure:**
- Coinbase-hosted facilitator for ERC-20 payments on Base, Polygon, Solana
- Free tier: 1,000 transactions/month
- Cloudflare integration for edge-level payment gates
- Works with Coinbase Agentic Wallets

**Use Cases:**
- Pay-per-API call (micropayments)
- API paywalls
- Machine-to-machine payments
- Programmatic resource access

**Relevance to ACP:** x402 proves the HTTP 402 pattern works for micropayments but is crypto-only (USDC). ACP could adopt the same HTTP pattern but extend it to fiat rails.

---

#### 5.1.5 Visa Trusted Agent Protocol (TAP)

- **Status**: Available in Visa Developer Center and GitHub (Oct 2025)
- **Partners**: Cloudflare, Adyen, Ant International, Checkout.com, Coinbase, CyberSource, Fiserv, Microsoft, Nuvei, Shopify, Stripe, Worldpay
- **Focus**: Trust establishment between AI agents and merchants

**Technical Approach:**
- Built on **HTTP Message Signature standard**
- Aligned with **Web Bot Auth** (Cloudflare)
- Agent-specific cryptographic signatures
- "Agent Intent" signals: trusted agent status, intent to retrieve details or purchase

**How It Works:**
1. Agent signs HTTP requests with cryptographic key
2. Merchant verifies agent identity and trust level
3. Agent Intent header indicates purpose (browse, purchase)
4. Existing payment infrastructure handles the actual payment

**Cloudflare Collaboration:**
- Cloudflare's Web Bot Auth integrates with Visa TAP, Mastercard Agent Pay, and American Express agentic commerce
- Edge-level agent identity verification

**Relevance to ACP:** TAP solves the trust/identity layer, not the payment layer. ACP could incorporate TAP-compatible agent identity verification while handling the payment orchestration separately.

---

#### 5.1.6 Alipay Agentic Commerce Trust Protocol

- **Status**: Live (Jan 2026), 120M+ weekly AI transactions
- **MCP Server**: `alipay/global-alipayplus-mcp` on GitHub
- **Focus**: AI agent payment execution within Alipay ecosystem

**Real-World Deployments:**
- Luckin Coffee: Voice-commanded ordering + payment via AI assistant
- Rokid smart glasses: Wearable AI payments via MCP server
- Taobao Instant Commerce: AI-led purchases

**Relevance to ACP:** Alipay proves the model works at massive scale (120M/week). However, it's ecosystem-locked. ACP should be what Alipay's protocol is, but open and cross-ecosystem.

---

### 5.2 W3C Web Payments Specifications

| Spec | Status | Relevance |
|------|--------|-----------|
| **Payment Request API** | Candidate Recommendation | Standardizes browser-level payment flow between merchant, user agent (browser), and payment method. Designed for human users, not AI agents. |
| **Payment Handler API** | Working Draft | Allows web apps to handle payment requests. Could theoretically be extended for agent payment handlers. |
| **Payment Method Identifiers** | Recommendation | Standardized strings identifying payment methods (e.g., `basic-card`, `https://example.com/pay`). ACP could adopt this pattern for payment method identification. |

**Assessment:** W3C Web Payments was designed for browser-based human checkout, not machine-to-machine payments. The Payment Method Identifier pattern is reusable, but the overall architecture doesn't fit agent-initiated payments.

### 5.3 ISO 20022

**What It Is:** International standard for electronic data interchange between financial institutions. XML/JSON-based message format for payment instructions, reporting, and securities.

**Message Types Relevant to ACP:**
- `pain.001` -- Customer Credit Transfer Initiation
- `pain.002` -- Payment Status Report
- `pain.008` -- Customer Direct Debit Initiation
- `camt.053` -- Bank-to-Customer Statement

**Key Properties:**
- Machine-readable XML tags for every data element
- Supports both XML and JSON formats
- Publicly available schemas on MyStandards
- Being adopted by SWIFT (replacing MT messages), FedNow, SEPA, and most modern RTP systems

**Relevance to ACP:** ISO 20022 message structures could be used as the canonical payment instruction format within ACP. The `pain.001` (credit transfer initiation) and `pain.008` (direct debit initiation) are directly applicable. Using ISO 20022 would ensure compatibility with bank-to-bank settlement systems worldwide.

### 5.4 Competitive Landscape Summary

```
                    [Trust/Identity Layer]
                    Visa TAP | Web Bot Auth | Cloudflare
                            |
                    [Agent Protocol Layer]
            AP2 (Google) | ACP (OpenAI/Stripe) | MPP (Stripe/Tempo) | x402 (Coinbase)
                            |
                    [Payment Orchestration Layer]
            Hyperswitch | Spreedly | Primer | CellPoint
                            |
                    [PSP Layer]
            Stripe | Adyen | dLocal | Razorpay | Flutterwave | etc.
                            |
                    [Payment Rail Layer]
            UPI | PIX | SEPA | FedNow | M-Pesa | Cards | Crypto | etc.
```

**Gap Analysis -- What's Missing:**
1. **No protocol supports ALL payment rails** -- ACP is card-only, MPP/x402 are card+crypto, AP2 is card with roadmap to expand
2. **No protocol is truly rail-agnostic** -- all have preferred rails or ecosystem lock-in
3. **No protocol handles mobile money** -- Africa's dominant payment method is unsupported
4. **No protocol handles mandate diversity** -- UPI AutoPay, SEPA DD, VRP, PIX Automatico all have different mandate structures but similar concepts
5. **No protocol addresses FX natively** -- cross-border agent payments need currency conversion awareness
6. **No protocol handles regulatory diversity** -- KYC/AML requirements vary by jurisdiction

---

## 6. Authentication & Authorization for Agent Payments

### 6.1 IETF OAuth 2.0 Extension for AI Agents

**Draft:** `draft-oauth-ai-agents-on-behalf-of-user` (v02, Aug 2025)
**Authors:** T. S. Senarath, A. Dissanayaka (WSO2)
**Status:** Informational draft

**New Parameters:**
- `requested_agent` (authorization endpoint): Unique identifier of the agent requesting delegation
- `agent_token` (token endpoint): Agent's own authentication credential containing `sub` claim

**Grant Type:** `urn:ietf:params:oauth:grant-type:agent-authorization_code`

**Flow:**
1. Client directs user to auth endpoint with `requested_agent` parameter
2. Auth server shows consent screen identifying the agent and requested scopes
3. User approves, auth server issues authorization code bound to (user, client, agent)
4. Agent exchanges code + agent_token for delegated access token
5. Resulting JWT contains: `sub` (user), `act.sub` (agent), `azp` (client app)

**Security Requirements:**
- PKCE mandatory
- Authorization codes single-use with short expiry
- Consent screen must clearly identify agent identity and scopes
- Token revocation on: user consent revocation, agent token revocation, account/agent disabling

**Related Draft:** `draft-oauth-ai-agents-on-behalf-of-user` also has a companion draft for multi-agent collaboration: `draft-song-oauth-ai-agent-collaborate-authz` and a separate AAuth (Agentic Authorization) extension: `draft-rosenberg-oauth-aauth`.

### 6.2 Delegated Payment Authority Models

| Model | How It Works | Examples |
|-------|-------------|----------|
| **Pre-authorized Mandate** | User sets rules (max amount, frequency, merchant scope, expiry). Agent operates within rules without per-transaction auth. | AP2 Intent Mandate, UPI AutoPay, SEPA DD, VRP |
| **Per-Transaction Approval** | Agent prepares payment, user must approve each one. | Standard card checkout, SCA |
| **Token-Scoped Delegation** | User saves payment method, agent gets single-use or scoped tokens. | OpenAI ACP Delegated Payment, Stripe SPT |
| **Wallet-Based** | Agent requests payment, user confirms in their wallet app (async). | M-Pesa STK push, Alipay/WeChat confirm |
| **Smart Contract** | Pre-programmed rules enforced on-chain. Agent triggers contract execution. | x402, crypto escrow |

### 6.3 Pre-authorized Mandates by Region

| Region | Mechanism | Mandate Parameters |
|--------|-----------|-------------------|
| India | UPI AutoPay e-mandate | Max amount per debit, frequency (daily/weekly/monthly/yearly), validity period, pause/revoke anytime |
| EU | SEPA Direct Debit mandate | Creditor ID, mandate reference, one-off or recurrent, no amount limit (but contestable 8 weeks) |
| UK | Variable Recurring Payment (VRP) | Max per-payment amount, max cumulative amount, frequency, specific payee, expiry date |
| Brazil | PIX Automatico (Jun 2025) | Authorized payee, amount, frequency, no per-transaction approval |
| Global (cards) | Card-on-file tokenization | Merchant-scoped token, recurring flag, no per-transaction CVV |
| Global (crypto) | Smart contract allowance | Token contract, spender address, max amount |

### 6.4 Recommended Authorization Architecture for ACP

```
[User]
  |
  |--> [Register Agent] -- OAuth 2.0 + agent extension
  |     |
  |     +--> Agent receives delegated token with:
  |           - agent_id (who is acting)
  |           - user_id (on whose behalf)
  |           - scopes (what actions allowed)
  |           - mandate constraints:
  |               - max_per_transaction (ISO 4217 amount)
  |               - max_cumulative (rolling period)
  |               - allowed_merchants (list or wildcard)
  |               - allowed_currencies
  |               - allowed_payment_methods
  |               - valid_from / valid_until
  |               - require_human_approval_above (threshold)
  |
  |--> [Agent Payment Request]
  |     |
  |     +--> ACP middleware checks:
  |           1. Token valid and not revoked?
  |           2. Within mandate constraints?
  |           3. If yes: execute via payment orchestration
  |           4. If no: request user approval (redirect/push)
  |           5. Log all decisions for audit trail
```

---

## 7. Key Technical Considerations

### 7.1 Idempotency in Payment APIs

**Why It Matters for Agents:** AI agents may retry requests due to timeouts, network issues, or multi-step reasoning. Without idempotency, retries can create duplicate payments.

**Best Practices:**
- Every mutating API call must include an idempotency key
- Key should be derived from the business operation (e.g., `order_id + payment_attempt`), not randomly generated
- Server stores key -> result mapping; returns cached result on duplicate
- TTL for idempotency records: typically 24-48 hours
- Webhook handlers must also be idempotent

**How Existing APIs Handle It:**
- Stripe: `Idempotency-Key` header
- Adyen: `X-Idempotency-Key` header (optional)
- MercadoPago: `X-Idempotency-Key` header
- Razorpay: Built into order creation flow

**ACP Recommendation:** Mandate `Idempotency-Key` header on all payment initiation requests. Define key generation rules: `{agent_id}:{user_id}:{intent_hash}:{attempt_number}`.

### 7.2 Currency Handling (ISO 4217)

**Standard:** ISO 4217 defines 3-letter alpha codes (USD, EUR, BRL) and numeric codes (840, 978, 986) for currencies.

**Minor Units (Critical for APIs):**
| Currency | Code | Minor Units | Example |
|----------|------|-------------|---------|
| USD | 840 | 2 | $10.50 = 1050 cents |
| EUR | 978 | 2 | EUR 10.50 = 1050 cents |
| JPY | 392 | 0 | JPY 1050 = 1050 (no decimals) |
| BHD | 048 | 3 | BHD 10.500 = 10500 fils |
| USDC (crypto) | N/A | 6 | 10.500000 = 10500000 |

**ACP Recommendation:**
- All amounts as integers in smallest unit (cents, paise, etc.)
- Currency as ISO 4217 alpha-3 code
- Include `minor_units` field in currency metadata
- For crypto: use 6-decimal precision per convention
- Never use floating-point for money

### 7.3 FX / Cross-Border Considerations

**Challenges:**
- Exchange rates fluctuate between quote and settlement
- Spread/markup varies by provider (0.5% to 3%+)
- Some countries have capital controls (Nigeria, China, India)
- Repatriation restrictions in some markets
- Dual exchange rates in some countries (official vs. parallel)

**How Aggregators Handle FX:**
- **Adyen**: FX reference rate + management fee (spread). Consolidates multi-currency settlements.
- **dLocal**: Handles local currency collection + FX + cross-border settlement in one flow
- **Wise (TransferWise)**: Mid-market rate + transparent flat fee
- **Rapyd**: FX built into collect/disburse API

**ACP Recommendation:**
- Separate `display_currency` (what user sees) from `settlement_currency` (what merchant receives)
- Include FX rate, spread, and total cost in payment response
- Allow agents to compare FX costs across providers
- Support "lock rate" for time-limited quotes

### 7.4 Settlement Times by Rail

| Rail | Settlement to Merchant | Notes |
|------|----------------------|-------|
| UPI | T+0 to T+1 | Instant credit to merchant bank |
| PIX | Instant | Real-time settlement |
| FedNow | Instant | Real-time settlement |
| SEPA Inst | <10 seconds | Real-time |
| Cards (Visa/MC) | T+1 to T+3 | Batch settlement |
| SEPA Credit Transfer | T+1 | Next business day |
| SEPA Direct Debit | T+2 to T+5 | Plus dispute window |
| ACH (US) | T+1 to T+3 | Batch processing |
| M-Pesa | T+0 to T+1 | Varies by integration |
| SWIFT wire | T+0 to T+3 | Depends on correspondent banking |
| Crypto (on-chain) | Minutes to 1 hour | Depends on chain/confirmation requirements |

### 7.5 Compliance Requirements

#### PCI DSS (Payment Card Industry Data Security Standard)
- Applies to anyone storing, processing, or transmitting cardholder data
- **Tokenization** is the primary strategy: swap PAN for non-sensitive token
- Using PSP client-side libraries (Stripe.js, Adyen Drop-in) keeps merchants out of PCI scope
- ACP should NEVER handle raw card data -- always use tokens from PSPs

#### PSD2 SCA (Strong Customer Authentication)
- Required for electronic payments in EU/EEA
- Two of three factors: knowledge (PIN), possession (phone), inherence (biometric)
- **Exemptions** relevant to agents:
  - Recurring payments (after first auth)
  - Merchant-initiated transactions (MIT)
  - Low-value transactions (<EUR 30, up to 5 consecutive or EUR 100 cumulative)
  - Trusted beneficiary list
- ACP should track SCA exemption eligibility per transaction

#### KYC/AML (Know Your Customer / Anti-Money Laundering)

**Regional Variation:**

| Region | Key Requirements |
|--------|-----------------|
| US | BSA, PATRIOT Act. SSN verification, OFAC screening |
| EU | AMLD6, eIDAS. ID verification, UBO registers, transaction monitoring |
| UK | MLR 2017. Risk-based approach, PEP screening |
| India | RBI KYC. Aadhaar/PAN verification, video KYC |
| Singapore | MAS Notice 626. Risk-based, enhanced for high-risk |
| Nigeria | CBN KYC. BVN verification, tiered KYC |
| Saudi Arabia | SAMA AML/CTF. National ID verification |
| Brazil | BCB Circular 3.978. CPF verification |

**Compliance Architecture for ACP:**
- ACP itself should NOT perform KYC -- delegate to PSPs/payment providers
- Each PSP handles KYC for their jurisdiction
- ACP should carry KYC verification status signals (verified/unverified/level)
- Transaction limits should be configurable per jurisdiction
- Sanctions screening (OFAC, EU, UN lists) should be PSP responsibility

#### Data Privacy
| Region | Law | Impact on ACP |
|--------|-----|---------------|
| EU | GDPR | Minimal PII in protocol messages. Data residency requirements. Right to deletion. |
| US | State laws (CCPA, etc.) | Disclosure requirements for data usage |
| Brazil | LGPD | Similar to GDPR |
| India | DPDP Act 2023 | Data localization for certain payment data |
| China | PIPL | Strict data localization. Cross-border transfer restrictions. |

### 7.6 How ACP Should Handle Regulatory Differences

**Jurisdiction-Aware Architecture:**

```
[ACP Core Protocol]
    |
    +---> [Jurisdiction Module: US]
    |       - PCI DSS scope management
    |       - BSA/PATRIOT compliance signals
    |       - FedNow/RTP rail selection
    |       - USD settlement
    |
    +---> [Jurisdiction Module: EU]
    |       - PSD2 SCA enforcement + exemptions
    |       - GDPR data minimization
    |       - SEPA rail selection
    |       - EUR settlement
    |
    +---> [Jurisdiction Module: India]
    |       - RBI data localization
    |       - UPI mandate compliance
    |       - INR settlement only for domestic
    |
    +---> [Jurisdiction Module: Brazil]
    |       - BCB PIX integration rules
    |       - LGPD compliance
    |       - BRL settlement
    |
    +---> [Jurisdiction Module: Africa]
            - Mobile money regulatory requirements
            - Per-country telco licensing
            - Multi-currency settlement
```

**Key Design Principles:**
1. **Protocol Core is jurisdiction-neutral**: Message format, agent identity, mandate structure
2. **Jurisdiction modules are pluggable**: Each module defines local constraints, required fields, compliance checks
3. **PSPs handle the actual compliance**: ACP carries signals, PSPs enforce rules
4. **Fail closed**: If jurisdiction requirements are unclear, require human approval

---

## Summary: Strategic Positioning for ACP

### What Exists Today (March 2026)

| Protocol | Owner | Rail Support | Scope | Status |
|----------|-------|-------------|-------|--------|
| ACP | OpenAI/Stripe | Cards only | Checkout | Live (US) |
| AP2 | Google | Cards (roadmap: all) | Auth/Trust | Spec published |
| MPP | Stripe/Tempo | Cards + Crypto | HTTP payments | Live (Mar 2026) |
| x402 | Coinbase/Cloudflare | Crypto (USDC) | Micropayments | Live |
| TAP | Visa/Cloudflare | N/A (trust layer) | Agent Identity | Live |
| Alipay Trust | Ant Group | Alipay ecosystem | Full commerce | Live (China) |

### The Gap ACP Should Fill

**A truly global, rail-agnostic, open-source middleware protocol that:**

1. **Supports ALL payment rails** -- not just cards and crypto, but UPI, PIX, SEPA, M-Pesa, mobile money, A2A, and every local payment method
2. **Defines a universal mandate format** -- mapping to UPI AutoPay, SEPA DD, VRP, PIX Automatico, card-on-file, and smart contract allowances
3. **Is not owned by any single company** -- unlike ACP (OpenAI/Stripe), AP2 (Google), x402 (Coinbase)
4. **Handles FX and cross-border natively** -- with rate transparency, currency conversion, and settlement currency management
5. **Is jurisdiction-aware** -- pluggable compliance modules for KYC/AML, SCA, data localization
6. **Supports the full spectrum of agent-payment interactions** -- from micropayments (x402-like) to complex multi-item checkout (ACP-like) to pre-authorized recurring (AP2 Intent Mandate-like)
7. **Sits at the orchestration layer** -- not tied to any single PSP, using platforms like Hyperswitch as reference implementation
8. **Uses existing standards** -- ISO 20022 message formats, ISO 4217 currencies, OAuth 2.0 agent extensions, HTTP Message Signatures, W3C Payment Method Identifiers

---

## Sources

### Asia
- [7 Best Payment Gateway APIs for India 2026](https://www.purshology.com/2026/03/7-best-payment-gateway-apis-for-web-developers-in-india-2026-guide/)
- [Razorpay UPI Payment Gateway](https://razorpay.com/upi-payment-gateway-india/)
- [Razorpay UPI Payment API Guide](https://razorpay.com/blog/upi-payment-api-guide)
- [UPI APIs by Decentro](https://decentro.tech/resources/upi-apis)
- [Alipay vs WeChat Pay vs UnionPay 2025](https://eggplantdigital.cn/alipay-vs-wechat-pay-vs-unionpay-your-chinese-payment-systems-guide-in-2025/)
- [5 Best Payment Gateways in China 2026](https://statrys.com/blog/best-payment-gateways-in-china)
- [AlphaPay: WeChat Pay, AliPay, UnionPay Integration](https://www.alphapay.com/how-to-add-wechat-pay-alipay-and-unionpay-to-your-website/)
- [SEA Payment Methods by Country (KOMOJU)](https://en.komoju.com/blog/payment-method/southeast-asia/)
- [GrabPay via Adyen](https://docs.adyen.com/payment-methods/grabpay/api-only)
- [GoPay Integration](https://www.gopay.com/en/integration/)
- [PayPay for Developers](https://blog.paypay.ne.jp/en/release-of-paypay-for-developers/)
- [Stripe Japan PayPay Integration 2025](https://stripe.com/newsroom/news/japan-payments-moment-2025)

### Latin America
- [PIX via MercadoPago](https://www.mercadopago.com.br/developers/en/docs/checkout-api-payments/integration-configuration/integrate-pix)
- [PagBrasil Automatic Pix Integration Guide](https://www.pagbrasil.com/blog/pix/pagbrasils-automatic-pix-integration-guide-for-developers/)
- [PIX Real-Time Payments (Checkout.com)](https://www.checkout.com/blog/what-is-pix-payment-system-real-time-payments-in-brazil)
- [Brazil PIX Cross-Border Future](https://thepaypers.com/payments/thought-leader-insights/brazils-instant-payment-pix-and-the-future-of-cross-border-payouts)
- [SPEI/CoDi Mexico Payments (Finofo)](https://www.finofo.com/blog/spei-codi-how-payments-work-in-mexico)
- [Mexico Payment Rails (Transfi)](https://www.transfi.com/blog/mexicos-payment-rails-how-they-work---inside-spei-codi-the-digital-shift-in-latin-americas-fintech-hub)
- [MercadoPago SPEI Integration](https://www.mercadopago.com.mx/developers/en/docs/checkout-api/payment-integration/spei-transfers)
- [Latin America Payments 2025](https://paymentspedia.com/payment-methods-of-the-world-2025/latam-payments-ecosystem-2025/)

### Europe
- [SEPA Instant Payments (Volante)](https://www.volantetech.com/sepa-instant-unlocking-real-time-payments/)
- [Wero / European Payments Initiative](https://en.wikipedia.org/wiki/European_Payments_Initiative)
- [EU Instant Payments Regulation (Nordea)](https://www.nordea.com/en/news/instant-payments-regulation-get-ready-for-fully-automated-instant-payments-and-payee-verification)
- [Europe Local Payment Methods (Fasto)](https://fasto.co/top-local-payment-methods-in-europe-2025/)
- [EMPSA Interoperability (BANCOMAT, BIZUM, MB WAY)](https://empsa.org/news/leading-european-mobile-payment-solutions-bancomat-bizum-and-mb-way-pioneer-interoperability-launching-first-instant-transactions/)
- [Swish API (Checkout.com)](https://www.checkout.com/docs/payments/add-payment-methods/swish/api-only)
- [Stripe SEPA Direct Debit](https://docs.stripe.com/payments/sepa-debit)

### Middle East
- [STC Pay (Tap Developers)](https://developers.tap.company/docs/stcpay)
- [STC Pay (Checkout.com)](https://www.checkout.com/payment-methods/stc-pay)
- [STC Pay API (Amazon Payment Services)](https://paymentservices.amazon.com/docs/api/payment-methods/stc-pay)
- [MENA Payment Methods (Xsolla)](https://xsolla.com/blog/more-payment-methods-in-mena)
- [15 Best Payment Gateways Middle East](https://infinmobile.com/15-best-payment-gateways-middle-east/)

### Africa
- [Payment APIs in Africa (Finance in Africa)](https://financeinafrica.com/insights/apis-africas-developers-money-code/)
- [Top 10 Payment Gateways Africa (FinHive)](https://finhive.africa/top-10-payment-gateways-driving-africas-digital-payments-growth/)
- [M-Pesa via Flutterwave](https://developer.flutterwave.com/v3.0/docs/m-pesa)
- [Top Payment Gateways Africa (ElemiTech)](https://www.elemitech.com/post/top-payment-gateways-in-africa)
- [B2B Payment Processing Africa (Kora)](https://www.korahq.com/blog/top-7-b2b-payment-processing-companies-in-africa)

### Global Aggregators
- [Adyen Payment Methods](https://docs.adyen.com/payment-methods)
- [Adyen API Explorer](https://docs.adyen.com/api-explorer/)
- [dLocal Global Payments](https://www.dlocal.com/)
- [dLocal Emerging Markets Handbook](https://www.dlocal.com/resources/emerging-markets-payments-handbook/)
- [Rapyd Global Payments](https://www.rapyd.net/)

### Open Banking
- [Plaid vs Tink vs TrueLayer Comparison 2026](https://www.fintegrationfs.com/post/plaid-vs-tink-vs-truelayer-which-open-banking-api-is-best-for-your-fintech)
- [Best Open Banking API Providers 2026](https://itexus.com/best-open-banking-api-providers/)
- [TrueLayer Alternatives (Yapily)](https://www.yapily.com/blog/5-truelayer-alternatives-for-open-banking/)
- [Top Open Banking Providers (DashDevs)](https://dashdevs.com/blog/open-banking-providers/)

### Real-Time Payments
- [Comparing RTP: PIX, UPI, FedNow](https://paymentscmi.com/insights/comparing-pix-upi-fednow/)
- [A2A Payment Experience: FedNow vs PIX vs UPI (Hyperswitch)](https://hyperswitch.io/blog/a2a-payment-experience-fednow-vs-pix-vs-upi)
- [Real-Time Payments API Guide (Lightspark)](https://www.lightspark.com/knowledge/real-time-payments-api-instant-global-transactions)

### Payment Orchestration
- [Spreedly Payments Orchestration Guide](https://www.spreedly.com/guides/payments-orchestration)
- [Spreedly Review (Fintech Review)](https://fintechreview.net/spreedly-review/)
- [Primer (Sacra)](https://sacra.com/c/primer/)
- [CellPoint Digital Payment Orchestration](https://cellpointdigital.com/solutions/payment-orchestration)
- [Hyperswitch Open Source (Juspay)](https://hyperswitch.io/)
- [Hyperswitch GitHub](https://github.com/juspay/hyperswitch)

### Agent Payment Protocols
- [AP2 Announcement (Google Cloud Blog)](https://cloud.google.com/blog/products/ai-machine-learning/announcing-agents-to-payments-ap2-protocol)
- [AP2 Specification](https://ap2-protocol.org/specification/)
- [AP2 GitHub](https://github.com/google-agentic-commerce/AP2)
- [OpenAI ACP Delegated Payment Spec](https://developers.openai.com/commerce/specs/payment)
- [ACP GitHub](https://github.com/agentic-commerce-protocol/agentic-commerce-protocol)
- [OpenAI Buy It in ChatGPT](https://openai.com/index/buy-it-in-chatgpt/)
- [MPP Overview (mpp.dev)](https://mpp.dev/overview)
- [Stripe MPP Documentation](https://docs.stripe.com/payments/machine/mpp)
- [Stripe MPP Blog Post](https://stripe.com/blog/machine-payments-protocol)
- [x402 GitHub (Coinbase)](https://github.com/coinbase/x402)
- [x402 Documentation (Coinbase)](https://docs.cdp.coinbase.com/x402/welcome)
- [x402 Whitepaper](https://www.x402.org/x402-whitepaper.pdf)
- [x402 Foundation (Cloudflare)](https://blog.cloudflare.com/x402/)
- [Visa Trusted Agent Protocol Announcement](https://investor.visa.com/news/news-details/2025/Visa-Introduces-Trusted-Agent-Protocol-An-Ecosystem-Led-Framework-for-AI-Commerce/default.aspx)
- [Cloudflare Agentic Commerce](https://blog.cloudflare.com/secure-agentic-commerce/)
- [Agentic Payments Explained: ACP, AP2, x402 (Orium)](https://orium.com/blog/agentic-payments-acp-ap2-x402)
- [Alipay MCP Server (GitHub)](https://github.com/alipay/global-alipayplus-mcp)
- [Alipay 120M AI Transactions](https://www.businesswire.com/news/home/20260213770962/en/Alipay-AI-Payment-Exceeds-120-Million-Transactions-in-One-Week-as-Agentic-Commerce-Accelerates-in-China)

### Authentication & Authorization
- [IETF OAuth 2.0 AI Agent Extension (Draft-02)](https://datatracker.ietf.org/doc/draft-oauth-ai-agents-on-behalf-of-user/)
- [OAuth 2.0 AI Agent Extension (Draft-00)](https://datatracker.ietf.org/doc/html/draft-oauth-ai-agents-on-behalf-of-user-00)
- [AAuth - Agentic Authorization Extension](https://www.ietf.org/archive/id/draft-rosenberg-oauth-aauth-00.html)
- [Explaining OAuth OBO for AI Agents (ceposta)](https://blog.christianposta.com/explaining-on-behalf-of-for-ai-agents/)
- [Delegated Agent Access (Scalekit)](https://www.scalekit.com/blog/delegated-agent-access)

### Technical Standards
- [ISO 20022 (Wikipedia)](https://en.wikipedia.org/wiki/ISO_20022)
- [ISO 20022 Message Definitions](https://www.iso20022.org/iso-20022-message-definitions)
- [ISO 20022 (SWIFT)](https://www.swift.com/standards/iso-20022/iso-20022-standards)
- [W3C Payment Request API](https://www.w3.org/TR/payment-request/)
- [W3C Payment Handler API](https://www.w3.org/TR/payment-handler/)
- [ISO 4217 Currency Codes](https://www.iso.org/iso-4217-currency-codes.html)
- [Adyen Currency Codes and Minor Units](https://docs.adyen.com/development-resources/currency-codes)

### Compliance
- [Global KYC Regulations Guide (KYCAID)](https://kycaid.com/blog/a-guide-to-global-kyc-regulations-key-differences-by-region/)
- [KYC AML Cross-Border Guide (Phoenix Strategy)](https://www.phoenixstrategy.group/blog/aml-kyc-cross-border-transactions-guide)
- [Multi-Jurisdiction KYC (FinLego)](https://finlego.com/blog/how-to-handle-region-specific-kyc-and-aml)
- [Idempotent Payment API Design](https://medium.com/codeelevation/how-to-design-idempotent-payment-apis-for-reliable-financial-transactions-24513f6420ae)
- [Payment Gateway Integration Guide 2026](https://neontri.com/blog/payment-gateway-integration/)

### Recurring Payments
- [UPI AutoPay (NPCI)](https://www.npci.org.in/what-we-do/autopay/product-overview)
- [UPI AutoPay API Explained (Castler)](https://castler.com/learning-hub/upi-autopay-api-explained)
- [Razorpay UPI AutoPay](https://razorpay.com/upi-autopay/)
