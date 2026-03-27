package acpchi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
	"github.com/paideia-ai/acp/transport/acphttp"
)

func TestPaywallNoPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))
	mw := Paywall(gateway, acp.Price{Amount: "5.00", Currency: core.USD})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}

	prHeader := w.Result().Header.Get(acphttp.HeaderPaymentRequired)
	if prHeader == "" {
		t.Fatal("ACP-Payment-Required header missing")
	}
}

func TestPaywallWithPayment(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))
	mw := Paywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"data": "premium"})
	})

	handler := mw(inner)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := acphttp.NewClient(gateway)
	resp, err := client.Get(server.URL + "/api/data")
	if err != nil {
		t.Fatalf("client.Get error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
}

func TestPaywallMiddlewareSignature(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))
	mw := Paywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD})

	// Verify the return type matches Chi's middleware signature.
	var _ func(http.Handler) http.Handler = mw
}
