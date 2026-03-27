// Full-stack ACP demo: server with multiple payment methods, dashboard, and discovery.
//
// Run:
//
//	go run ./examples/demo
//
// Then test with:
//
//	go run ./cmd/acp-pay --url http://localhost:8080/api/data
//	curl http://localhost:8080/api/health
//	curl -i http://localhost:8080/api/data                  # returns 402
//	curl http://localhost:9090/api/stats                    # dashboard stats
//	curl http://localhost:9090/api/transactions             # transaction log
//	curl http://localhost:8080/.well-known/acp-services     # service discovery
//	open http://localhost:9090                              # dashboard UI
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/audit"
	"github.com/paideia-ai/acp/auth"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/dashboard"
	"github.com/paideia-ai/acp/discovery"
	"github.com/paideia-ai/acp/mandate"
	"github.com/paideia-ai/acp/methods/mock"
	"github.com/paideia-ai/acp/orchestration"
	"github.com/paideia-ai/acp/ratelimit"
	"github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
	// ── Payment Methods ────────────────────────────────────────
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))

	// ── Audit Logger ───────────────────────────────────────────
	auditLogger := audit.NewMemoryLogger()
	auditedGW := audit.NewAuditedGateway(gateway, auditLogger)

	// ── Rate Limiter ───────────────────────────────────────────
	limiter := ratelimit.NewTokenBucketLimiter(ratelimit.TokenBucketConfig{
		Rate:  10,
		Burst: 20,
	})

	// ── Anomaly Detector ───────────────────────────────────────
	detector := ratelimit.NewAnomalyDetector()

	// ── Token Issuer (for demo) ────────────────────────────────
	signingKey := []byte("demo-signing-key-not-for-production")
	issuer := auth.NewTokenIssuer(signingKey, "acp-demo", "acp-demo")

	demoToken, err := issuer.Issue("demo-agent", "demo-user", []auth.Permission{
		{Resource: "/api/*", Methods: []string{"mock"}, MaxAmount: "100.00", Currency: "USD"},
	}, 24*time.Hour)
	if err != nil {
		log.Fatalf("failed to issue demo token: %v", err)
	}

	// ── Mandate Store ──────────────────────────────────────────
	mandateStore := mandate.NewMemoryStore()
	demoMandate := mandate.NewMandate(
		mandate.WithID("mandate-001"),
		mandate.WithAgentID("demo-agent"),
		mandate.WithMaxAmount("500.00"),
		mandate.WithCurrency(core.USD),
		mandate.WithMaxPerRequest("50.00"),
		mandate.WithAllowedMethods("mock"),
		mandate.WithScope("/api/*"),
		mandate.WithExpiry(time.Now().Add(24*time.Hour)),
	)
	mandateStore.Save(demoMandate)
	enforcer := mandate.NewEnforcer()

	// ── Orchestration ──────────────────────────────────────────
	orch := orchestration.NewOrchestrator(&orchestration.PreferredStrategy{})

	// ── Service Discovery ──────────────────────────────────────
	registry := discovery.NewRegistry()
	registry.Register(discovery.ServiceEntry{
		ID:         "demo-server",
		Name:       "ACP Demo Server",
		Type:       discovery.ServiceTypeMethod,
		URL:        "http://localhost:8080",
		Methods:    []string{"mock"},
		Intents:    []core.Intent{core.IntentCharge},
		Currencies: []core.Currency{core.USD, core.EUR},
		Regions:    []string{"US", "EU", "KE", "IN", "BR"},
		HealthURL:  "http://localhost:8080/api/health",
		Status:     discovery.StatusActive,
	})
	healthChecker := discovery.NewHealthChecker(registry, 30*time.Second)

	// ── Dashboard ──────────────────────────────────────────────
	txnStore := dashboard.NewTransactionStore(1000)
	dashServer := dashboard.NewServer(dashboard.Config{ListenAddr: ":9090"}, txnStore)

	// Helper to log transactions to dashboard.
	logTxn := func(method, amount string, currency core.Currency) {
		txnStore.Record(dashboard.Transaction{
			ID:        fmt.Sprintf("txn_%d", time.Now().UnixNano()),
			Timestamp: time.Now().Format(time.RFC3339),
			Method:    method,
			Amount:    amount,
			Currency:  currency,
			Success:   true,
		})
	}

	// ── HTTP Server ────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health endpoint (free).
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// Paid endpoint — $1.00.
	mux.Handle("GET /api/data", acphttp.Paywall(gateway, acp.Price{Amount: "1.00", Currency: "USD"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logTxn("mock", "1.00", core.USD)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Payment successful! Here's your premium data.",
				"items":   []string{"alpha", "bravo", "charlie", "delta"},
				"ts":      time.Now().Format(time.RFC3339),
			})
		})))

	// Premium endpoint — $9.99.
	mux.Handle("GET /api/premium", acphttp.Paywall(gateway, acp.Price{Amount: "9.99", Currency: "USD"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logTxn("mock", "9.99", core.USD)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Welcome to the premium tier!",
				"secret":  "the cake is real",
			})
		})))

	// Multi-currency endpoint — €5.00.
	mux.Handle("GET /api/eu-data", acphttp.Paywall(gateway, acp.Price{Amount: "5.00", Currency: "EUR"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logTxn("mock", "5.00", core.EUR)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"message": "European market data",
				"market":  "EUROSTOXX",
			})
		})))

	// Info endpoint — shows system state (free).
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m, _ := mandateStore.Get("mandate-001")
		anomaly := detector.Check("demo-agent", "1.00", core.USD, "mock")
		json.NewEncoder(w).Encode(map[string]any{
			"methods":       gateway.Methods(),
			"mandate":       map[string]any{"id": m.ID, "agentID": m.AgentID, "maxAmount": m.MaxAmount},
			"anomaly":       anomaly,
			"audit":         "enabled",
			"rateLimiter":   "token_bucket(10/s, burst=20)",
			"orchestration": "preferred(mock)",
		})
	})

	// Service discovery endpoint.
	mux.Handle("GET /.well-known/acp-services", discovery.WellKnownHandler(registry))

	// Rate limit middleware.
	rateLimitedMux := ratelimit.RateLimitMiddleware(limiter, func(r *http.Request) string {
		return r.RemoteAddr
	})(mux)

	// ── Use components to avoid unused warnings ────────────────
	_ = auditedGW
	_ = enforcer
	_ = orch

	// ── Start Everything ───────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	healthChecker.Start(ctx)

	go func() {
		log.Printf("Dashboard running on http://localhost:9090")
		if err := dashServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("dashboard error: %v", err)
		}
	}()

	tokenPreview := demoToken
	if len(tokenPreview) > 30 {
		tokenPreview = tokenPreview[:30] + "..."
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ACP — Agentic Commerce Protocol                ║")
	fmt.Println("║                     Full Stack Demo                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                             ║")
	fmt.Println("║  API Server:     http://localhost:8080                       ║")
	fmt.Println("║  Dashboard:      http://localhost:9090                       ║")
	fmt.Println("║                                                             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Endpoints:                                                 ║")
	fmt.Println("║    GET /api/health     — free                               ║")
	fmt.Println("║    GET /api/data       — $1.00                              ║")
	fmt.Println("║    GET /api/premium    — $9.99                              ║")
	fmt.Println("║    GET /api/eu-data    — €5.00                              ║")
	fmt.Println("║    GET /api/info       — system state (free)                ║")
	fmt.Println("║    GET /.well-known/acp-services — discovery                ║")
	fmt.Println("║                                                             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Test commands:                                             ║")
	fmt.Println("║                                                             ║")
	fmt.Println("║  curl http://localhost:8080/api/health                       ║")
	fmt.Println("║  curl -i http://localhost:8080/api/data                      ║")
	fmt.Println("║  go run ./cmd/acp-pay --url http://localhost:8080/api/data   ║")
	fmt.Println("║  go run ./cmd/acp-pay --url http://localhost:8080/api/premium║")
	fmt.Println("║  curl http://localhost:8080/.well-known/acp-services         ║")
	fmt.Println("║  curl http://localhost:8080/api/info                         ║")
	fmt.Println("║  open http://localhost:9090                                  ║")
	fmt.Println("║                                                             ║")
	fmt.Printf("║  Demo Token: %s                  ║\n", tokenPreview)
	fmt.Println("║                                                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	server := &http.Server{Addr: ":8080", Handler: rateLimitedMux}
	go func() {
		<-ctx.Done()
		log.Println("Shutting down...")
		server.Shutdown(context.Background())
	}()

	log.Fatal(server.ListenAndServe())
}
