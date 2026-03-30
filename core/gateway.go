package core

import "context"

// GatewayInterface defines the subset of Gateway methods used by wrappers
// such as audit logging and rate limiting.
type GatewayInterface interface {
	BuildPaymentRequired(resource Resource, price Price) (*PaymentRequired, error)
	Verify(ctx context.Context, payload PaymentPayload) (*VerifyResponse, error)
	Settle(ctx context.Context, payload PaymentPayload) (*SettleResponse, error)
	Methods() []string
	Method(name string) (Method, bool)
}
