// Package ratelimit provides rate limiting and anomaly detection for ACP
// payments.
//
// It ships two [Limiter] implementations and an [AnomalyDetector] that can
// flag unusual spending patterns per agent.
//
// # Limiters
//
//   - [TokenBucketLimiter] -- classic token bucket with configurable rate,
//     burst, and automatic cleanup of stale entries.
//   - [SlidingWindowLimiter] -- sliding window counter that caps the number
//     of requests within a rolling time window.
//
// # Anomaly Detection
//
// [AnomalyDetector] tracks per-agent payment history and flags transactions
// that exceed configurable amount or frequency thresholds, or use a
// previously unseen payment method.
//
// # Gateway Wrapper
//
// [RateLimitedGateway] decorates any [GatewayInterface] so that verify and
// settle calls are rejected when the rate limit is exceeded.
//
// # Usage
//
//	limiter := ratelimit.NewTokenBucketLimiter(ratelimit.TokenBucketConfig{
//	    Rate:  10,  // 10 requests per second
//	    Burst: 20,
//	})
//	defer limiter.Stop()
//
//	rlGateway := ratelimit.NewRateLimitedGateway(gateway, limiter, func(p core.PaymentPayload) string {
//	    return p.Resource.URL
//	})
//
// For HTTP-level rate limiting, use [RateLimitMiddleware]:
//
//	mux.Handle("/api/", ratelimit.RateLimitMiddleware(limiter, ratelimit.IPKeyFunc)(handler))
package ratelimit
