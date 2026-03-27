package audit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/paideia-ai/acp/core"
)

func TestMemoryLoggerLogAndGet(t *testing.T) {
	logger := NewMemoryLogger()
	entry := Entry{
		ID:        "e-1",
		Timestamp: time.Now(),
		AgentID:   "agent-1",
		Method:    "mock",
		Amount:    "10.00",
		Currency:  core.USD,
		Status:    StatusPending,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	got, err := logger.Get("e-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-1")
	}
	if got.Amount != "10.00" {
		t.Errorf("Amount = %q, want %q", got.Amount, "10.00")
	}
}

func TestMemoryLoggerLogEmptyID(t *testing.T) {
	logger := NewMemoryLogger()
	err := logger.Log(Entry{})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestMemoryLoggerGetNotFound(t *testing.T) {
	logger := NewMemoryLogger()
	_, err := logger.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestMemoryLoggerQuery(t *testing.T) {
	logger := NewMemoryLogger()
	now := time.Now()

	for i := 0; i < 10; i++ {
		agent := "agent-1"
		if i%3 == 0 {
			agent = "agent-2"
		}
		status := StatusSettled
		if i%2 == 0 {
			status = StatusFailed
		}
		logger.Log(Entry{
			ID:        fmt.Sprintf("q-%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			AgentID:   agent,
			Method:    "mock",
			Amount:    fmt.Sprintf("%d.00", i+1),
			Currency:  core.USD,
			Status:    status,
		})
	}

	// Filter by agent.
	results, err := logger.Query(Filter{AgentID: "agent-2"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	for _, r := range results {
		if r.AgentID != "agent-2" {
			t.Errorf("expected AgentID agent-2, got %q", r.AgentID)
		}
	}

	// Filter by status.
	results, err = logger.Query(Filter{Status: StatusSettled})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	for _, r := range results {
		if r.Status != StatusSettled {
			t.Errorf("expected status settled, got %q", r.Status)
		}
	}

	// Filter with limit and offset.
	results, err = logger.Query(Filter{Limit: 3, Offset: 2})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Filter by time range.
	results, err = logger.Query(Filter{
		TimeFrom: now.Add(3 * time.Minute),
		TimeTo:   now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	for _, r := range results {
		if r.Timestamp.Before(now.Add(3*time.Minute)) || r.Timestamp.After(now.Add(6*time.Minute)) {
			t.Errorf("entry timestamp %v outside range", r.Timestamp)
		}
	}
}

// mockGateway is a test double for GatewayInterface.
type mockGateway struct {
	verifyResp *core.VerifyResponse
	verifyErr  error
	settleResp *core.SettleResponse
	settleErr  error
}

func (g *mockGateway) BuildPaymentRequired(resource core.Resource, price core.Price) (*core.PaymentRequired, error) {
	return nil, nil
}
func (g *mockGateway) Verify(_ context.Context, _ core.PaymentPayload) (*core.VerifyResponse, error) {
	return g.verifyResp, g.verifyErr
}
func (g *mockGateway) Settle(_ context.Context, _ core.PaymentPayload) (*core.SettleResponse, error) {
	return g.settleResp, g.settleErr
}
func (g *mockGateway) Methods() []string                     { return nil }
func (g *mockGateway) Method(_ string) (core.Method, bool) { return nil, false }

func TestAuditedGatewayVerifySuccess(t *testing.T) {
	logger := NewMemoryLogger()
	inner := &mockGateway{
		verifyResp: &core.VerifyResponse{Valid: true, Payer: "user-1"},
	}
	gw := NewAuditedGateway(inner, logger)

	payload := core.PaymentPayload{
		Resource: core.Resource{URL: "/test"},
		Accepted: core.PaymentOption{
			Method:   "mock",
			Intent:   core.IntentCharge,
			Amount:   "5.00",
			Currency: core.USD,
		},
	}

	resp, err := gw.Verify(context.Background(), payload)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !resp.Valid {
		t.Error("expected valid response")
	}

	entry, err := logger.Get("audit-1")
	if err != nil {
		t.Fatalf("Get audit entry failed: %v", err)
	}
	if entry.Status != StatusVerified {
		t.Errorf("status = %q, want %q", entry.Status, StatusVerified)
	}
}

func TestAuditedGatewayVerifyFailure(t *testing.T) {
	logger := NewMemoryLogger()
	inner := &mockGateway{
		verifyErr: fmt.Errorf("payment declined"),
	}
	gw := NewAuditedGateway(inner, logger)

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00", Currency: core.USD},
	}

	_, err := gw.Verify(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error")
	}

	entry, err := logger.Get("audit-1")
	if err != nil {
		t.Fatalf("Get audit entry failed: %v", err)
	}
	if entry.Status != StatusFailed {
		t.Errorf("status = %q, want %q", entry.Status, StatusFailed)
	}
	if entry.Error == "" {
		t.Error("expected error message in audit entry")
	}
}

func TestAuditedGatewaySettleSuccess(t *testing.T) {
	logger := NewMemoryLogger()
	inner := &mockGateway{
		settleResp: &core.SettleResponse{Success: true, Transaction: "txn-123"},
	}
	gw := NewAuditedGateway(inner, logger)

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00", Currency: core.USD},
	}

	resp, err := gw.Settle(context.Background(), payload)
	if err != nil {
		t.Fatalf("Settle failed: %v", err)
	}
	if !resp.Success {
		t.Error("expected successful settlement")
	}

	entry, err := logger.Get("audit-1")
	if err != nil {
		t.Fatalf("Get audit entry failed: %v", err)
	}
	if entry.Status != StatusSettled {
		t.Errorf("status = %q, want %q", entry.Status, StatusSettled)
	}
	if entry.Transaction != "txn-123" {
		t.Errorf("transaction = %q, want %q", entry.Transaction, "txn-123")
	}
}

func TestAuditedGatewaySettleFailure(t *testing.T) {
	logger := NewMemoryLogger()
	inner := &mockGateway{
		settleErr: fmt.Errorf("network error"),
	}
	gw := NewAuditedGateway(inner, logger)

	payload := core.PaymentPayload{
		Accepted: core.PaymentOption{Method: "mock", Amount: "5.00", Currency: core.USD},
	}

	_, err := gw.Settle(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error")
	}

	entry, err := logger.Get("audit-1")
	if err != nil {
		t.Fatalf("Get audit entry failed: %v", err)
	}
	if entry.Status != StatusFailed {
		t.Errorf("status = %q, want %q", entry.Status, StatusFailed)
	}
}
