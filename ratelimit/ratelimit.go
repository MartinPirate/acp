// Package ratelimit provides rate limiting and anomaly detection for ACP payments.
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Limiter controls request rates.
type Limiter interface {
	Allow(key string) (bool, error)
	Reset(key string)
}

// --- Token Bucket Limiter ---

// TokenBucketConfig configures a TokenBucketLimiter.
type TokenBucketConfig struct {
	Rate            float64       // tokens per second
	Burst           int           // max burst size
	CleanupInterval time.Duration // how often to purge stale entries
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

// TokenBucketLimiter implements the token bucket algorithm.
type TokenBucketLimiter struct {
	mu      sync.Mutex
	config  TokenBucketConfig
	buckets map[string]*tokenBucket
	stopCh  chan struct{}
}

// NewTokenBucketLimiter creates a new token bucket limiter.
func NewTokenBucketLimiter(cfg TokenBucketConfig) *TokenBucketLimiter {
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = time.Minute
	}
	l := &TokenBucketLimiter{
		config:  cfg,
		buckets: make(map[string]*tokenBucket),
		stopCh:  make(chan struct{}),
	}
	go l.cleanup()
	return l
}

// Allow checks if a request for the given key is allowed.
func (l *TokenBucketLimiter) Allow(key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &tokenBucket{
			tokens:   float64(l.config.Burst),
			lastTime: now,
		}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * l.config.Rate
	if b.tokens > float64(l.config.Burst) {
		b.tokens = float64(l.config.Burst)
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false, nil
	}
	b.tokens--
	return true, nil
}

// Reset clears rate limit state for a key.
func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// Stop terminates the background cleanup goroutine.
func (l *TokenBucketLimiter) Stop() {
	close(l.stopCh)
}

func (l *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for key, b := range l.buckets {
				// Remove entries that have been idle for 2x the cleanup interval
				if now.Sub(b.lastTime) > 2*l.config.CleanupInterval {
					delete(l.buckets, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

// --- Sliding Window Limiter ---

// SlidingWindowConfig configures a SlidingWindowLimiter.
type SlidingWindowConfig struct {
	WindowSize  time.Duration
	MaxRequests int
}

// SlidingWindowLimiter implements a sliding window counter.
type SlidingWindowLimiter struct {
	mu      sync.Mutex
	config  SlidingWindowConfig
	windows map[string][]time.Time
}

// NewSlidingWindowLimiter creates a new sliding window limiter.
func NewSlidingWindowLimiter(cfg SlidingWindowConfig) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		config:  cfg,
		windows: make(map[string][]time.Time),
	}
}

// Allow checks if a request for the given key is allowed.
func (l *SlidingWindowLimiter) Allow(key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.config.WindowSize)

	// Prune expired timestamps.
	times := l.windows[key]
	start := 0
	for start < len(times) && times[start].Before(cutoff) {
		start++
	}
	times = times[start:]

	if len(times) >= l.config.MaxRequests {
		l.windows[key] = times
		return false, nil
	}

	l.windows[key] = append(times, now)
	return true, nil
}

// Reset clears rate limit state for a key.
func (l *SlidingWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}

// --- Anomaly Detection ---

// AnomalyResult holds the result of an anomaly check.
type AnomalyResult struct {
	IsAnomaly bool
	Reasons   []string
	RiskScore float64
}

type agentStats struct {
	totalAmount *big.Rat
	count       int64
	methods     map[string]bool
	lastSeen    time.Time
	// sliding window for frequency detection
	recentTimes []time.Time
}

// AnomalyDetector tracks payment patterns per agent and detects anomalies.
type AnomalyDetector struct {
	mu    sync.Mutex
	stats map[string]*agentStats
	// Configurable thresholds
	AmountMultiplier    float64 // flag if amount > N * average (default 3)
	FrequencyMultiplier float64 // flag if rate > N * normal (default 2)
	FrequencyWindow     time.Duration
}

// NewAnomalyDetector creates a new anomaly detector.
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		stats:               make(map[string]*agentStats),
		AmountMultiplier:    3.0,
		FrequencyMultiplier: 2.0,
		FrequencyWindow:     time.Hour,
	}
}

