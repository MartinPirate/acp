package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func TestRegistryRegisterAndDiscover(t *testing.T) {
	r := NewRegistry()
	entry := ServiceEntry{
		ID:         "svc-1",
		Name:       "Mock Payments",
		Type:       ServiceTypeMethod,
		URL:        "https://mock.example.com",
		Methods:    []string{"mock"},
		Intents:    []core.Intent{core.IntentCharge},
		Currencies: []core.Currency{core.USD},
		Regions:    []string{"US"},
		Status:     StatusActive,
	}
	r.Register(entry)

	results := r.Discover(Query{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "svc-1" {
		t.Errorf("expected ID svc-1, got %s", results[0].ID)
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "svc-1", Name: "A", Status: StatusActive})
	r.Register(ServiceEntry{ID: "svc-2", Name: "B", Status: StatusActive})

	r.Unregister("svc-1")

	results := r.All()
	if len(results) != 1 {
		t.Fatalf("expected 1 result after unregister, got %d", len(results))
	}
	if results[0].ID != "svc-2" {
		t.Errorf("expected svc-2, got %s", results[0].ID)
	}
}

func TestDiscoverByMethod(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Methods: []string{"card", "upi"}, Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Methods: []string{"pix"}, Status: StatusActive})

	results := r.Discover(Query{Methods: []string{"upi"}})
	if len(results) != 1 || results[0].ID != "1" {
		t.Errorf("expected service 1 for UPI query, got %v", results)
	}
}

func TestDiscoverByIntent(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Intents: []core.Intent{core.IntentCharge}, Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Intents: []core.Intent{core.IntentSubscribe}, Status: StatusActive})

	results := r.Discover(Query{Intents: []core.Intent{core.IntentSubscribe}})
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("expected service 2 for subscribe query, got %v", results)
	}
}

func TestDiscoverByCurrency(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Currencies: []core.Currency{core.USD, core.EUR}, Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Currencies: []core.Currency{core.INR}, Status: StatusActive})

	results := r.Discover(Query{Currencies: []core.Currency{core.INR}})
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("expected service 2 for INR query, got %v", results)
	}
}

func TestDiscoverByRegion(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Regions: []string{"US", "GB"}, Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Regions: []string{"IN"}, Status: StatusActive})

	results := r.Discover(Query{Regions: []string{"IN"}})
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("expected service 2 for IN region, got %v", results)
	}
}

func TestDiscoverByStatus(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Status: StatusOffline})

	results := r.Discover(Query{Status: StatusActive})
	if len(results) != 1 || results[0].ID != "1" {
		t.Errorf("expected only active service, got %v", results)
	}
}

func TestDiscoverByAmountRange(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "small", MinAmount: "0.01", MaxAmount: "100", Status: StatusActive})
	r.Register(ServiceEntry{ID: "large", MinAmount: "100", MaxAmount: "10000", Status: StatusActive})

	// Query for amounts around 50 -- small qualifies, large does not (min 100 > max 50)
	results := r.Discover(Query{MaxAmount: "50"})
	if len(results) != 1 || results[0].ID != "small" {
		t.Errorf("expected only small service, got %v", results)
	}
}

func TestUpdateStatus(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Status: StatusActive})

	r.UpdateStatus("1", StatusDegraded)
	entry, ok := r.Get("1")
	if !ok {
		t.Fatal("service not found")
	}
	if entry.Status != StatusDegraded {
		t.Errorf("expected degraded, got %s", entry.Status)
	}
	if entry.LastChecked.IsZero() {
		t.Error("expected LastChecked to be set")
	}
}

func TestHealthCheckerPing(t *testing.T) {
	// Start a test HTTP server that returns 200.
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	// Start a test HTTP server that returns 500.
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthy.Close()

	r := NewRegistry()
	r.Register(ServiceEntry{ID: "h", HealthURL: healthy.URL, Status: StatusActive})
	r.Register(ServiceEntry{ID: "u", HealthURL: unhealthy.URL, Status: StatusActive})

	hc := NewHealthChecker(r, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	hc.Start(ctx)

	// Give the health checker time to run at least once.
	time.Sleep(300 * time.Millisecond)
	cancel()
	hc.Stop()

	he, _ := r.Get("h")
	if he.Status != StatusActive {
		t.Errorf("expected healthy service to be active, got %s", he.Status)
	}

	ue, _ := r.Get("u")
	if ue.Status != StatusOffline {
		t.Errorf("expected unhealthy service to be offline, got %s", ue.Status)
	}
}

func TestWellKnownHandler(t *testing.T) {
	r := NewRegistry()
	r.Register(ServiceEntry{ID: "1", Name: "Active", Status: StatusActive})
	r.Register(ServiceEntry{ID: "2", Name: "Offline", Status: StatusOffline})

	handler := WellKnownHandler(r)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/acp-services", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var entries []ServiceEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 active entry, got %d", len(entries))
	}
	if entries[0].ID != "1" {
		t.Errorf("expected active service, got %s", entries[0].ID)
	}
}
