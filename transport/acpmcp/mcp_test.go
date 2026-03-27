package acpmcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
)

func TestPaymentRequiredResult(t *testing.T) {
	pr := &core.PaymentRequired{
		ACPVersion: core.ACPVersion,
		Resource:   core.Resource{URL: "mcp://tool/search", Description: "MCP tool: search"},
		Accepts: []core.PaymentOption{
			{Intent: core.IntentCharge, Method: "mock", Currency: core.USD, Amount: "1.00"},
		},
	}

	result := PaymentRequiredResult(pr)
	if !result.IsError {
		t.Error("expected IsError = true")
	}
	if len(result.Content) < 2 {
		t.Fatal("expected at least 2 content blocks")
	}
	if result.Metadata == nil {
		t.Fatal("expected metadata to be set")
	}
	if _, ok := result.Metadata[MetaKeyPayment]; !ok {
		t.Error("expected acp/payment key in metadata")
	}
}

func TestExtractPaymentPresent(t *testing.T) {
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

	// Convert to map[string]any (as it would arrive in JSON).
	raw, _ := json.Marshal(payload)
	var asMap map[string]any
	json.Unmarshal(raw, &asMap)

	params := ToolCallParams{
		Meta: map[string]any{
			MetaKeyPayment: asMap,
		},
	}

	extracted, err := ExtractPayment(params)
	if err != nil {
		t.Fatalf("ExtractPayment error: %v", err)
	}
	if extracted == nil {
		t.Fatal("expected non-nil payload")
	}
	if extracted.Accepted.Method != "mock" {
		t.Errorf("method = %q, want mock", extracted.Accepted.Method)
	}
}

func TestExtractPaymentAbsent(t *testing.T) {
	params := ToolCallParams{Arguments: map[string]any{"query": "test"}}
	extracted, err := ExtractPayment(params)
	if err != nil {
		t.Fatalf("ExtractPayment error: %v", err)
	}
	if extracted != nil {
		t.Error("expected nil payload when no metadata")
	}
}

func TestAttachPaymentResponse(t *testing.T) {
	result := &ToolResult{
		Content: []ContentBlock{{Type: "text", Text: "result"}},
	}
	sr := &core.SettleResponse{
		ACPVersion:  core.ACPVersion,
		Success:     true,
		Method:      "mock",
		Transaction: "txn_123",
	}

	AttachPaymentResponse(result, sr)

	if result.Metadata == nil {
		t.Fatal("expected metadata to be set")
	}
	if _, ok := result.Metadata[MetaKeyPaymentResponse]; !ok {
		t.Error("expected acp/payment-response key in metadata")
	}
}

func TestMCPPaywallNoPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))

	handler := MCPPaywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD}, "search",
		func(ctx context.Context, params ToolCallParams) (*ToolResult, error) {
			t.Error("handler should not be called without payment")
			return nil, nil
		},
	)

	result, err := handler(context.Background(), ToolCallParams{
		Arguments: map[string]any{"query": "test"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError = true for payment required")
	}
	if result.Metadata == nil {
		t.Fatal("expected metadata with payment requirements")
	}
}

func TestMCPPaywallWithPayment(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))

	handler := MCPPaywall(gateway, acp.Price{Amount: "1.00", Currency: core.USD}, "search",
		func(ctx context.Context, params ToolCallParams) (*ToolResult, error) {
			return &ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "search results"}},
			}, nil
		},
	)

	// Build a valid payment payload.
	payload := core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Accepted: core.PaymentOption{
			Intent:   core.IntentCharge,
			Method:   "mock",
			Currency: core.USD,
			Amount:   "1.00",
		},
		Payload: json.RawMessage(`{"token":"mock_tok_test"}`),
	}
	raw, _ := json.Marshal(payload)
	var asMap map[string]any
	json.Unmarshal(raw, &asMap)

	result, err := handler(context.Background(), ToolCallParams{
		Arguments: map[string]any{"query": "test"},
		Meta:      map[string]any{MetaKeyPayment: asMap},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError = false after successful payment")
	}
	if result.Content[0].Text != "search results" {
		t.Errorf("content = %q, want 'search results'", result.Content[0].Text)
	}
	if _, ok := result.Metadata[MetaKeyPaymentResponse]; !ok {
		t.Error("expected payment response in metadata")
	}

	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
}
