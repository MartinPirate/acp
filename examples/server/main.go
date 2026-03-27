// Example server demonstrating ACP paywall middleware.
//
// Run:
//
//	go run ./examples/server
//
// Then test with:
//
//	go run ./cmd/acp-pay --url http://localhost:8080/api/data
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/methods/mock"
	"github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
	gateway := acp.NewGateway(
		acp.WithMethod(mock.New(mock.Config{})),
	)

	mux := http.NewServeMux()

	// Free endpoint — no payment required.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Paid endpoint — requires $1.00 payment.
	mux.Handle("GET /api/data", acphttp.Paywall(gateway, acp.Price{
		Amount:   "1.00",
		Currency: "USD",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": "You paid for this premium data!",
			"items":   []string{"alpha", "bravo", "charlie"},
		})
	})))

	// Premium endpoint — requires $9.99 payment.
	mux.Handle("GET /api/premium", acphttp.Paywall(gateway, acp.Price{
		Amount:   "9.99",
		Currency: "USD",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Welcome to the premium tier!",
			"secret":  "the cake is not a lie",
		})
	})))

	addr := ":8080"
	fmt.Printf("ACP example server running on %s\n\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/health   — free")
	fmt.Println("  GET /api/data     — $1.00")
	fmt.Println("  GET /api/premium  — $9.99")
	fmt.Println()
	fmt.Println("Test with:")
	fmt.Println("  curl http://localhost:8080/api/health")
	fmt.Println("  curl http://localhost:8080/api/data        # returns 402")
	fmt.Println("  go run ./cmd/acp-pay --url http://localhost:8080/api/data")

	log.Fatal(http.ListenAndServe(addr, mux))
}
