// Package acpchi provides Chi-compatible middleware for the Agentic Commerce Protocol.
package acpchi

import (
	"net/http"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/transport/acphttp"
)

// Paywall returns a Chi-compatible middleware that requires payment before
// allowing access to downstream handlers. Chi uses stdlib http.Handler, so
// this wraps the existing acphttp.Paywall with the Chi middleware signature.
func Paywall(gateway *acp.Gateway, price acp.Price) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return acphttp.Paywall(gateway, price, next)
	}
}
