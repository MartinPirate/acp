package acp

import (
	"context"
	"testing"

	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/methods/mock"
)

func TestGatewayMethodRegistration(t *testing.T) {
	g := NewGateway(
		WithMethod(mock.New(mock.Config{})),
	)

	methods := g.Methods()
	if len(methods) != 1 {
		t.Fatalf("Methods() = %v, want 1 method", methods)
	}

	m, ok := g.Method("mock")
	if !ok {
		t.Fatal("Method('mock') not found")
	}
	if m.Name() != "mock" {
		t.Errorf("Method.Name() = %q, want 'mock'", m.Name())
	}

	_, ok = g.Method("nonexistent")
	if ok {
		t.Error("Method('nonexistent') should not be found")
	}
}

func TestGatewayBuildPaymentRequired(t *testing.T) {
	g := NewGateway(
		WithMethod(mock.New(mock.Config{})),
	)

	resource := Resource{
		URL:         "https://api.example.com/data",
		Description: "Test resource",
	}
	price := Price{Amount: "5.99", Currency: "USD"}

	pr, err := g.BuildPaymentRequired(resource, price)
	if err != nil {
		t.Fatalf("BuildPaymentRequired error: %v", err)
	}

	if pr.ACPVersion != core.ACPVersion {
		t.Errorf("acpVersion = %d, want %d", pr.ACPVersion, core.ACPVersion)
	}
	if len(pr.Accepts) == 0 {
		t.Fatal("Accepts is empty")
	}
	if pr.Accepts[0].Method != "mock" {
		t.Errorf("Accepts[0].Method = %q, want 'mock'", pr.Accepts[0].Method)
	}
	if pr.Accepts[0].Amount != "5.99" {
		t.Errorf("Accepts[0].Amount = %q, want '5.99'", pr.Accepts[0].Amount)
	}
}

func TestGatewayVerifyAndSettle(t *testing.T) {
	mockMethod := mock.New(mock.Config{})
	g := NewGateway(WithMethod(mockMethod))

	ctx := context.Background()

	// Create a payment payload.
	option := core.PaymentOption{
		Intent:   core.IntentCharge,
		Method:   "mock",
		Currency: core.USD,
		Amount:   "5.99",
	}
	methodPayload, err := mockMethod.CreatePayload(ctx, option)
	if err != nil {
		t.Fatalf("CreatePayload error: %v", err)
	}

	payload := PaymentPayload{
		ACPVersion: core.ACPVersion,
		Resource:   Resource{URL: "https://example.com/data"},
		Accepted:   option,
		Payload:    methodPayload,
	}

	// Verify.
	vr, err := g.Verify(ctx, payload)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !vr.Valid {
		t.Fatalf("Verify returned invalid: %s", vr.Reason)
	}

	// Settle.
	sr, err := g.Settle(ctx, payload)
	if err != nil {
		t.Fatalf("Settle error: %v", err)
	}
	if !sr.Success {
		t.Fatal("Settle returned unsuccessful")
	}
	if sr.Method != "mock" {
		t.Errorf("Settle.Method = %q, want 'mock'", sr.Method)
	}
	if sr.Transaction == "" {
		t.Error("Settle.Transaction is empty")
	}

	// Verify the transaction was recorded.
	txns := mockMethod.Transactions()
	if len(txns) != 1 {
		t.Fatalf("Transactions() = %d, want 1", len(txns))
	}
}

func TestGatewayUnknownMethod(t *testing.T) {
	g := NewGateway(WithMethod(mock.New(mock.Config{})))

	payload := PaymentPayload{
		Accepted: PaymentOption{Method: "nonexistent"},
	}

	_, err := g.Verify(context.Background(), payload)
	if err == nil {
		t.Fatal("Verify should fail for unknown method")
	}
	if !core.IsPaymentError(err, core.ErrMethodUnavailable) {
		t.Errorf("expected ErrMethodUnavailable, got: %v", err)
	}
}
