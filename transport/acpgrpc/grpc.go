// Package acpgrpc provides gRPC transport bindings for the Agentic Commerce Protocol.
// It implements unary server and client interceptors that handle payment flows
// using gRPC metadata.
package acpgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Metadata keys used by ACP within gRPC calls.
const (
	MetaKeyPaymentRequired = "acp-payment-required"
	MetaKeyPayment         = "acp-payment"
	MetaKeyPaymentResponse = "acp-payment-response"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that requires
// payment before allowing the RPC to proceed. It uses gRPC metadata to exchange
// payment data and returns codes.FailedPrecondition (closest to HTTP 402) when
// payment is required.
func UnaryServerInterceptor(gateway *acp.Gateway, price acp.Price) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Try to read payment from metadata.
		payload, err := extractPaymentFromContext(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid acp-payment metadata: %v", err)
		}

		// No payment present -- return FailedPrecondition with payment requirements.
		if payload == nil {
			resource := core.Resource{
				URL:         "grpc://" + info.FullMethod,
				Description: info.FullMethod,
			}
			pr, err := gateway.BuildPaymentRequired(resource, price)
			if err != nil {
				log.Printf("acp: failed to build payment requirements: %v", err)
				return nil, status.Errorf(codes.Internal, "no payment methods available")
			}
			prJSON, err := json.Marshal(pr)
			if err != nil {
				log.Printf("acp: failed to marshal payment requirements: %v", err)
				return nil, status.Errorf(codes.Internal, "failed to encode payment requirements")
			}

			// Send payment requirements via response metadata.
			md := metadata.Pairs(MetaKeyPaymentRequired, string(prJSON))
			if err := grpc.SetHeader(ctx, md); err != nil {
				log.Printf("acp: failed to set header metadata: %v", err)
			}
			return nil, status.Errorf(codes.FailedPrecondition, "payment required: %s", string(prJSON))
		}

		// Payment present -- verify.
		verifyResp, err := gateway.Verify(ctx, *payload)
		if err != nil {
			log.Printf("acp: verification error: %v", err)
			return nil, status.Errorf(codes.FailedPrecondition, "payment verification failed: %v", err)
		}
		if !verifyResp.Valid {
			reason := "payment verification failed"
			if verifyResp.Reason != "" {
				reason = verifyResp.Reason
			}
			return nil, status.Errorf(codes.FailedPrecondition, "%s", reason)
		}

		// Verified -- settle.
		settleResp, err := gateway.Settle(ctx, *payload)
		if err != nil {
			log.Printf("acp: settlement error: %v", err)
			return nil, status.Errorf(codes.FailedPrecondition, "settlement failed: %v", err)
		}
		if !settleResp.Success {
			return nil, status.Errorf(codes.FailedPrecondition, "settlement unsuccessful")
		}

		// Write settlement receipt via response metadata.
		srJSON, err := json.Marshal(settleResp)
		if err != nil {
			log.Printf("acp: failed to marshal settlement response: %v", err)
		} else {
			md := metadata.Pairs(MetaKeyPaymentResponse, string(srJSON))
			if err := grpc.SetHeader(ctx, md); err != nil {
				log.Printf("acp: failed to set payment response header: %v", err)
			}
		}

		return handler(ctx, req)
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor that handles
// payment flows. When a FailedPrecondition with payment requirements is received,
// it automatically builds a payment and retries the RPC.
func UnaryClientInterceptor(gateway *acp.Gateway) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Capture response metadata for payment requirements.
		var headerMD metadata.MD
		opts = append(opts, grpc.Header(&headerMD))

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		// Check if the error is a payment-required response.
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.FailedPrecondition {
			return err
		}

		// Extract payment requirements from metadata.
		pr, extractErr := extractPaymentRequired(headerMD)
		if extractErr != nil {
			return fmt.Errorf("acp: failed to extract payment requirements: %w", extractErr)
		}
		if pr == nil {
			// Not an ACP payment error, return original error.
			return err
		}

		// Select a payment method.
		var selected *core.PaymentOption
		for i := range pr.Accepts {
			if _, has := gateway.Method(pr.Accepts[i].Method); has {
				selected = &pr.Accepts[i]
				break
			}
		}
		if selected == nil {
			return core.NewPaymentError(core.ErrMethodUnavailable, "no compatible payment method found")
		}

		// Create the payment payload.
		m, _ := gateway.Method(selected.Method)
		methodPayload, err := m.CreatePayload(ctx, *selected)
		if err != nil {
			return fmt.Errorf("acp: failed to create payment payload: %w", err)
		}

		paymentPayload := core.PaymentPayload{
			ACPVersion: core.ACPVersion,
			Resource:   pr.Resource,
			Accepted:   *selected,
			Payload:    methodPayload,
		}
		payloadJSON, err := json.Marshal(paymentPayload)
		if err != nil {
			return fmt.Errorf("acp: failed to marshal payment payload: %w", err)
		}

		// Retry with payment in metadata.
		md := metadata.Pairs(MetaKeyPayment, string(payloadJSON))
		payCtx := metadata.NewOutgoingContext(ctx, md)

		return invoker(payCtx, method, req, reply, cc, opts...)
	}
}

// extractPaymentFromContext reads the ACP payment payload from incoming gRPC metadata.
func extractPaymentFromContext(ctx context.Context) (*core.PaymentPayload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	vals := md.Get(MetaKeyPayment)
	if len(vals) == 0 {
		return nil, nil
	}
	var payload core.PaymentPayload
	if err := json.Unmarshal([]byte(vals[0]), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// extractPaymentRequired reads payment requirements from response metadata.
func extractPaymentRequired(md metadata.MD) (*core.PaymentRequired, error) {
	vals := md.Get(MetaKeyPaymentRequired)
	if len(vals) == 0 {
		return nil, nil
	}
	var pr core.PaymentRequired
	if err := json.Unmarshal([]byte(vals[0]), &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
