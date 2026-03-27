package acpecho

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
	"github.com/paideia-ai/acp/transport/acphttp"
)

func TestPaywallNoPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))

	e := echo.New()
	e.Use(Paywall(gateway, acp.Price{Amount: "5.00", Currency: core.USD}))
	e.GET("/api/data", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}

	prHeader := w.Result().Header.Get(acphttp.HeaderPaymentRequired)
	if prHeader == "" {
		t.Fatal("ACP-Payment-Required header missing")
	}

	var pr core.PaymentRequired
	if err := json.NewDecoder(w.Body).Decode(&pr); err != nil {
		t.Fatalf("decode body error: %v", err)
	}
	if pr.ACPVersion != core.ACPVersion {
		t.Errorf("acpVersion = %d, want %d", pr.ACPVersion, core.ACPVersion)
	}
}

func TestPaywallWithValidPayment(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))

	e := echo.New()
	e.Use(Paywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD}))
	e.GET("/api/data", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"data": "premium"})
	})

	server := httptest.NewServer(e)
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

	receiptHeader := resp.Header.Get(acphttp.HeaderPaymentResponse)
	if receiptHeader == "" {
		t.Error("ACP-Payment-Response header missing")
	}

	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
}

func TestPaywallInvalidPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{ShouldFail: true})))

	e := echo.New()
	e.Use(Paywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD}))
	e.GET("/api/data", func(c echo.Context) error {
		t.Error("handler should not be called when payment fails")
		return nil
	})

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
	encoded, _ := acphttp.EncodeHeader(payload)

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set(acphttp.HeaderPayment, encoded)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", w.Code)
	}
}
