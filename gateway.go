// Package acp implements the Agentic Commerce Protocol — a universal middleware
// that enables AI agents to request and make payments through any payment provider.
package acp

import (
	"context"
	"fmt"
	"sync"

	"github.com/paideia-ai/acp/core"
)

// Re-export core types at package level for ergonomic API.
type (
	Price           = core.Price
	Budget          = core.Budget
	BudgetEnforcer  = core.BudgetEnforcer
	PaymentRequired = core.PaymentRequired
	PaymentPayload  = core.PaymentPayload
	PaymentOption   = core.PaymentOption
	SettleResponse  = core.SettleResponse
	VerifyResponse  = core.VerifyResponse
	Resource        = core.Resource
	Method          = core.Method
	Intent          = core.Intent
	Currency        = core.Currency
)

// Gateway is the central payment gateway that coordinates methods.
type Gateway struct {
	mu      sync.RWMutex
	methods map[string]core.Method
}

// Option configures a Gateway.
type Option func(*Gateway)

// WithMethod registers a payment method with the gateway.
func WithMethod(m core.Method) Option {
	return func(g *Gateway) {
		g.methods[m.Name()] = m
	}
}

// NewGateway creates a gateway with the given options.
func NewGateway(opts ...Option) *Gateway {
	g := &Gateway{methods: make(map[string]core.Method)}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// BuildPaymentRequired constructs a 402 response for a given price and resource.
// It queries all registered methods for their PaymentOptions.
func (g *Gateway) BuildPaymentRequired(resource core.Resource, price Price) (*PaymentRequired, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var accepts []core.PaymentOption
	for _, m := range g.methods {
		opt, err := m.BuildOption(core.IntentCharge, price)
		if err != nil {
			// Method doesn't support this currency/intent — skip it.
			continue
		}
		accepts = append(accepts, opt)
	}

	if len(accepts) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable,
			fmt.Sprintf("no registered method supports %s %s", price.Amount, price.Currency))
	}

	return &PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource:   resource,
		Accepts:    accepts,
	}, nil
}

// Verify verifies a payment payload by dispatching to the correct method.
func (g *Gateway) Verify(ctx context.Context, payload PaymentPayload) (*VerifyResponse, error) {
	m, err := g.resolveMethod(payload.Accepted.Method)
	if err != nil {
		return nil, err
	}
	return m.Verify(ctx, payload, payload.Accepted)
}

// Settle settles a payment payload by dispatching to the correct method.
func (g *Gateway) Settle(ctx context.Context, payload PaymentPayload) (*SettleResponse, error) {
	m, err := g.resolveMethod(payload.Accepted.Method)
	if err != nil {
		return nil, err
	}
	return m.Settle(ctx, payload, payload.Accepted)
}

// Methods returns the names of all registered methods.
func (g *Gateway) Methods() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	names := make([]string, 0, len(g.methods))
	for name := range g.methods {
		names = append(names, name)
	}
	return names
}

// Method returns a specific registered method by name.
func (g *Gateway) Method(name string) (core.Method, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	m, ok := g.methods[name]
	return m, ok
}

func (g *Gateway) resolveMethod(name string) (core.Method, error) {
	g.mu.RLock()
	m, ok := g.methods[name]
	g.mu.RUnlock()
	if !ok {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable,
			fmt.Sprintf("method %q not registered", name))
	}
	return m, nil
}
