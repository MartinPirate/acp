package acphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
)

// MethodSelector chooses which payment option to use from a 402 response.
type MethodSelector func(options []core.PaymentOption) (*core.PaymentOption, error)

// Client is an HTTP client that automatically handles 402 responses.
type Client struct {
	httpClient *http.Client
	gateway    *acp.Gateway
	budget     *core.BudgetEnforcer
	selector   MethodSelector
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithBudget sets spending limits on the client.
func WithBudget(b core.Budget) ClientOption {
	return func(cl *Client) {
		cl.budget = core.NewBudgetEnforcer(b)
	}
}

// WithSelector sets a custom method selection strategy.
func WithSelector(s MethodSelector) ClientOption {
	return func(cl *Client) {
		cl.selector = s
	}
}

// NewClient creates a payment-aware HTTP client.
func NewClient(gateway *acp.Gateway, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		gateway:    gateway,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.selector == nil {
		c.selector = defaultSelector(gateway)
	}
	return c
}

// Get performs a GET request, automatically paying if a 402 is returned.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Do performs an arbitrary request with automatic 402 handling.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Buffer the body in case we need to retry.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("acp: failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Make the initial request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Not a 402 — return as-is.
	if resp.StatusCode != http.StatusPaymentRequired {
		return resp, nil
	}

	// Decode the 402 response.
	pr, err := decodePaymentRequired(resp)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to decode 402 response: %w", err)
	}

	// Select a payment method.
	selected, err := c.selector(pr.Accepts)
	if err != nil {
		return nil, fmt.Errorf("acp: method selection failed: %w", err)
	}

	// Check budget.
	if c.budget != nil {
		if err := c.budget.Check(selected.Amount, selected.Currency); err != nil {
			return nil, err
		}
	}

	// Create method-specific payload.
	method, ok := c.gateway.Method(selected.Method)
	if !ok {
		return nil, core.NewPaymentError(core.ErrMethodUnavailable,
			fmt.Sprintf("method %q not available in client gateway", selected.Method))
	}

	methodPayload, err := method.CreatePayload(req.Context(), *selected)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to create payment payload: %w", err)
	}

	paymentPayload := core.PaymentPayload{
		ACPVersion: core.ACPVersion,
		Resource:   pr.Resource,
		Accepted:   *selected,
		Payload:    methodPayload,
	}

	encoded, err := EncodeHeader(paymentPayload)
	if err != nil {
		return nil, fmt.Errorf("acp: failed to encode payment header: %w", err)
	}

	// Build the retry request.
	retryReq := req.Clone(context.WithoutCancel(req.Context()))
	if bodyBytes != nil {
		retryReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	retryReq.Header.Set(HeaderPayment, encoded)

	// Make the retry request.
	resp2, err := c.httpClient.Do(retryReq)
	if err != nil {
		return nil, err
	}

	// Record spend in budget.
	if c.budget != nil && resp2.StatusCode == http.StatusOK {
		c.budget.Record(selected.Amount, selected.Currency)
	}

	return resp2, nil
}

func decodePaymentRequired(resp *http.Response) (*core.PaymentRequired, error) {
	// Try header first.
	header := resp.Header.Get(HeaderPaymentRequired)
	if header != "" {
		var pr core.PaymentRequired
		if err := DecodeHeader(header, &pr); err == nil {
			return &pr, nil
		}
	}

	// Fall back to body.
	defer resp.Body.Close()
	var pr core.PaymentRequired
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

func defaultSelector(gateway *acp.Gateway) MethodSelector {
	return func(options []core.PaymentOption) (*core.PaymentOption, error) {
		for i := range options {
			if _, ok := gateway.Method(options[i].Method); ok {
				return &options[i], nil
			}
		}
		return nil, core.NewPaymentError(core.ErrMethodUnavailable, "no compatible payment method found")
	}
}
