package acphttp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
)

func TestEncodeDecodeHeader(t *testing.T) {
	original := core.PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource: core.Resource{
			URL:         "https://example.com/api",
			Description: "Test",
		},
		Accepts: []core.PaymentOption{
			{
				Intent:   core.IntentCharge,
				Method:   "mock",
				Currency: core.USD,
				Amount:   "5.99",
			},
		},
	}

	encoded, err := EncodeHeader(original)
	if err != nil {
		t.Fatalf("EncodeHeader error: %v", err)
	}

	var decoded core.PaymentRequired
	if err := DecodeHeader(encoded, &decoded); err != nil {
		t.Fatalf("DecodeHeader error: %v", err)
	}

	if decoded.ACPVersion != original.ACPVersion {
		t.Errorf("acpVersion = %d, want %d", decoded.ACPVersion, original.ACPVersion)
	}
	if decoded.Resource.URL != original.Resource.URL {
		t.Errorf("resource.url = %q, want %q", decoded.Resource.URL, original.Resource.URL)
	}
}

func TestPaywallNoPayment(t *testing.T) {
	gateway := acp.NewGateway(
		acp.WithMethod(mock.New(mock.Config{})),
	)

	handler := Paywall(gateway, acp.Price{Amount: "5.99", Currency: "USD"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"data": "secret"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", resp.StatusCode)
	}

	// Check header is present.
	prHeader := resp.Header.Get(HeaderPaymentRequired)
	if prHeader == "" {
		t.Fatal("ACP-Payment-Required header missing")
	}

	// Decode body.
	var pr core.PaymentRequired
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatalf("decode body error: %v", err)
	}
	if pr.ACPVersion != core.ACPVersion {
		t.Errorf("acpVersion = %d, want %d", pr.ACPVersion, core.ACPVersion)
	}
	if len(pr.Accepts) == 0 {
		t.Fatal("accepts is empty")
	}
}

func TestEndToEndFlow(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))

	responseBody := map[string]string{"data": "premium content"}

	handler := Paywall(gateway, acp.Price{Amount: "1.00", Currency: "USD"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	// Create a client with the same mock method.
	client := NewClient(gateway)

	// Make the request — client auto-handles 402.
	resp, err := client.Get(server.URL + "/api/data")
	if err != nil {
		t.Fatalf("client.Get error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	// Check payment response header.
	prHeader := resp.Header.Get(HeaderPaymentResponse)
	if prHeader == "" {
		t.Error("ACP-Payment-Response header missing on 200 response")
	}

	// Decode the response body.
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response error: %v", err)
	}
	if result["data"] != "premium content" {
		t.Errorf("response data = %q, want 'premium content'", result["data"])
	}

	// Verify mock recorded the transaction.
	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
	if txns[0].Amount != "1.00" {
		t.Errorf("transaction amount = %q, want '1.00'", txns[0].Amount)
	}
}

func TestEndToEndWithBudget(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))

	handler := Paywall(gateway, acp.Price{Amount: "10.00", Currency: "USD"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	// Client with a budget that's too low.
	client := NewClient(gateway, WithBudget(core.Budget{
		MaxPerRequest: "5.00",
		Currency:      core.USD,
	}))

	_, err := client.Get(server.URL + "/api/data")
	if err == nil {
		t.Fatal("expected budget error")
	}
	if !core.IsPaymentError(err, core.ErrBudgetExceeded) {
		t.Errorf("expected ErrBudgetExceeded, got: %v", err)
	}
}

func TestPaywallInvalidPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{ShouldFail: true})))

	handler := Paywall(gateway, acp.Price{Amount: "1.00", Currency: "USD"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when payment fails")
	}))

	// Build a valid-looking but failing payment payload.
	option := core.PaymentOption{
		Intent:   core.IntentCharge,
		Method:   "mock",
		Currency: core.USD,
		Amount:   "1.00",
	}
	payload := core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Accepted:   option,
		Payload:    json.RawMessage(`{"token":"test"}`),
	}
	encoded, _ := EncodeHeader(payload)

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set(HeaderPayment, encoded)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", w.Code)
	}
}
