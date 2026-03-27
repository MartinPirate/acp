package acpgrpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptorNoPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{})))
	interceptor := UnaryServerInterceptor(gateway, acp.Price{Amount: "1.00", Currency: core.USD})

	// Create a context that supports grpc.SetHeader by using a mock stream.
	ctx := grpc.NewContextWithServerTransportStream(
		context.Background(),
		&mockServerTransportStream{},
	)

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	handler := func(ctx context.Context, req any) (any, error) {
		t.Error("handler should not be called without payment")
		return nil, nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected error for missing payment")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", st.Code())
	}
}

func TestUnaryServerInterceptorWithPayment(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	gateway := acp.NewGateway(acp.WithMethod(mockMethod))
	interceptor := UnaryServerInterceptor(gateway, acp.Price{Amount: "1.00", Currency: core.USD})

	// Build payment payload.
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
	payloadJSON, _ := json.Marshal(payload)

	// Set up incoming metadata with payment.
	md := metadata.Pairs(MetaKeyPayment, string(payloadJSON))
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = grpc.NewContextWithServerTransportStream(ctx, &mockServerTransportStream{})

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "result", nil
	}

	result, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	if result != "result" {
		t.Errorf("result = %v, want 'result'", result)
	}

	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
}

func TestUnaryServerInterceptorInvalidPayment(t *testing.T) {
	gateway := acp.NewGateway(acp.WithMethod(mock.New(mock.Config{ShouldFail: true})))
	interceptor := UnaryServerInterceptor(gateway, acp.Price{Amount: "1.00", Currency: core.USD})

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
	payloadJSON, _ := json.Marshal(payload)

	md := metadata.Pairs(MetaKeyPayment, string(payloadJSON))
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = grpc.NewContextWithServerTransportStream(ctx, &mockServerTransportStream{})

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	handler := func(ctx context.Context, req any) (any, error) {
		t.Error("handler should not be called when payment fails")
		return nil, nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected error for failed payment")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", st.Code())
	}
}

func TestExtractPaymentFromContext(t *testing.T) {
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
	payloadJSON, _ := json.Marshal(payload)

	md := metadata.Pairs(MetaKeyPayment, string(payloadJSON))
	ctx := metadata.NewIncomingContext(context.Background(), md)

	extracted, err := extractPaymentFromContext(ctx)
	if err != nil {
		t.Fatalf("extractPaymentFromContext error: %v", err)
	}
	if extracted == nil {
		t.Fatal("expected non-nil payload")
	}
	if extracted.Accepted.Method != "mock" {
		t.Errorf("method = %q, want mock", extracted.Accepted.Method)
	}
}

func TestExtractPaymentFromContextAbsent(t *testing.T) {
	ctx := context.Background()
	extracted, err := extractPaymentFromContext(ctx)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if extracted != nil {
		t.Error("expected nil payload when no metadata")
	}
}

// mockServerTransportStream implements grpc.ServerTransportStream for testing.
type mockServerTransportStream struct {
	headers metadata.MD
}

func (s *mockServerTransportStream) Method() string                  { return "/test.Service/Method" }
func (s *mockServerTransportStream) SetHeader(md metadata.MD) error  { s.headers = md; return nil }
func (s *mockServerTransportStream) SendHeader(md metadata.MD) error { return nil }
func (s *mockServerTransportStream) SetTrailer(md metadata.MD) error { return nil }
