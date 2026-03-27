// Package acpa2a provides A2A (Agent-to-Agent) transport bindings for the
// Agentic Commerce Protocol. It helps users integrate ACP payment flows into
// existing A2A task-based systems.
package acpa2a

import (
	"encoding/json"
	"fmt"

	"github.com/paideia-ai/acp/core"
)

// Payment status values tracked in A2A task metadata.
const (
	StatusPaymentRequired  = "payment-required"
	StatusPaymentSubmitted = "payment-submitted"
	StatusPaymentVerified  = "payment-verified"
	StatusPaymentCompleted = "payment-completed"
	StatusPaymentFailed    = "payment-failed"
)

// Metadata keys used by ACP within A2A tasks.
const (
	MetaKeyPaymentStatus   = "acp.payment.status"
	MetaKeyPaymentRequired = "acp.payment.required"
	MetaKeyPayment         = "acp.payment"
	MetaKeyPaymentResponse = "acp.payment.response"
)

// Task represents an A2A task with metadata for payment tracking.
type Task struct {
	ID       string         `json:"id"`
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Input    any            `json:"input,omitempty"`
	Output   any            `json:"output,omitempty"`
}

// TaskPaymentRequired marks a task as requiring payment. It sets the task
// status to "input-required" and stores payment requirements in metadata.
func TaskPaymentRequired(task *Task, pr *core.PaymentRequired) {
	if task.Metadata == nil {
		task.Metadata = make(map[string]any)
	}
	task.Status = "input-required"
	task.Metadata[MetaKeyPaymentStatus] = StatusPaymentRequired
	task.Metadata[MetaKeyPaymentRequired] = pr
}

// ExtractTaskPayment extracts a PaymentPayload from task metadata.
// Returns nil, nil if no payment is present.
func ExtractTaskPayment(task *Task) (*core.PaymentPayload, error) {
	if task.Metadata == nil {
		return nil, nil
	}
	raw, ok := task.Metadata[MetaKeyPayment]
	if !ok {
		return nil, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to marshal task payment: %w", err)
	}
	var payload core.PaymentPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("acp: failed to decode task payment: %w", err)
	}
	return &payload, nil
}

// UpdateTaskPaymentStatus updates the payment lifecycle status of a task.
func UpdateTaskPaymentStatus(task *Task, status string) {
	if task.Metadata == nil {
		task.Metadata = make(map[string]any)
	}
	task.Metadata[MetaKeyPaymentStatus] = status
}

// GetTaskPaymentStatus returns the current payment status from task metadata.
// Returns an empty string if no payment status is set.
func GetTaskPaymentStatus(task *Task) string {
	if task.Metadata == nil {
		return ""
	}
	s, _ := task.Metadata[MetaKeyPaymentStatus].(string)
	return s
}

// AttachTaskPayment attaches a PaymentPayload to a task's metadata.
// This is called by the client/agent after building the payment.
func AttachTaskPayment(task *Task, payload *core.PaymentPayload) {
	if task.Metadata == nil {
		task.Metadata = make(map[string]any)
	}
	task.Metadata[MetaKeyPayment] = payload
	task.Metadata[MetaKeyPaymentStatus] = StatusPaymentSubmitted
}

// AttachTaskPaymentResponse attaches a settlement response to a task's metadata.
func AttachTaskPaymentResponse(task *Task, sr *core.SettleResponse) {
	if task.Metadata == nil {
		task.Metadata = make(map[string]any)
	}
	task.Metadata[MetaKeyPaymentResponse] = sr
	task.Metadata[MetaKeyPaymentStatus] = StatusPaymentCompleted
}

// ExtractPaymentRequired extracts payment requirements from task metadata.
// Returns nil, nil if no requirements are present.
func ExtractPaymentRequired(task *Task) (*core.PaymentRequired, error) {
	if task.Metadata == nil {
		return nil, nil
	}
	raw, ok := task.Metadata[MetaKeyPaymentRequired]
	if !ok {
		return nil, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to marshal payment required: %w", err)
	}
	var pr core.PaymentRequired
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("acp: failed to decode payment required: %w", err)
	}
	return &pr, nil
}