// Check evaluates whether a payment is anomalous for the given agent.
func (d *AnomalyDetector) Check(agentID, amount string, currency core.Currency, method string) AnomalyResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	result := AnomalyResult{}
	now := time.Now()

	s, exists := d.stats[agentID]
	if !exists {
		// First transaction for this agent — create stats entry.
		amt := new(big.Rat)
		amt.SetString(amount)
		d.stats[agentID] = &agentStats{
			totalAmount: amt,
			count:       1,
			methods:     map[string]bool{method: true},
			lastSeen:    now,
			recentTimes: []time.Time{now},
		}
		return result
	}

	paymentAmt := new(big.Rat)
	if _, ok := paymentAmt.SetString(amount); !ok {
		result.IsAnomaly = true
		result.Reasons = append(result.Reasons, "invalid payment amount")
		result.RiskScore = 1.0
		return result
	}

	// Check unusual amount (> AmountMultiplier * average).
	if s.count > 0 {
		avg := new(big.Rat).Quo(s.totalAmount, new(big.Rat).SetInt64(s.count))
		threshold := new(big.Rat).Mul(avg, new(big.Rat).SetFloat64(d.AmountMultiplier))
		if paymentAmt.Cmp(threshold) > 0 {
			result.IsAnomaly = true
			result.Reasons = append(result.Reasons,
				fmt.Sprintf("amount %s exceeds %.0fx average (%s)",
					amount, d.AmountMultiplier, avg.FloatString(2)))
			result.RiskScore += 0.5
		}
	}

	// Check unusual frequency.
	cutoff := now.Add(-d.FrequencyWindow)
	recentCount := 0
	for _, t := range s.recentTimes {
		if t.After(cutoff) {
			recentCount++
		}
	}
	if s.count > 2 {
		// Calculate expected rate (total count / age in windows).
		age := now.Sub(s.recentTimes[0])
		if age > 0 {
			windowCount := age.Seconds() / d.FrequencyWindow.Seconds()
			if windowCount < 1 {
				windowCount = 1
			}
			expectedRate := float64(s.count) / windowCount
			currentRate := float64(recentCount + 1)
			if currentRate > d.FrequencyMultiplier*expectedRate {
				result.IsAnomaly = true
				result.Reasons = append(result.Reasons,
					fmt.Sprintf("request frequency %.1f/window exceeds %.0fx normal rate %.1f/window",
						currentRate, d.FrequencyMultiplier, expectedRate))
				result.RiskScore += 0.3
			}
		}
	}

	// Check new/unknown method.
	if !s.methods[method] {
		result.IsAnomaly = true
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("unknown payment method %q for this agent", method))
		result.RiskScore += 0.2
	}

	// Cap risk score at 1.0.
	result.RiskScore = math.Min(result.RiskScore, 1.0)

	// Update stats.
	s.totalAmount.Add(s.totalAmount, paymentAmt)
	s.count++
	s.methods[method] = true
	s.lastSeen = now
	// Prune old times from window.
	pruned := s.recentTimes[:0]
	for _, t := range s.recentTimes {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	s.recentTimes = append(pruned, now)

	return result
}

// --- Rate Limited Gateway ---

// GatewayInterface defines the subset of Gateway methods that can be wrapped.
type GatewayInterface interface {
	BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error)
	Verify(ctx context.Context, payload core.PaymentPayload) (*core.VerifyResponse, error)
	Settle(ctx context.Context, payload core.PaymentPayload) (*core.SettleResponse, error)
	Methods() []string
	Method(name string) (core.Method, bool)
}

// RateLimitedGateway wraps a GatewayInterface with per-agent rate limiting.
type RateLimitedGateway struct {
	inner   GatewayInterface
	limiter Limiter
	keyFunc func(core.PaymentPayload) string
}

// NewRateLimitedGateway creates a rate-limited wrapper around a gateway.
// keyFunc extracts the rate limit key from a payment payload (e.g. agent ID).
func NewRateLimitedGateway(inner GatewayInterface, limiter Limiter, keyFunc func(core.PaymentPayload) string) *RateLimitedGateway {
	if keyFunc == nil {
		keyFunc = func(p core.PaymentPayload) string {
			return p.Resource.URL
		}
	}
	return &RateLimitedGateway{inner: inner, limiter: limiter, keyFunc: keyFunc}
}

// BuildPaymentRequired delegates to the inner gateway (not rate limited).
func (g *RateLimitedGateway) BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error) {
	return g.inner.BuildPaymentRequired(resource, price)
}

// Verify checks the rate limit before delegating to the inner gateway.
func (g *RateLimitedGateway) Verify(ctx context.Context, payload core.PaymentPayload) (*core.VerifyResponse, error) {
	key := g.keyFunc(payload)
	allowed, err := g.limiter.Allow(key)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if !allowed {
		return nil, core.NewPaymentError(core.ErrTimeout, "rate limit exceeded")
	}
	return g.inner.Verify(ctx, payload)
}

// Settle checks the rate limit before delegating to the inner gateway.
func (g *RateLimitedGateway) Settle(ctx context.Context, payload core.PaymentPayload) (*core.SettleResponse, error) {
	key := g.keyFunc(payload)
	allowed, err := g.limiter.Allow(key)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if !allowed {
		return nil, core.NewPaymentError(core.ErrTimeout, "rate limit exceeded")
	}
	return g.inner.Settle(ctx, payload)
}

// Methods delegates to the inner gateway.
func (g *RateLimitedGateway) Methods() []string {
	return g.inner.Methods()
}

// Method delegates to the inner gateway.
func (g *RateLimitedGateway) Method(name string) (core.Method, bool) {
	return g.inner.Method(name)
}

// --- HTTP Middleware ---

// KeyFunc extracts a rate limit key from an HTTP request.
type KeyFunc func(r *http.Request) string

// IPKeyFunc extracts the remote address as the rate limit key.
func IPKeyFunc(r *http.Request) string {
	return r.RemoteAddr
}

// RateLimitMiddleware returns HTTP middleware that applies rate limiting.
func RateLimitMiddleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, err := limiter.Allow(key)
			if err != nil {
				http.Error(w, "internal rate limiter error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
