package acpa2a

import (
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp/core"
)

func TestTaskPaymentRequired(t *testing.T) {
	task := &Task{ID: "task-1", Status: "working"}
	pr := &core.PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource:   core.Resource{URL: "a2a://task/task-1", Description: "A2A task"},
		Accepts: []core.PaymentOption{
			{Intent: core.IntentCharge, Method: "mock", Currency: core.USD, Amount: "2.00"},
		},
	}

	TaskPaymentRequired(task, pr)

	if task.Status != "input-required" {
		t.Errorf("status = %q, want input-required", task.Status)
	}
	if GetTaskPaymentStatus(task) != StatusPaymentRequired {
		t.Errorf("payment status = %q, want %q", GetTaskPaymentStatus(task), StatusPaymentRequired)
	}
	if task.Metadata[MetaKeyPaymentRequired] == nil {
		t.Error("expected payment requirements in metadata")
	}
}

func TestExtractTaskPayment(t *testing.T) {
	payload := core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Accepted: core.PaymentOption{
			Intent:   core.IntentCharge,
			Method:   "mock",
			Currency: core.USD,
			Amount:   "1.00",
		},
		Payload: json.RawMessage(`{"token":"test"}`),
	}

	task := &Task{ID: "task-1", Status: "working"}
	AttachTaskPayment(task, &payload)

	extracted, err := ExtractTaskPayment(task)
	if err != nil {
		t.Fatalf("ExtractTaskPayment error: %v", err)
	}
	if extracted == nil {
		t.Fatal("expected non-nil payload")
	}
	if extracted.Accepted.Method != "mock" {
		t.Errorf("method = %q, want mock", extracted.Accepted.Method)
	}
	if GetTaskPaymentStatus(task) != StatusPaymentSubmitted {
		t.Errorf("payment status = %q, want %q", GetTaskPaymentStatus(task), StatusPaymentSubmitted)
	}
}

func TestExtractTaskPaymentAbsent(t *testing.T) {
	task := &Task{ID: "task-1", Status: "working"}
	extracted, err := ExtractTaskPayment(task)
	if err != nil {
		t.Fatalf("ExtractTaskPayment error: %v", err)
	}
	if extracted != nil {
		t.Error("expected nil payload when no metadata")
	}
}

func TestUpdateTaskPaymentStatus(t *testing.T) {
	task := &Task{ID: "task-1", Status: "working"}

	UpdateTaskPaymentStatus(task, StatusPaymentVerified)
	if GetTaskPaymentStatus(task) != StatusPaymentVerified {
		t.Errorf("payment status = %q, want %q", GetTaskPaymentStatus(task), StatusPaymentVerified)
	}

	UpdateTaskPaymentStatus(task, StatusPaymentFailed)
	if GetTaskPaymentStatus(task) != StatusPaymentFailed {
		t.Errorf("payment status = %q, want %q", GetTaskPaymentStatus(task), StatusPaymentFailed)
	}
}

func TestAttachTaskPaymentResponse(t *testing.T) {
	task := &Task{ID: "task-1", Status: "working"}
	sr := &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "mock",
		Transaction: "txn_123",
	}

	AttachTaskPaymentResponse(task, sr)

	if GetTaskPaymentStatus(task) != StatusPaymentCompleted {
		t.Errorf("payment status = %q, want %q", GetTaskPaymentStatus(task), StatusPaymentCompleted)
	}
	if task.Metadata[MetaKeyPaymentResponse] == nil {
		t.Error("expected payment response in metadata")
	}
}

func TestExtractPaymentRequired(t *testing.T) {
	task := &Task{ID: "task-1", Status: "working"}
	pr := &core.PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource:   core.Resource{URL: "a2a://task/task-1"},
		Accepts: []core.PaymentOption{
			{Intent: core.IntentCharge, Method: "mock", Currency: core.USD, Amount: "1.00"},
		},
	}

	TaskPaymentRequired(task, pr)

	extracted, err := ExtractPaymentRequired(task)
	if err != nil {
		t.Fatalf("ExtractPaymentRequired error: %v", err)
	}
	if extracted == nil {
		t.Fatal("expected non-nil payment required")
	}
	if extracted.ACPVersion != core.ACPVersion {
		t.Errorf("acpVersion = %d, want %d", extracted.ACPVersion, core.ACPVersion)
	}
	if len(extracted.Accepts) != 1 {
		t.Fatalf("expected 1 payment option, got %d", len(extracted.Accepts))
	}
}

func TestGetTaskPaymentStatusEmpty(t *testing.T) {
	task := &Task{ID: "task-1"}
	if status := GetTaskPaymentStatus(task); status != "" {
		t.Errorf("expected empty status, got %q", status)
	}
}

func TestPaymentLifecycle(t *testing.T) {
	task := &Task{ID: "task-1", Status: "submitted"}

	// Step 1: Server marks task as payment-required.
	pr := &core.PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource:   core.Resource{URL: "a2a://task/task-1"},
		Accepts: []core.PaymentOption{
			{Intent: core.IntentCharge, Method: "mock", Currency: core.USD, Amount: "5.00"},
		},
	}
	TaskPaymentRequired(task, pr)
	if GetTaskPaymentStatus(task) != StatusPaymentRequired {
		t.Fatalf("step 1: status = %q", GetTaskPaymentStatus(task))
	}

	// Step 2: Client attaches payment.
	payload := &core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Accepted:   pr.Accepts[0],
		Payload:    json.RawMessage(`{"token":"test"}`),
	}
	AttachTaskPayment(task, payload)
	if GetTaskPaymentStatus(task) != StatusPaymentSubmitted {
		t.Fatalf("step 2: status = %q", GetTaskPaymentStatus(task))
	}

	// Step 3: Server verifies.
	UpdateTaskPaymentStatus(task, StatusPaymentVerified)
	if GetTaskPaymentStatus(task) != StatusPaymentVerified {
		t.Fatalf("step 3: status = %q", GetTaskPaymentStatus(task))
	}

	// Step 4: Server settles and attaches receipt.
	sr := &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "mock",
		Transaction: "txn_lifecycle",
	}
	AttachTaskPaymentResponse(task, sr)
	if GetTaskPaymentStatus(task) != StatusPaymentCompleted {
		t.Fatalf("step 4: status = %q", GetTaskPaymentStatus(task))
	}
}
