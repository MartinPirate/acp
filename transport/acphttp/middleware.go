package acphttp

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
)

// Paywall wraps an http.Handler, requiring payment before access.
func Paywall(gateway *acp.Gateway, price acp.Price, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read payment from header.
		payload, err := ReadPaymentPayload(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, core.ErrInvalidPayload, "invalid ACP-Payment header: "+err.Error())
			return
		}

		// No payment present — return 402.
		if payload == nil {
			resource := core.Resource{
				URL:         r.URL.String(),
				Description: r.URL.Path,
				MimeType:    r.Header.Get("Accept"),
			}
			pr, err := gateway.BuildPaymentRequired(resource, price)
			if err != nil {
				log.Printf("acp: failed to build payment requirements: %v", err)
				writeError(w, http.StatusInternalServerError, core.ErrMethodUnavailable, "no payment methods available")
				return
			}
			if err := WritePaymentRequired(w, pr); err != nil {
				log.Printf("acp: failed to write 402 response: %v", err)
			}
			return
		}

		// Payment present — verify.
		verifyResp, err := gateway.Verify(r.Context(), *payload)
		if err != nil {
			log.Printf("acp: verification error: %v", err)
			writeError(w, http.StatusPaymentRequired, core.ErrVerificationFailed, "payment verification failed: "+err.Error())
			return
		}
		if !verifyResp.Valid {
			reason := "payment verification failed"
			if verifyResp.Reason != "" {
				reason = verifyResp.Reason
			}
			writeError(w, http.StatusPaymentRequired, core.ErrVerificationFailed, reason)
			return
		}

		// Verified — settle.
		settleResp, err := gateway.Settle(r.Context(), *payload)
		if err != nil {
			log.Printf("acp: settlement error: %v", err)
			writeError(w, http.StatusPaymentRequired, core.ErrSettlementFailed, "settlement failed: "+err.Error())
			return
		}
		if !settleResp.Success {
			writeError(w, http.StatusPaymentRequired, core.ErrSettlementFailed, "settlement unsuccessful")
			return
		}

		// Write settlement receipt header, then serve the resource.
		if err := WritePaymentResponse(w, settleResp); err != nil {
			log.Printf("acp: failed to write payment response header: %v", err)
		}
		next.ServeHTTP(w, r)
	})
}

// PaywallFunc is a convenience wrapper for http.HandlerFunc.
func PaywallFunc(gateway *acp.Gateway, price acp.Price, fn http.HandlerFunc) http.Handler {
	return Paywall(gateway, price, fn)
}

type errorResponse struct {
	Code    core.ErrorCode `json:"code"`
	Message string         `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code core.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}
