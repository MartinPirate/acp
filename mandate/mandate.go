package mandate

import (
	"fmt"
	"math/big"
	"path"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Mandate represents a pre-authorized spending authority for an agent.
type Mandate struct {
	ID             string
	AgentID        string
	MaxAmount      string
	Currency       core.Currency
	MaxPerRequest  string
	AllowedMethods []string
	AllowedIntents []core.Intent
	Scope          []string // URL glob patterns
	CreatedAt      time.Time
	ExpiresAt      time.Time
	Metadata       map[string]string
}

// MandateOption configures a Mandate.
type MandateOption func(*Mandate)

// WithID sets the mandate ID.
func WithID(id string) MandateOption {
	return func(m *Mandate) { m.ID = id }
}

// WithAgentID sets the agent this mandate is for.
func WithAgentID(id string) MandateOption {
	return func(m *Mandate) { m.AgentID = id }
}

// WithMaxAmount sets the total spending limit.
func WithMaxAmount(amount string) MandateOption {
	return func(m *Mandate) { m.MaxAmount = amount }
}

// WithCurrency sets the mandate currency.
func WithCurrency(c core.Currency) MandateOption {
	return func(m *Mandate) { m.Currency = c }
}

// WithMaxPerRequest sets the per-request cap.
func WithMaxPerRequest(amount string) MandateOption {
	return func(m *Mandate) { m.MaxPerRequest = amount }
}

// WithAllowedMethods sets which payment methods are allowed.
func WithAllowedMethods(methods ...string) MandateOption {
	return func(m *Mandate) { m.AllowedMethods = methods }
}

// WithAllowedIntents sets which intents are allowed.
func WithAllowedIntents(intents ...core.Intent) MandateOption {
	return func(m *Mandate) { m.AllowedIntents = intents }
}

// WithScope sets URL patterns this mandate covers.
func WithScope(patterns ...string) MandateOption {
	return func(m *Mandate) { m.Scope = patterns }
}

// WithExpiry sets the mandate expiry time.
func WithExpiry(t time.Time) MandateOption {
	return func(m *Mandate) { m.ExpiresAt = t }
}

// WithMetadata sets arbitrary key-value pairs.
func WithMetadata(md map[string]string) MandateOption {
	return func(m *Mandate) { m.Metadata = md }
}

// NewMandate creates a Mandate with functional options.
func NewMandate(opts ...MandateOption) *Mandate {
	m := &Mandate{
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Enforcer validates payments against mandates and tracks cumulative spend.
type Enforcer struct {
	mu    sync.Mutex
	spent map[string]*big.Rat // mandate ID -> cumulative spend
}

// NewEnforcer creates a new Enforcer.
func NewEnforcer() *Enforcer {
	return &Enforcer{
		spent: make(map[string]*big.Rat),
	}
}

// Check validates that a payment is allowed under the given mandate.
func (e *Enforcer) Check(mandate *Mandate, payload core.PaymentPayload, resource core.Resource) error {
	// Check expiry.
	if !mandate.ExpiresAt.IsZero() && time.Now().After(mandate.ExpiresAt) {
		return core.NewPaymentError(core.ErrMandateExpired,
			fmt.Sprintf("mandate %s expired at %s", mandate.ID, mandate.ExpiresAt.Format(time.RFC3339)))
	}

	// Check currency.
	if mandate.Currency != payload.Accepted.Currency {
		return core.NewPaymentError(core.ErrCurrencyMismatch,
			fmt.Sprintf("mandate currency %s does not match payment currency %s",
				mandate.Currency, payload.Accepted.Currency))
	}

	// Check method.
	if len(mandate.AllowedMethods) > 0 && !contains(mandate.AllowedMethods, payload.Accepted.Method) {
		return core.NewPaymentError(core.ErrMethodUnavailable,
			fmt.Sprintf("method %q not allowed by mandate %s", payload.Accepted.Method, mandate.ID))
	}

	// Check intent.
	if len(mandate.AllowedIntents) > 0 && !containsIntent(mandate.AllowedIntents, payload.Accepted.Intent) {
		return core.NewPaymentError(core.ErrUnsupportedIntent,
			fmt.Sprintf("intent %q not allowed by mandate %s", payload.Accepted.Intent, mandate.ID))
	}

	// Check scope.
	if len(mandate.Scope) > 0 && !matchesScope(mandate.Scope, resource.URL) {
		return core.NewPaymentError(core.ErrMandateExceeded,
			fmt.Sprintf("resource %q not in scope of mandate %s", resource.URL, mandate.ID))
	}

	amount := payload.Accepted.Amount

	// Check per-request cap.
	if mandate.MaxPerRequest != "" {
		cmp, err := core.CompareAmounts(amount, mandate.MaxPerRequest)
		if err != nil {
			return core.NewPaymentError(core.ErrInvalidPayload, err.Error())
		}
		if cmp > 0 {
			return core.NewPaymentError(core.ErrAmountTooHigh,
				fmt.Sprintf("amount %s exceeds per-request limit %s", amount, mandate.MaxPerRequest))
		}
	}

	// Check cumulative spend against max amount.
	if mandate.MaxAmount != "" {
		e.mu.Lock()
		cumulative := e.spent[mandate.ID]
		if cumulative == nil {
			cumulative = new(big.Rat)
		}
		e.mu.Unlock()

		paymentAmt := new(big.Rat)
		if _, ok := paymentAmt.SetString(amount); !ok {
			return core.NewPaymentError(core.ErrInvalidPayload, "invalid amount: "+amount)
		}

		projected := new(big.Rat).Add(cumulative, paymentAmt)
		maxAmt := new(big.Rat)
		if _, ok := maxAmt.SetString(mandate.MaxAmount); !ok {
			return core.NewPaymentError(core.ErrInvalidPayload, "invalid max amount: "+mandate.MaxAmount)
		}

		if projected.Cmp(maxAmt) > 0 {
			return core.NewPaymentError(core.ErrMandateExceeded,
				fmt.Sprintf("projected spend %s would exceed mandate limit %s",
					projected.FloatString(10), mandate.MaxAmount))
		}
	}

	return nil
}

// Record tracks a successful payment against a mandate.
func (e *Enforcer) Record(mandate *Mandate, amount string) error {
	paymentAmt := new(big.Rat)
	if _, ok := paymentAmt.SetString(amount); !ok {
		return core.NewPaymentError(core.ErrInvalidPayload, "invalid amount: "+amount)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	cumulative := e.spent[mandate.ID]
	if cumulative == nil {
		cumulative = new(big.Rat)
	}
	e.spent[mandate.ID] = new(big.Rat).Add(cumulative, paymentAmt)
	return nil
}

// Spent returns the cumulative spend for a mandate.
func (e *Enforcer) Spent(mandateID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.spent[mandateID]
	if s == nil {
		return "0"
	}
	return s.FloatString(10)
}

// Store is the persistence interface for mandates.
type Store interface {
	Save(m *Mandate) error
	Get(id string) (*Mandate, error)
	List(agentID string) ([]*Mandate, error)
	Revoke(id string) error
}

// MemoryStore is an in-memory Store implementation.
type MemoryStore struct {
	mu       sync.RWMutex
	mandates map[string]*Mandate
}

// NewMemoryStore creates a new in-memory mandate store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		mandates: make(map[string]*Mandate),
	}
}

// Save persists a mandate.
func (s *MemoryStore) Save(m *Mandate) error {
	if m.ID == "" {
		return fmt.Errorf("mandate ID must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mandates[m.ID] = m
	return nil
}

// Get retrieves a mandate by ID.
func (s *MemoryStore) Get(id string) (*Mandate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.mandates[id]
	if !ok {
		return nil, fmt.Errorf("mandate %q not found", id)
	}
	return m, nil
}

// List returns all mandates for the given agent.
func (s *MemoryStore) List(agentID string) ([]*Mandate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Mandate
	for _, m := range s.mandates {
		if m.AgentID == agentID {
			result = append(result, m)
		}
	}
	return result, nil
}

// Revoke removes a mandate by ID.
func (s *MemoryStore) Revoke(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mandates[id]; !ok {
		return fmt.Errorf("mandate %q not found", id)
	}
	delete(s.mandates, id)
	return nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func containsIntent(intents []core.Intent, intent core.Intent) bool {
	for _, i := range intents {
		if i == intent {
			return true
		}
	}
	return false
}

func matchesScope(patterns []string, url string) bool {
	for _, pattern := range patterns {
		if matched, err := path.Match(pattern, url); err == nil && matched {
			return true
		}
	}
	return false
}
