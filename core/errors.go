package core

import "errors"

// ErrorCode is a machine-readable payment error code.
type ErrorCode string

const (
	ErrInsufficientFunds  ErrorCode = "insufficient_funds"
	ErrMethodUnavailable  ErrorCode = "method_unavailable"
	ErrMandateExceeded    ErrorCode = "mandate_exceeded"
	ErrMandateExpired     ErrorCode = "mandate_expired"
	ErrCurrencyMismatch   ErrorCode = "currency_mismatch"
	ErrAmountTooHigh      ErrorCode = "amount_too_high"
	ErrVerificationFailed ErrorCode = "verification_failed"
	ErrSettlementFailed   ErrorCode = "settlement_failed"
	ErrTimeout            ErrorCode = "timeout"
	ErrInvalidPayload     ErrorCode = "invalid_payload"
	ErrUnsupportedIntent  ErrorCode = "unsupported_intent"
	ErrBudgetExceeded     ErrorCode = "budget_exceeded"
)

// PaymentError is a structured error with code, message, and optional method.
type PaymentError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Method  string    `json:"method,omitempty"`
}

func (e *PaymentError) Error() string {
	if e.Method != "" {
		return string(e.Code) + " [" + e.Method + "]: " + e.Message
	}
	return string(e.Code) + ": " + e.Message
}

// NewPaymentError creates a new PaymentError.
func NewPaymentError(code ErrorCode, message string) *PaymentError {
	return &PaymentError{Code: code, Message: message}
}

// NewMethodError creates a PaymentError scoped to a specific method.
func NewMethodError(code ErrorCode, method, message string) *PaymentError {
	return &PaymentError{Code: code, Method: method, Message: message}
}

// IsPaymentError checks if err is a *PaymentError and optionally matches one of the given codes.
func IsPaymentError(err error, codes ...ErrorCode) bool {
	var pe *PaymentError
	if !errors.As(err, &pe) {
		return false
	}
	if len(codes) == 0 {
		return true
	}
	for _, c := range codes {
		if pe.Code == c {
			return true
		}
	}
	return false
}
