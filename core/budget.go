package core

import (
	"math/big"
	"sync"
)

// Budget defines spending limits for a client.
type Budget struct {
	MaxPerRequest string   `json:"maxPerRequest,omitempty"`
	MaxPerSession string   `json:"maxPerSession,omitempty"`
	Currency      Currency `json:"currency"`
}

// BudgetEnforcer tracks spending against a budget.
type BudgetEnforcer struct {
	mu     sync.Mutex
	budget Budget
	spent  string
}

// NewBudgetEnforcer creates a new enforcer from a budget.
func NewBudgetEnforcer(b Budget) *BudgetEnforcer {
	return &BudgetEnforcer{
		budget: b,
		spent:  "0",
	}
}

// Check returns nil if the amount is within budget, or a PaymentError if not.
func (e *BudgetEnforcer) Check(amount string, currency Currency) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.budget.Currency != currency {
		return NewPaymentError(ErrCurrencyMismatch, "budget currency is "+string(e.budget.Currency)+", payment is "+string(currency))
	}

	if e.budget.MaxPerRequest != "" {
		cmp, err := CompareAmounts(amount, e.budget.MaxPerRequest)
		if err != nil {
			return NewPaymentError(ErrInvalidPayload, err.Error())
		}
		if cmp > 0 {
			return NewPaymentError(ErrBudgetExceeded, "amount "+amount+" exceeds per-request limit "+e.budget.MaxPerRequest)
		}
	}

	if e.budget.MaxPerSession != "" {
		total, err := addAmounts(e.spent, amount)
		if err != nil {
			return NewPaymentError(ErrInvalidPayload, err.Error())
		}
		cmp, err := CompareAmounts(total, e.budget.MaxPerSession)
		if err != nil {
			return NewPaymentError(ErrInvalidPayload, err.Error())
		}
		if cmp > 0 {
			return NewPaymentError(ErrBudgetExceeded, "session total "+total+" would exceed limit "+e.budget.MaxPerSession)
		}
	}

	return nil
}

// Record adds an amount to cumulative spend. Called after successful settlement.
func (e *BudgetEnforcer) Record(amount string, currency Currency) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.budget.Currency != currency {
		return NewPaymentError(ErrCurrencyMismatch, "budget currency is "+string(e.budget.Currency)+", payment is "+string(currency))
	}

	total, err := addAmounts(e.spent, amount)
	if err != nil {
		return NewPaymentError(ErrInvalidPayload, err.Error())
	}
	e.spent = total
	return nil
}

// Spent returns the cumulative amount spent in this session.
func (e *BudgetEnforcer) Spent() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.spent
}

func addAmounts(a, b string) (string, error) {
	ra := new(big.Rat)
	if _, ok := ra.SetString(a); !ok {
		return "", NewPaymentError(ErrInvalidPayload, "invalid amount: "+a)
	}
	rb := new(big.Rat)
	if _, ok := rb.SetString(b); !ok {
		return "", NewPaymentError(ErrInvalidPayload, "invalid amount: "+b)
	}
	result := new(big.Rat).Add(ra, rb)
	return result.FloatString(10), nil
}
