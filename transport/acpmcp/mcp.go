// Package acpmcp provides MCP (Model Context Protocol) transport bindings for the
// Agentic Commerce Protocol. It helps users integrate ACP payment flows into
// existing MCP tool servers.
package acpmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
)

// Metadata keys used by ACP within MCP tool calls.
const (
	MetaKeyPayment         = "acp/payment"
	MetaKeyPaymentResponse = "acp/payment-response"
)

// ToolResult represents an MCP tool call result.
type ToolResult struct {
	Content  []ContentBlock `json:"content"`
	IsError  bool           `json:"isError,omitempty"`
	Metadata map[string]any `json:"_meta,omitempty"`
}

// ContentBlock is a piece of content in an MCP tool result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data any    `json:"data,omitempty"`
}

// ToolCallParams represents the parameters of an MCP tool call, including metadata.
type ToolCallParams struct {
	Arguments map[string]any `json:"arguments,omitempty"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

// ToolHandler is a function that handles an MCP tool call.
type ToolHandler func(ctx context.Context, params ToolCallParams) (*ToolResult, error)

// PaymentRequiredResult creates an MCP tool error result indicating that
// payment is required. The payment requirements are included as structured
// content so the calling agent can parse and fulfill them.
func PaymentRequiredResult(pr *core.PaymentRequired) *ToolResult {
	prJSON, err := json.Marshal(pr)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []ContentBlock{{Type: "text", Text: "payment required but failed to encode requirements"}},
		}
	}

	return &ToolResult{
		IsError: true,
		Content: []ContentBlock{
			{Type: "text", Text: "Payment required to use this tool."},
			{Type: "text", Text: string(prJSON)},
		},
		Metadata: map[string]any{
			MetaKeyPayment: pr,
		},
	}
}

// ExtractPayment extracts a PaymentPayload from tool call parameters.
// Returns nil, nil if no payment metadata is present.
func ExtractPayment(params ToolCallParams) (*core.PaymentPayload, error) {
	if params.Meta == nil {
		return nil, nil
	}
	raw, ok := params.Meta[MetaKeyPayment]
	if !ok {
		return nil, nil
	}

	// Marshal then unmarshal to handle both map[string]any and pre-typed values.
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to marshal payment metadata: %w", err)
	}
	var payload core.PaymentPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("acp: failed to decode payment metadata: %w", err)
	}
	return &payload, nil
}

// AttachPaymentResponse adds settlement information to a tool result's metadata.
func AttachPaymentResponse(result *ToolResult, sr *core.SettleResponse) {
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[MetaKeyPaymentResponse] = sr
}

// MCPPaywall wraps a ToolHandler, requiring payment before executing it.
// The toolName is used to build the resource description.
func MCPPaywall(gateway *acp.Gateway, price acp.Price, toolName string, handler ToolHandler) ToolHandler {
	return func(ctx context.Context, params ToolCallParams) (*ToolResult, error) {
		// Try to extract payment from params.
		payload, err := ExtractPayment(params)
		if err != nil {
			return &ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: "invalid payment metadata: " + err.Error()}},
			}, nil
		}

		// No payment present -- return payment required.
		if payload == nil {
			resource := core.Resource{
				URL:         "mcp://tool/" + toolName,
				Description: "MCP tool: " + toolName,
			}
			pr, err := gateway.BuildPaymentRequired(resource, price)
			if err != nil {
				log.Printf("acp: failed to build payment requirements: %v", err)
				return &ToolResult{
					IsError: true,
					Content: []ContentBlock{{Type: "text", Text: "no payment methods available"}},
				}, nil
			}
			return PaymentRequiredResult(pr), nil
		}

		// Payment present -- verify.
		verifyResp, err := gateway.Verify(ctx, *payload)
		if err != nil {
			log.Printf("acp: verification error: %v", err)
			return &ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: "payment verification failed: " + err.Error()}},
			}, nil
		}
		if !verifyResp.Valid {
			reason := "payment verification failed"
			if verifyResp.Reason != "" {
				reason = verifyResp.Reason
			}
			return &ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: reason}},
			}, nil
		}

		// Verified -- settle.
		settleResp, err := gateway.Settle(ctx, *payload)
		if err != nil {
			log.Printf("acp: settlement error: %v", err)
			return &ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: "settlement failed: " + err.Error()}},
			}, nil
		}
		if !settleResp.Success {
			return &ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: "settlement unsuccessful"}},
			}, nil
		}

		// Execute the tool handler.
		result, err := handler(ctx, params)
		if err != nil {
			return nil, err
		}

		// Attach settlement receipt.
		AttachPaymentResponse(result, settleResp)
		return result, nil
	}
}
