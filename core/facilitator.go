package core

import "context"

// Facilitator is a remote service that verifies and settles payments.
type Facilitator interface {
	Verify(ctx context.Context, payload PaymentPayload, requirements PaymentRequired) (*VerifyResponse, error)
	Settle(ctx context.Context, payload PaymentPayload, requirements PaymentRequired) (*SettleResponse, error)
	Supported(ctx context.Context) (*SupportedResponse, error)
}

// SupportedResponse declares what a facilitator handles.
type SupportedResponse struct {
	Methods    []string   `json:"methods"`
	Intents    []Intent   `json:"intents"`
	Currencies []Currency `json:"currencies"`
}
