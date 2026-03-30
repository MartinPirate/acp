package discovery

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

// ServiceType identifies whether a service is a payment method or facilitator.
type ServiceType string

const (
	ServiceTypeMethod      ServiceType = "method"
	ServiceTypeFacilitator ServiceType = "facilitator"
)

// ServiceStatus represents the operational status of a service.
type ServiceStatus string

const (
	StatusActive   ServiceStatus = "active"
	StatusDegraded ServiceStatus = "degraded"
	StatusOffline  ServiceStatus = "offline"
)

// ServiceEntry describes a registered payment service.
type ServiceEntry struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ServiceType       `json:"type"`
	URL         string            `json:"url"`
	Methods     []string          `json:"methods,omitempty"`
	Intents     []core.Intent     `json:"intents,omitempty"`
	Currencies  []core.Currency   `json:"currencies,omitempty"`
	Regions     []string          `json:"regions,omitempty"`
	MinAmount   string            `json:"minAmount,omitempty"`
	MaxAmount   string            `json:"maxAmount,omitempty"`
	HealthURL   string            `json:"healthURL,omitempty"`
	Status      ServiceStatus     `json:"status"`
	LastChecked time.Time         `json:"lastChecked,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Query specifies filters for discovering services.
type Query struct {
	Methods    []string          `json:"methods,omitempty"`
	Intents    []core.Intent     `json:"intents,omitempty"`
	Currencies []core.Currency   `json:"currencies,omitempty"`
	Regions    []string          `json:"regions,omitempty"`
	MinAmount  string            `json:"minAmount,omitempty"`
	MaxAmount  string            `json:"maxAmount,omitempty"`
	Status     ServiceStatus     `json:"status,omitempty"`
}

// Registry is a thread-safe registry of available payment services.
type Registry struct {
	mu       sync.RWMutex
	services map[string]ServiceEntry
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]ServiceEntry),
	}
}

// Register adds or updates a service entry in the registry.
func (r *Registry) Register(entry ServiceEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[entry.ID] = entry
}

// Unregister removes a service entry from the registry.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, id)
}

// Discover returns all services matching the given query filters.
// An empty query returns all services.
func (r *Registry) Discover(query Query) []ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ServiceEntry
	for _, entry := range r.services {
		if matchesQuery(entry, query) {
			results = append(results, entry)
		}
	}
	return results
}

// All returns every registered service entry.
func (r *Registry) All() []ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]ServiceEntry, 0, len(r.services))
	for _, e := range r.services {
		entries = append(entries, e)
	}
	return entries
}

// Get returns a single service entry by ID.
func (r *Registry) Get(id string) (ServiceEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.services[id]
	return e, ok
}

// UpdateStatus updates the status and last-checked timestamp for a service.
func (r *Registry) UpdateStatus(id string, status ServiceStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.services[id]; ok {
		entry.Status = status
		entry.LastChecked = time.Now()
		r.services[id] = entry
	}
}

func matchesQuery(entry ServiceEntry, q Query) bool {
	if q.Status != "" && entry.Status != q.Status {
		return false
	}
	if len(q.Methods) > 0 && !hasOverlap(entry.Methods, q.Methods) {
		return false
	}
	if len(q.Intents) > 0 && !hasIntentOverlap(entry.Intents, q.Intents) {
		return false
	}
	if len(q.Currencies) > 0 && !hasCurrencyOverlap(entry.Currencies, q.Currencies) {
		return false
	}
	if len(q.Regions) > 0 && !hasOverlap(entry.Regions, q.Regions) {
		return false
	}
	if q.MinAmount != "" && entry.MaxAmount != "" {
		// Entry's max must be >= query's min
		cmp, err := compareAmountStrings(entry.MaxAmount, q.MinAmount)
		if err == nil && cmp < 0 {
			return false
		}
	}
	if q.MaxAmount != "" && entry.MinAmount != "" {
		// Entry's min must be <= query's max
		cmp, err := compareAmountStrings(entry.MinAmount, q.MaxAmount)
		if err == nil && cmp > 0 {
			return false
		}
	}
	return true
}

func hasOverlap(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func hasIntentOverlap(a, b []core.Intent) bool {
	set := make(map[core.Intent]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func hasCurrencyOverlap(a, b []core.Currency) bool {
	set := make(map[core.Currency]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func compareAmountStrings(a, b string) (int, error) {
	ra := new(big.Rat)
	if _, ok := ra.SetString(a); !ok {
		return 0, core.NewPaymentError(core.ErrInvalidPayload, "invalid amount: "+a)
	}
	rb := new(big.Rat)
	if _, ok := rb.SetString(b); !ok {
		return 0, core.NewPaymentError(core.ErrInvalidPayload, "invalid amount: "+b)
	}
	return ra.Cmp(rb), nil
}

// HealthChecker periodically pings registered services and updates their status.
type HealthChecker struct {
	registry *Registry
	interval time.Duration
	client   *http.Client
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewHealthChecker creates a health checker that polls at the given interval.
func NewHealthChecker(registry *Registry, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		interval: interval,
		client:   &http.Client{Timeout: 5 * time.Second},
		done:     make(chan struct{}),
	}
}

// Start begins the background health checking loop.
func (hc *HealthChecker) Start(ctx context.Context) {
	ctx, hc.cancel = context.WithCancel(ctx)
	go hc.run(ctx)
}

// Stop halts the background health checker and waits for it to finish.
func (hc *HealthChecker) Stop() {
	if hc.cancel != nil {
		hc.cancel()
		<-hc.done
	}
}

func (hc *HealthChecker) run(ctx context.Context) {
	defer close(hc.done)

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Run an initial check immediately.
	hc.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAll(ctx)
		}
	}
}

func (hc *HealthChecker) checkAll(ctx context.Context) {
	entries := hc.registry.All()
	for _, entry := range entries {
		if entry.HealthURL == "" {
			continue
		}
		status := hc.ping(ctx, entry.HealthURL)
		hc.registry.UpdateStatus(entry.ID, status)
	}
}

func (hc *HealthChecker) ping(_ context.Context, url string) ServiceStatus {
	// Use a per-request timeout instead of the parent context so that
	// cancelling the health-checker loop does not abort in-flight pings.
	pingCtx, cancel := context.WithTimeout(context.Background(), hc.client.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("discovery: health check request error for %s: %v", url, err)
		return StatusOffline
	}
	resp, err := hc.client.Do(req)
	if err != nil {
		return StatusOffline
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return StatusActive
	}
	if resp.StatusCode >= 500 {
		return StatusOffline
	}
	return StatusDegraded
}

// WellKnownHandler returns an http.Handler that serves the
// /.well-known/acp-services endpoint, returning all active services as JSON.
func WellKnownHandler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entries := registry.Discover(Query{Status: StatusActive})
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			log.Printf("discovery: failed to encode well-known response: %v", err)
		}
	})
}
