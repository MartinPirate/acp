package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BaseConfig provides common configuration shared by all payment methods.
type BaseConfig struct {
	HTTPClient *http.Client
}

// GetHTTPClient returns the configured client or http.DefaultClient.
func (c BaseConfig) GetHTTPClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// UnmarshalMethodPayload unmarshals a raw JSON payload into dst and returns a
// typed PaymentError on failure.
func UnmarshalMethodPayload(raw json.RawMessage, dst any, methodName string) error {
	if err := json.Unmarshal(raw, dst); err != nil {
		return NewPaymentError(ErrInvalidPayload, "invalid "+methodName+" payload: "+err.Error())
	}
	return nil
}

// BuildSettleResponse creates a standardized settlement response.
func BuildSettleResponse(method, txnID string, receipt any) (*SettleResponse, error) {
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return nil, NewPaymentError(ErrSettlementFailed, "failed to marshal receipt: "+err.Error())
	}
	return &SettleResponse{
		ACPVersion:  ACPVersion,
		Success:     true,
		Method:      method,
		Transaction: txnID,
		SettledAt:   time.Now().Format(time.RFC3339),
		Receipt:     receiptJSON,
	}, nil
}

// GenerateTxnID creates a unique transaction ID with a method prefix.
func GenerateTxnID(prefix string) string {
	return fmt.Sprintf("%s_txn_%d", prefix, time.Now().UnixNano())
}

// ValidateBuildOption checks intent and currency support, returning appropriate errors.
func ValidateBuildOption(methodName string, intent Intent, currency Currency, supportedIntents []Intent, supportedCurrencies []Currency) error {
	if !ContainsIntent(supportedIntents, intent) {
		return NewPaymentError(ErrUnsupportedIntent, fmt.Sprintf("%s does not support intent %q", methodName, intent))
	}
	if !ContainsCurrency(supportedCurrencies, currency) {
		return NewPaymentError(ErrCurrencyMismatch, fmt.Sprintf("%s does not support currency %q", methodName, currency))
	}
	return nil
}
