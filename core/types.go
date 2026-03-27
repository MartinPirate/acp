package core

import "encoding/json"

// ACPVersion is the current protocol version.
const ACPVersion = 1

// Resource identifies what is being paid for.
type Resource struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PaymentOption is one accepted payment method+intent+amount.
type PaymentOption struct {
	Intent      Intent          `json:"intent"`
	Method      string          `json:"method"`
	Currency    Currency        `json:"currency"`
	Amount      string          `json:"amount"`
	Description string          `json:"description,omitempty"`
	Extra       json.RawMessage `json:"extra,omitempty"`
}

// PaymentRequired is the 402 response payload (ACP-Payment-Required header).
type PaymentRequired struct {
	ACPVersion int             `json:"acpVersion"`
	Resource   Resource        `json:"resource"`
	Accepts    []PaymentOption `json:"accepts"`
	Extensions json.RawMessage `json:"extensions,omitempty"`
}

// PaymentPayload is what the agent sends back (ACP-Payment header).
type PaymentPayload struct {
	ACPVersion int             `json:"acpVersion"`
	Resource   Resource        `json:"resource"`
	Accepted   PaymentOption   `json:"accepted"`
	Payload    json.RawMessage `json:"payload"`
	Extensions json.RawMessage `json:"extensions,omitempty"`
}

// VerifyResponse is what the facilitator returns from /verify.
type VerifyResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	Payer  string `json:"payer,omitempty"`
}

// SettleResponse is the settlement receipt (ACP-Payment-Response header).
type SettleResponse struct {
	ACPVersion  int             `json:"acpVersion"`
	Success     bool            `json:"success"`
	Method      string          `json:"method"`
	Transaction string          `json:"transaction"`
	SettledAt   string          `json:"settledAt"`
	Receipt     json.RawMessage `json:"receipt,omitempty"`
	Extensions  json.RawMessage `json:"extensions,omitempty"`
}

// Price is the simple server-side price declaration.
type Price struct {
	Amount   string   `json:"amount"`
	Currency Currency `json:"currency"`
}
