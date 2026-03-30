// Package core defines the foundational types, interfaces, errors, and utilities
// for the Agentic Commerce Protocol (ACP).
//
// This package is imported by all other ACP packages and provides:
//
//   - Protocol types: [PaymentRequired], [PaymentPayload], [PaymentOption],
//     [VerifyResponse], [SettleResponse], [Resource], and [Price].
//   - The [Method] interface that every payment rail must implement.
//   - The [Facilitator] interface for remote verify/settle services.
//   - The [Intent] type with values [IntentCharge], [IntentAuthorize],
//     [IntentSubscribe], and [IntentMandate].
//   - The [Currency] type with ISO 4217 codes and crypto token symbols.
//   - The [Budget] and [BudgetEnforcer] for client-side spending limits.
//   - Structured errors via [PaymentError] and [ErrorCode] constants.
//   - Shared helpers: [ValidateBuildOption], [BuildSettleResponse],
//     [GenerateTxnID], [UnmarshalMethodPayload], and [IdempotencyKey].
//
// # Implementing a Payment Method
//
// Every payment rail implements [Method]:
//
//	type MyMethod struct{ config Config }
//
//	func (m *MyMethod) Name() string                    { return "my-rail" }
//	func (m *MyMethod) SupportedIntents() []Intent      { return []Intent{IntentCharge} }
//	func (m *MyMethod) SupportedCurrencies() []Currency { return []Currency{USD, EUR} }
//
//	func (m *MyMethod) BuildOption(intent Intent, price Price) (PaymentOption, error) {
//	    if err := ValidateBuildOption("my-rail", intent, price.Currency,
//	        m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
//	        return PaymentOption{}, err
//	    }
//	    return PaymentOption{Intent: intent, Method: "my-rail",
//	        Currency: price.Currency, Amount: price.Amount}, nil
//	}
//
//	func (m *MyMethod) CreatePayload(ctx context.Context, opt PaymentOption) (json.RawMessage, error) { ... }
//	func (m *MyMethod) Verify(ctx context.Context, p PaymentPayload, o PaymentOption) (*VerifyResponse, error) { ... }
//	func (m *MyMethod) Settle(ctx context.Context, p PaymentPayload, o PaymentOption) (*SettleResponse, error) { ... }
package core
