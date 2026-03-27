package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func TestTokenBucketLimiter(t *testing.T) {
	limiter := NewTokenBucketLimiter(TokenBucketConfig{
		Rate:            10, // 10 per second
		Burst:           3,
		CleanupInterval: time.Hour, // no cleanup during test
	})
	defer limiter.Stop()

	// Should allow burst.
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow("key1")
		if err != nil {
			t.Fatalf("Allow error: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Burst exhausted — should be denied.
	allowed, err := limiter.Allow("key1")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allowed {
		t.Error("request after burst should be denied")
	}

	// Different key should work.
	allowed, err = limiter.Allow("key2")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allowed {
		t.Error("different key should be allowed")
	}
}

func TestTokenBucketLimiterRefill(t *testing.T) {
	limiter := NewTokenBucketLimiter(TokenBucketConfig{
		Rate:            1000, // very fast refill
		Burst:           1,
		CleanupInterval: time.Hour,
	})
	defer limiter.Stop()

	// Use the one token.
	limiter.Allow("key1")

	// Wait briefly for refill.
	time.Sleep(5 * time.Millisecond)

	allowed, _ := limiter.Allow("key1")
	if !allowed {
		t.Error("should be allowed after refill")
	}
}

func TestTokenBucketLimiterReset(t *testing.T) {
	limiter := NewTokenBucketLimiter(TokenBucketConfig{
		Rate:            0.001, // very slow refill
		Burst:           1,
		CleanupInterval: time.Hour,
	})
	defer limiter.Stop()

	limiter.Allow("key1") // use up the token

	limiter.Reset("key1")

	allowed, _ := limiter.Allow("key1")
	if !allowed {
		t.Error("should be allowed after reset")
	}
}

func TestSlidingWindowLimiter(t *testing.T) {
	limiter := NewSlidingWindowLimiter(SlidingWindowConfig{
		WindowSize:  100 * time.Millisecond,
		MaxRequests: 3,
	})

	// Should allow up to max.
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow("key1")
		if err != nil {
			t.Fatalf("Allow error: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Should deny the 4th.
	allowed, _ := limiter.Allow("key1")
	if allowed {
		t.Error("4th request should be denied")
	}

	// Wait for window to expire.
	time.Sleep(150 * time.Millisecond)

	allowed, _ = limiter.Allow("key1")
	if !allowed {
		t.Error("should be allowed after window expires")
	}
}

func TestSlidingWindowLimiterReset(t *testing.T) {
	limiter := NewSlidingWindowLimiter(SlidingWindowConfig{
		WindowSize:  time.Hour,
		MaxRequests: 1,
	})

	limiter.Allow("key1")

	limiter.Reset("key1")

	allowed, _ := limiter.Allow("key1")
	if !allowed {
		t.Error("should be allowed after reset")
	}
}

func TestAnomalyDetectorFirstTransaction(t *testing.T) {
	d := NewAnomalyDetector()
	result := d.Check("agent-1", "10.00", core.USD, "card")
	if result.IsAnomaly {
		t.Error("first transaction should not be anomalous")
	}
}

func TestAnomalyDetectorUnusualAmount(t *testing.T) {
	d := NewAnomalyDetector()

	// Build history with small amounts.
	for i := 0; i < 10; i++ {
		d.Check("agent-1", "10.00", core.USD, "card")
	}

	// Large amount should trigger anomaly.
	result := d.Check("agent-1", "500.00", core.USD, "card")
	if !result.IsAnomaly {
		t.Error("unusually large amount should be flagged")
	}

	found := false
	for _, r := range result.Reasons {
		if len(r) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected reasons to be populated")
	}
}

func TestAnomalyDetectorNewMethod(t *testing.T) {
	d := NewAnomalyDetector()

	// Build history with "card".
	d.Check("agent-1", "10.00", core.USD, "card")
	d.Check("agent-1", "10.00", core.USD, "card")

	// New method should trigger.
	result := d.Check("agent-1", "10.00", core.USD, "crypto")
	if !result.IsAnomaly {
		t.Error("new payment method should be flagged")
	}

	hasMethodReason := false
	for _, r := range result.Reasons {
		if r != "" {
			hasMethodReason = true
		}
	}
	if !hasMethodReason {
		t.Error("expected method-related reason")
	}
}

func TestAnomalyDetectorInvalidAmount(t *testing.T) {
	d := NewAnomalyDetector()
	d.Check("agent-1", "10.00", core.USD, "card")

	result := d.Check("agent-1", "not-a-number", core.USD, "card")
	if !result.IsAnomaly {
		t.Error("invalid amount should be flagged")
	}
	if result.RiskScore != 1.0 {
		t.Errorf("risk score = %f, want 1.0", result.RiskScore)
	}
}

// --- RateLimitedGateway tests ---

type mockGateway struct {
	verifyResp *core.VerifyResponse
	settleResp *core.SettleResponse
}

func (g *mockGateway) BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error) {
	return nil, nil
}
func (g *mockGateway) Verify(_ context.Context, _ core.PaymentPayload) (*core.VerifyResponse, error) {
	return g.verifyResp, nil
}
func (g *mockGateway) Settle(_ context.Context, _ core.PaymentPayload) (*core.SettleResponse, error) {
	return g.settleResp, nil
}
func (g *mockGateway) Methods() []string                     { return nil }
func (g *mockGateway) Method(_ string) (core.Method, bool) { return nil, false }

func TestRateLimitedGatewayAllow(t *testing.T) {
	inner := &mockGateway{
		verifyResp: &core.VerifyResponse{Valid: true},
	}
	limiter := NewTokenBucketLimiter(TokenBucketConfig{Rate: 100, Burst: 10, CleanupInterval: time.Hour})
	defer limiter.Stop()

	gw := NewRateLimitedGateway(inner, limiter, func(p core.PaymentPayload) string {
		return "test-key"
	})

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00"},
	}

	resp, err := gw.Verify(context.Background(), payload)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !resp.Valid {
		t.Error("expected valid response")
	}
}

func TestRateLimitedGatewayDeny(t *testing.T) {
	inner := &mockGateway{
		verifyResp: &core.VerifyResponse{Valid: true},
	}
	limiter := NewTokenBucketLimiter(TokenBucketConfig{Rate: 0.001, Burst: 1, CleanupInterval: time.Hour})
	defer limiter.Stop()

	gw := NewRateLimitedGateway(inner, limiter, func(p core.PaymentPayload) string {
		return "test-key"
	})

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00"},
	}

	// First should pass.
	_, err := gw.Verify(context.Background(), payload)
	if err != nil {
		t.Fatalf("first Verify failed: %v", err)
	}

	// Second should be rate limited.
	_, err = gw.Verify(context.Background(), payload)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestRateLimitedGatewaySettleDeny(t *testing.T) {
	inner := &mockGateway{
		settleResp: &core.SettleResponse{Success: true},
	}
	limiter := NewTokenBucketLimiter(TokenBucketConfig{Rate: 0.001, Burst: 1, CleanupInterval: time.Hour})
	defer limiter.Stop()

	gw := NewRateLimitedGateway(inner, limiter, func(p core.PaymentPayload) string {
		return "test-key"
	})

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00"},
	}

	_, err := gw.Settle(context.Background(), payload)
	if err != nil {
		t.Fatalf("first Settle failed: %v", err)
	}

	_, err = gw.Settle(context.Background(), payload)
	if err == nil {
		t.Fatal("expected rate limit error on settle")
	}
}

// --- HTTP Middleware tests ---

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewTokenBucketLimiter(TokenBucketConfig{Rate: 0.001, Burst: 2, CleanupInterval: time.Hour})
	defer limiter.Stop()

	handler := RateLimitMiddleware(limiter, IPKeyFunc)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		}),
	)

	// First two requests should pass.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
	}

	// Third should be rate limited.
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}
