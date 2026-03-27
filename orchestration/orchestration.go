// Package orchestration provides smart payment rail selection strategies
// for choosing the optimal payment method based on cost, speed, region,
// and other configurable criteria.
package orchestration

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Strategy selects the best payment option from a set of candidates.
type Strategy interface {
	Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error)
}

// SelectionContext provides context for payment method selection.
type SelectionContext struct {
	AgentID     string
	Resource    core.Resource
	Amount      string
	Currency    core.Currency
	Region      string
	Preferences []string
	History     []HistoryEntry
}

// HistoryEntry records past payment performance for a method.
type HistoryEntry struct {
	Method    string
	Success   bool
	Latency   time.Duration
	Timestamp time.Time
}

// FeeInfo describes the cost structure for a payment method.
type FeeInfo struct {
	FixedFee       string        `json:"fixedFee"`
	PercentFee     float64       `json:"percentFee"`
	Currency       core.Currency `json:"currency"`
	SettlementTime time.Duration `json:"settlementTime"`
}

// FeeTable maps method names to their fee information.
type FeeTable struct {
	mu   sync.RWMutex
	fees map[string]FeeInfo
}

// NewFeeTable creates a fee table from an initial map.
func NewFeeTable(fees map[string]FeeInfo) *FeeTable {
	copied := make(map[string]FeeInfo, len(fees))
	for k, v := range fees {
		copied[k] = v
	}
	return &FeeTable{fees: copied}
}

// Get returns fee info for a method.
func (ft *FeeTable) Get(method string) (FeeInfo, bool) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	fi, ok := ft.fees[method]
	return fi, ok
}

// Set updates fee info for a method.
func (ft *FeeTable) Set(method string, info FeeInfo) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.fees[method] = info
}

// totalFee calculates the total fee for a given transaction amount.
func totalFee(info FeeInfo, amount string) (*big.Rat, error) {
	fixed := new(big.Rat)
	if info.FixedFee != "" {
		if _, ok := fixed.SetString(info.FixedFee); !ok {
			return nil, fmt.Errorf("invalid fixed fee: %s", info.FixedFee)
		}
	}
	amt := new(big.Rat)
	if _, ok := amt.SetString(amount); !ok {
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}
	percent := new(big.Rat).SetFloat64(info.PercentFee)
	percentFee := new(big.Rat).Mul(amt, percent)
	return new(big.Rat).Add(fixed, percentFee), nil
}

// CheapestStrategy selects the method with the lowest total fees.
type CheapestStrategy struct {
	FeeTable *FeeTable
}

// Select picks the cheapest option based on the fee table.
func (s *CheapestStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	var best *core.PaymentOption
	var bestFee *big.Rat

	for i := range options {
		opt := &options[i]
		info, ok := s.FeeTable.Get(opt.Method)
		if !ok {
			continue
		}
		fee, err := totalFee(info, opt.Amount)
		if err != nil {
			continue
		}
		if bestFee == nil || fee.Cmp(bestFee) < 0 {
			best = opt
			bestFee = fee
		}
	}

	if best == nil {
		// Fall back to first option if no fee data available.
		return &options[0], nil
	}
	return best, nil
}

// FastestStrategy selects the method with the fastest settlement time.
type FastestStrategy struct {
	FeeTable *FeeTable
}

// Select picks the fastest-settling option.
func (s *FastestStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	var best *core.PaymentOption
	var bestTime time.Duration
	first := true

	for i := range options {
		opt := &options[i]
		info, ok := s.FeeTable.Get(opt.Method)
		if !ok {
			continue
		}
		if first || info.SettlementTime < bestTime {
			best = opt
			bestTime = info.SettlementTime
			first = false
		}
	}

	if best == nil {
		return &options[0], nil
	}
	return best, nil
}

// PreferredStrategy tries preferred methods first, falling back to remaining.
type PreferredStrategy struct{}

