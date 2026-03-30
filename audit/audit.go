package audit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Status represents the state of an audit entry.
type Status string

const (
	StatusPending  Status = "pending"
	StatusVerified Status = "verified"
	StatusSettled  Status = "settled"
	StatusFailed   Status = "failed"
)

// Entry represents a single payment audit log entry.
type Entry struct {
	ID          string
	Timestamp   time.Time
	AgentID     string
	Resource    core.Resource
	Method      string
	Intent      core.Intent
	Amount      string
	Currency    core.Currency
	Status      Status
	Transaction string // provider transaction ID
	PaymentHash string // hash of the payment payload for integrity
	MandateID   string
	Error       string
	Metadata    map[string]string
}

// Filter specifies criteria for querying audit entries.
type Filter struct {
	AgentID         string
	Method          string
	Status          Status
	ResourcePattern string // glob pattern
	TimeFrom        time.Time
	TimeTo          time.Time
	Limit           int
	Offset          int
}

// Logger is the interface for audit logging.
type Logger interface {
	Log(entry Entry) error
	Query(filter Filter) ([]Entry, error)
	Get(id string) (*Entry, error)
}

// MemoryLogger is an in-memory Logger implementation.
type MemoryLogger struct {
	mu      sync.RWMutex
	entries []Entry
	index   map[string]int // ID -> position in entries slice
}

// NewMemoryLogger creates a new in-memory audit logger.
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		index: make(map[string]int),
	}
}

// Log records an audit entry.
func (l *MemoryLogger) Log(entry Entry) error {
	if entry.ID == "" {
		return fmt.Errorf("audit entry ID must not be empty")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.index[entry.ID] = len(l.entries)
	l.entries = append(l.entries, entry)
	return nil
}

// Get retrieves an audit entry by ID.
func (l *MemoryLogger) Get(id string) (*Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	idx, ok := l.index[id]
	if !ok {
		return nil, fmt.Errorf("audit entry %q not found", id)
	}
	e := l.entries[idx]
	return &e, nil
}

// Query returns audit entries matching the given filter.
func (l *MemoryLogger) Query(filter Filter) ([]Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []Entry
	skipped := 0

	for _, e := range l.entries {
		if !matchesFilter(e, filter) {
			continue
		}
		if skipped < filter.Offset {
			skipped++
			continue
		}
		results = append(results, e)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

func matchesFilter(e Entry, f Filter) bool {
	if f.AgentID != "" && e.AgentID != f.AgentID {
		return false
	}
	if f.Method != "" && e.Method != f.Method {
		return false
	}
	if f.Status != "" && e.Status != f.Status {
		return false
	}
	if f.ResourcePattern != "" && !strings.Contains(e.Resource.URL, f.ResourcePattern) {
		return false
	}
	if !f.TimeFrom.IsZero() && e.Timestamp.Before(f.TimeFrom) {
		return false
	}
	if !f.TimeTo.IsZero() && e.Timestamp.After(f.TimeTo) {
		return false
	}
	return true
}

// GatewayInterface defines the subset of Gateway methods that AuditedGateway wraps.
type GatewayInterface interface {
	BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error)
	Verify(ctx context.Context, payload core.PaymentPayload) (*core.VerifyResponse, error)
	Settle(ctx context.Context, payload core.PaymentPayload) (*core.SettleResponse, error)
	Methods() []string
	Method(name string) (core.Method, bool)
}

// AuditedGateway wraps a GatewayInterface and logs all verify/settle operations.
type AuditedGateway struct {
	inner  GatewayInterface
	logger Logger
	nextID func() string
}

// AuditedGatewayOption configures an AuditedGateway.
type AuditedGatewayOption func(*AuditedGateway)

// WithIDGenerator sets a custom ID generator for audit entries.
func WithIDGenerator(fn func() string) AuditedGatewayOption {
	return func(ag *AuditedGateway) { ag.nextID = fn }
}

// NewAuditedGateway wraps a gateway with audit logging.
func NewAuditedGateway(inner GatewayInterface, logger Logger, opts ...AuditedGatewayOption) *AuditedGateway {
	counter := 0
	ag := &AuditedGateway{
		inner:  inner,
		logger: logger,
		nextID: func() string {
			counter++
			return fmt.Sprintf("audit-%d", counter)
		},
	}
	for _, opt := range opts {
		opt(ag)
	}
	return ag
}

// BuildPaymentRequired delegates to the inner gateway.
func (ag *AuditedGateway) BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error) {
	return ag.inner.BuildPaymentRequired(resource, price)
}

// Verify delegates to the inner gateway and logs the result.
func (ag *AuditedGateway) Verify(ctx context.Context, payload core.PaymentPayload) (*core.VerifyResponse, error) {
	entry := Entry{
		ID:        ag.nextID(),
		Timestamp: time.Now(),
		Resource:  payload.Resource,
		Method:    payload.Accepted.Method,
		Intent:    payload.Accepted.Intent,
		Amount:    payload.Accepted.Amount,
		Currency:  payload.Accepted.Currency,
		Status:    StatusPending,
	}

	resp, err := ag.inner.Verify(ctx, payload)
	if err != nil {
		entry.Status = StatusFailed
		entry.Error = err.Error()
		if logErr := ag.logger.Log(entry); logErr != nil {
			return nil, fmt.Errorf("audit log failed: %w (original: %v)", logErr, err)
		}
		return nil, err
	}

	if resp.Valid {
		entry.Status = StatusVerified
	} else {
		entry.Status = StatusFailed
		entry.Error = resp.Reason
	}

	if logErr := ag.logger.Log(entry); logErr != nil {
		return resp, fmt.Errorf("audit log failed: %w", logErr)
	}
	return resp, nil
}

// Settle delegates to the inner gateway and logs the result.
func (ag *AuditedGateway) Settle(ctx context.Context, payload core.PaymentPayload) (*core.SettleResponse, error) {
	entry := Entry{
		ID:        ag.nextID(),
		Timestamp: time.Now(),
		Resource:  payload.Resource,
		Method:    payload.Accepted.Method,
		Intent:    payload.Accepted.Intent,
		Amount:    payload.Accepted.Amount,
		Currency:  payload.Accepted.Currency,
		Status:    StatusPending,
	}

	resp, err := ag.inner.Settle(ctx, payload)
	if err != nil {
		entry.Status = StatusFailed
		entry.Error = err.Error()
		if logErr := ag.logger.Log(entry); logErr != nil {
			return nil, fmt.Errorf("audit log failed: %w (original: %v)", logErr, err)
		}
		return nil, err
	}

	if resp.Success {
		entry.Status = StatusSettled
		entry.Transaction = resp.Transaction
	} else {
		entry.Status = StatusFailed
	}

	if logErr := ag.logger.Log(entry); logErr != nil {
		return resp, fmt.Errorf("audit log failed: %w", logErr)
	}
	return resp, nil
}

// Methods delegates to the inner gateway.
func (ag *AuditedGateway) Methods() []string {
	return ag.inner.Methods()
}

// Method delegates to the inner gateway.
func (ag *AuditedGateway) Method(name string) (core.Method, bool) {
	return ag.inner.Method(name)
}
