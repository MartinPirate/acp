package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func seedStore() *TransactionStore {
	store := NewTransactionStore(100)
	store.Record(Transaction{ID: "tx-1", Method: "card", Amount: "10.00", Currency: core.USD, Success: true, Timestamp: "2026-03-27T10:00:00Z"})
	store.Record(Transaction{ID: "tx-2", Method: "card", Amount: "20.00", Currency: core.USD, Success: true, Timestamp: "2026-03-27T10:01:00Z"})
	store.Record(Transaction{ID: "tx-3", Method: "upi", Amount: "5.00", Currency: core.INR, Success: false, Timestamp: "2026-03-27T10:02:00Z"})
	return store
}

func TestStatsEndpoint(t *testing.T) {
	store := seedStore()
	srv := NewServer(Config{ListenAddr: ":0"}, store)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stats Stats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if stats.TotalTransactions != 3 {
		t.Errorf("expected 3 transactions, got %d", stats.TotalTransactions)
	}

	// 2 out of 3 succeeded
	expected := 2.0 / 3.0
	if diff := stats.SuccessRate - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("expected success rate ~%.2f, got %.2f", expected, stats.SuccessRate)
	}
}

func TestTransactionsEndpoint(t *testing.T) {
	store := seedStore()
	srv := NewServer(Config{ListenAddr: ":0"}, store)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions?limit=2", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Transactions []Transaction `json:"transactions"`
		Count        int           `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Count != 2 {
		t.Errorf("expected 2 transactions, got %d", resp.Count)
	}

	// Most recent first
	if resp.Transactions[0].ID != "tx-3" {
		t.Errorf("expected tx-3 first (most recent), got %s", resp.Transactions[0].ID)
	}
}

func TestMethodsEndpoint(t *testing.T) {
	store := NewTransactionStore(10)
	srv := NewServer(Config{ListenAddr: ":0"}, store)
	srv.SetMethods([]MethodInfo{
		{Name: "card", Status: "active"},
		{Name: "upi", Status: "active"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/methods", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var methods []MethodInfo
	if err := json.NewDecoder(w.Body).Decode(&methods); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(methods) != 2 {
		t.Errorf("expected 2 methods, got %d", len(methods))
	}
}

func TestHealthEndpoint(t *testing.T) {
	store := NewTransactionStore(10)
	srv := NewServer(Config{ListenAddr: ":0"}, store)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var health map[string]any
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("expected status ok, got %v", health["status"])
	}
}

func TestDashboardHTML(t *testing.T) {
	store := NewTransactionStore(10)
	srv := NewServer(Config{ListenAddr: ":0"}, store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	body := w.Body.String()
	if len(body) < 100 {
		t.Error("expected substantial HTML content")
	}
}

func TestTransactionStoreEviction(t *testing.T) {
	store := NewTransactionStore(3)
	store.Record(Transaction{ID: "1"})
	store.Record(Transaction{ID: "2"})
	store.Record(Transaction{ID: "3"})
	store.Record(Transaction{ID: "4"}) // should evict "1"

	recent := store.Recent(10)
	if len(recent) != 3 {
		t.Fatalf("expected 3, got %d", len(recent))
	}
	// Most recent first
	if recent[0].ID != "4" {
		t.Errorf("expected 4 first, got %s", recent[0].ID)
	}
	if recent[2].ID != "2" {
		t.Errorf("expected 2 last, got %s", recent[2].ID)
	}
}

func TestTransactionLimitClamped(t *testing.T) {
	store := seedStore()
	srv := NewServer(Config{ListenAddr: ":0"}, store)

	// Request with absurdly high limit -- should be clamped to 1000
	req := httptest.NewRequest(http.MethodGet, "/api/transactions?limit=999999", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
