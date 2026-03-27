package acphttp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/paideia-ai/acp/core"
)

// Header names used by the ACP protocol over HTTP.
const (
	HeaderPaymentRequired = "ACP-Payment-Required"
	HeaderPayment         = "ACP-Payment"
	HeaderPaymentResponse = "ACP-Payment-Response"
	HeaderIdempotencyKey  = "Idempotency-Key"
)

// EncodeHeader marshals v to JSON and base64-encodes it.
func EncodeHeader(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeHeader base64-decodes and unmarshals a header value into dst.
func DecodeHeader(header string, dst any) error {
	data, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// WritePaymentRequired writes a 402 response with payment requirements.
func WritePaymentRequired(w http.ResponseWriter, pr *core.PaymentRequired) error {
	encoded, err := EncodeHeader(pr)
	if err != nil {
		return err
	}
	w.Header().Set(HeaderPaymentRequired, encoded)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	return json.NewEncoder(w).Encode(pr)
}

// ReadPaymentPayload extracts and decodes ACP-Payment from a request.
// Returns nil, nil if no payment header is present.
func ReadPaymentPayload(r *http.Request) (*core.PaymentPayload, error) {
	header := r.Header.Get(HeaderPayment)
	if header == "" {
		return nil, nil
	}
	var payload core.PaymentPayload
	if err := DecodeHeader(header, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// WritePaymentResponse writes the settlement receipt header.
func WritePaymentResponse(w http.ResponseWriter, sr *core.SettleResponse) error {
	encoded, err := EncodeHeader(sr)
	if err != nil {
		return err
	}
	w.Header().Set(HeaderPaymentResponse, encoded)
	return nil
}