// Select picks the first option matching a preference, or the first option overall.
func (s *PreferredStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	prefSet := make(map[string]int, len(ctx.Preferences))
	for i, p := range ctx.Preferences {
		prefSet[p] = i
	}

	var bestIdx int = -1
	var bestPriority int

	for i := range options {
		priority, ok := prefSet[options[i].Method]
		if ok && (bestIdx == -1 || priority < bestPriority) {
			bestIdx = i
			bestPriority = priority
		}
	}

	if bestIdx >= 0 {
		return &options[bestIdx], nil
	}
	return &options[0], nil
}

// RegionStrategy selects the method best suited for the payer's region.
type RegionStrategy struct {
	// RegionMethods maps ISO 3166 country codes to preferred method names.
	RegionMethods map[string][]string
}

// Select picks the first option matching the payer's region preference.
func (s *RegionStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	preferred, ok := s.RegionMethods[ctx.Region]
	if !ok {
		return &options[0], nil
	}

	prefSet := make(map[string]int, len(preferred))
	for i, m := range preferred {
		prefSet[m] = i
	}

	var bestIdx int = -1
	var bestPriority int

	for i := range options {
		priority, ok := prefSet[options[i].Method]
		if ok && (bestIdx == -1 || priority < bestPriority) {
			bestIdx = i
			bestPriority = priority
		}
	}

	if bestIdx >= 0 {
		return &options[bestIdx], nil
	}
	return &options[0], nil
}

// CompositeStrategy chains multiple strategies, trying each in order.
// The first strategy to return a result wins. If a strategy returns an error,
// the next strategy is tried.
type CompositeStrategy struct {
	Strategies []Strategy
}

// Select tries each strategy in order until one succeeds.
func (s *CompositeStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	for _, strategy := range s.Strategies {
		result, err := strategy.Select(options, ctx)
		if err == nil && result != nil {
			return result, nil
		}
	}

	return &options[0], nil
}

// RoundRobinStrategy distributes selections across methods for load balancing.
type RoundRobinStrategy struct {
	counter atomic.Uint64
}

// Select picks options in round-robin order.
func (s *RoundRobinStrategy) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	if len(options) == 0 {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no payment options available")
	}

	idx := s.counter.Add(1) - 1
	pick := idx % uint64(len(options))
	return &options[pick], nil
}

// Orchestrator selects the optimal payment method using a configurable strategy.
type Orchestrator struct {
	mu       sync.RWMutex
	strategy Strategy
}

// NewOrchestrator creates an orchestrator with the given strategy.
func NewOrchestrator(strategy Strategy) *Orchestrator {
	return &Orchestrator{strategy: strategy}
}

// SetStrategy replaces the current selection strategy.
func (o *Orchestrator) SetStrategy(s Strategy) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.strategy = s
}

// Select picks the best payment option from the available options.
func (o *Orchestrator) Select(options []core.PaymentOption, ctx SelectionContext) (*core.PaymentOption, error) {
	o.mu.RLock()
	s := o.strategy
	o.mu.RUnlock()
	return s.Select(options, ctx)
}

// OrchestratedGateway wraps a payment gateway and uses an orchestrator
// for automatic method selection on the client side.
type OrchestratedGateway struct {
	Gateway      GatewayInterface
	Orchestrator *Orchestrator
}

// GatewayInterface is the subset of gateway methods needed by OrchestratedGateway.
type GatewayInterface interface {
	BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error)
	Verify(ctx context.Context, payload core.PaymentPayload) (*core.VerifyResponse, error)
	Settle(ctx context.Context, payload core.PaymentPayload) (*core.SettleResponse, error)
}

// SelectAndPay builds a payment required response, selects the best method,
// and returns the selected option for the caller to complete the payment.
func (og *OrchestratedGateway) SelectAndPay(ctx context.Context, resource core.Resource, price core.Price, selCtx SelectionContext) (*core.PaymentOption, error) {
	pr, err := og.Gateway.BuildPaymentRequired(resource, price)
	if err != nil {
		return nil, fmt.Errorf("build payment required: %w", err)
	}

	selected, err := og.Orchestrator.Select(pr.Accepts, selCtx)
	if err != nil {
		return nil, fmt.Errorf("select payment method: %w", err)
	}

	return selected, nil
}
