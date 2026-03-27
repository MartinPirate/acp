package core

import (
	"context"
	"encoding/json"
)

// Method is a concrete payment rail implementation.
// Each method knows how to build payment options, create payloads,
// verify payments, and settle on its specific rail.
type Method interface {
	// Name returns the method identifier (e.g., "card", "upi", "mock").
	Name() string

	// SupportedIntents returns which intents this method can handle.
	SupportedIntents() []Intent

	// SupportedCurrencies returns which currencies this method supports.
	SupportedCurrencies() []Currency

	// BuildOption constructs a PaymentOption for the 402 response.
	// Returns an error if the method cannot handle the given intent or currency.
	BuildOption(intent Intent, price Price) (PaymentOption, error)

	// CreatePayload builds the method-specific payload for a client payment.
	// Called by the client after selecting a payment option.
	CreatePayload(ctx context.Context, option PaymentOption) (json.RawMessage, error)

	// Verify checks if a payment payload is valid without settling.
	Verify(ctx context.Context, payload PaymentPayload, option PaymentOption) (*VerifyResponse, error)

	// Settle executes the payment on this rail.
	Settle(ctx context.Context, payload PaymentPayload, option PaymentOption) (*SettleResponse, error)
}

// SupportsIntent checks if a method supports a given intent.
func SupportsIntent(m Method, intent Intent) bool {
	for _, i := range m.SupportedIntents() {
		if i == intent {
			return true
		}
	}
	return false
}

// SupportsCurrency checks if a method supports a given currency.
func SupportsCurrency(m Method, currency Currency) bool {
	for _, c := range m.SupportedCurrencies() {
		if c == currency {
			return true
		}
	}
	return false
}
