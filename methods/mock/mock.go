package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

// Config for the mock payment method.
type Config struct {
	// ShouldFail makes Verify/Settle return errors (for testing error paths).
	ShouldFail bool
	// SettleDelay adds artificial latency to Settle.
	SettleDelay time.Duration
}

// MockMethod implements core.Method for testing.
type MockMethod struct {
	config       Config
	mu           sync.Mutex
	transactions []Transaction
}

// Transaction records a mock settlement.
type Transaction struct {
	ID        string          `json:"id"`
	Amount    string          `json:"amount"`
	Currency  core.Currency   `json:"currency"`
	SettledAt time.Time       `json:"settledAt"`
	Payload   json.RawMessage `json:"payload"`
}

type mockPayload struct {
	Token string `json:"token"`
}

// New creates a new mock payment method.
func New(cfg Config) *MockMethod {
	return &MockMethod{
		config: cfg,
	}
}

func (m *MockMethod) Name() string { return "mock" }

func (m *MockMethod) SupportedIntents() []core.Intent {
	return []core.Intent{core.IntentCharge, core.IntentAuthorize}
}

func (m *MockMethod) SupportedCurrencies() []core.Currency {
	// Mock supports all currencies.
	return []core.Currency{
		core.USD, core.EUR, core.GBP, core.JPY, core.INR,
		core.BRL, core.KES, core.NGN, core.ZAR, core.CNY,
	}
}

func (m *MockMethod) BuildOption(intent core.Intent, price core.Price) (core.PaymentOption, error) {
	if err := core.ValidateBuildOption("mock", intent, price.Currency, m.SupportedIntents(), m.SupportedCurrencies()); err != nil {
		return core.PaymentOption{}, err
	}
	return core.PaymentOption{
		Intent:      intent,
		Method:      "mock",
		Currency:    price.Currency,
		Amount:      price.Amount,
		Description: "Mock payment (testing only)",
	}, nil
}

func (m *MockMethod) CreatePayload(_ context.Context, _ core.PaymentOption) (json.RawMessage, error) {
	p := mockPayload{Token: fmt.Sprintf("mock_tok_%d", time.Now().UnixNano())}
	return json.Marshal(p)
}

func (m *MockMethod) Verify(_ context.Context, payload core.PaymentPayload, _ core.PaymentOption) (*core.VerifyResponse, error) {
	if m.config.ShouldFail {
		return &core.VerifyResponse{Valid: false, Reason: "mock configured to fail"}, nil
	}

	var p mockPayload
	if err := core.UnmarshalMethodPayload(payload.Payload, &p, "mock"); err != nil {
		return nil, err
	}
	if p.Token == "" {
		return &core.VerifyResponse{Valid: false, Reason: "empty token"}, nil
	}

	return &core.VerifyResponse{Valid: true, Payer: "mock-payer"}, nil
}

func (m *MockMethod) Settle(ctx context.Context, payload core.PaymentPayload, option core.PaymentOption) (*core.SettleResponse, error) {
	if m.config.ShouldFail {
		return nil, core.NewPaymentError(core.ErrSettlementFailed, "mock configured to fail")
	}

	if m.config.SettleDelay > 0 {
		select {
		case <-time.After(m.config.SettleDelay):
		case <-ctx.Done():
			return nil, core.NewPaymentError(core.ErrTimeout, "settlement timed out")
		}
	}

	txn := Transaction{
		ID:        core.GenerateTxnID("mock"),
		Amount:    option.Amount,
		Currency:  option.Currency,
		SettledAt: time.Now(),
		Payload:   payload.Payload,
	}

	m.mu.Lock()
	m.transactions = append(m.transactions, txn)
	m.mu.Unlock()

	return core.BuildSettleResponse("mock", txn.ID, txn)
}

// Transactions returns all recorded transactions (for test assertions).
func (m *MockMethod) Transactions() []Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Transaction, len(m.transactions))
	copy(out, m.transactions)
	return out
}

// Reset clears all recorded transactions.
func (m *MockMethod) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = nil
}
