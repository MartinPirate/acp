// acp-facilitator is a reference facilitator server that exposes /verify,
// /settle, and /supported endpoints for remote payment processing.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
)

func main() {
	addr := flag.String("addr", ":8181", "listen address")
	flag.Parse()

	gateway := acp.NewGateway(
		acp.WithMethod(mock.New(mock.Config{})),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /verify", handleVerify(gateway))
	mux.HandleFunc("POST /settle", handleSettle(gateway))
	mux.HandleFunc("GET /supported", handleSupported(gateway))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Printf("acp-facilitator listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

type facilitatorRequest struct {
	Payload      core.PaymentPayload  `json:"payload"`
	Requirements core.PaymentRequired `json:"requirements"`
}

func handleVerify(gateway *acp.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req facilitatorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		resp, err := gateway.Verify(r.Context(), req.Payload)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleSettle(gateway *acp.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req facilitatorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		resp, err := gateway.Settle(r.Context(), req.Payload)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleSupported(gateway *acp.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		methods := gateway.Methods()

		var intents []core.Intent
		var currencies []core.Currency
		seen := make(map[string]bool)

		for _, name := range methods {
			m, _ := gateway.Method(name)
			for _, i := range m.SupportedIntents() {
				key := "i:" + string(i)
				if !seen[key] {
					intents = append(intents, i)
					seen[key] = true
				}
			}
			for _, c := range m.SupportedCurrencies() {
				key := "c:" + string(c)
				if !seen[key] {
					currencies = append(currencies, c)
					seen[key] = true
				}
			}
		}

		writeJSON(w, http.StatusOK, core.SupportedResponse{
			Methods:    methods,
			Intents:    intents,
			Currencies: currencies,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
