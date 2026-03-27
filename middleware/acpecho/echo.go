// Package acpecho provides Echo-compatible middleware for the Agentic Commerce Protocol.
package acpecho

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/transport/acphttp"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Paywall returns an Echo middleware that requires payment before
// allowing access to downstream handlers.
func Paywall(gateway *acp.Gateway, price acp.Price) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()

			// Try to read payment from header.
			payload, err := acphttp.ReadPaymentPayload(r)
			if err != nil {
				return c.JSON(http.StatusBadRequest, errorResponse{
					Code:    string(core.ErrInvalidPayload),
					Message: "invalid ACP-Payment header: " + err.Error(),
				})
			}

			// No payment present -- return 402.
			if payload == nil {
				resource := core.Resource{
					URL:         r.URL.String(),
					Description: r.URL.Path,
					MimeType:    r.Header.Get("Accept"),
				}
				pr, err := gateway.BuildPaymentRequired(resource, price)
				if err != nil {
					log.Printf("acp: failed to build payment requirements: %v", err)
					return c.JSON(http.StatusInternalServerError, errorResponse{
						Code:    string(core.ErrMethodUnavailable),
						Message: "no payment methods available",
					})
				}
				encoded, err := acphttp.EncodeHeader(pr)
				if err != nil {
					log.Printf("acp: failed to encode payment required header: %v", err)
					return c.JSON(http.StatusInternalServerError, errorResponse{
						Code:    string(core.ErrMethodUnavailable),
						Message: "failed to encode payment requirements",
					})
				}
				c.Response().Header().Set(acphttp.HeaderPaymentRequired, encoded)
				return c.JSON(http.StatusPaymentRequired, pr)
			}

			// Payment present -- verify.
			verifyResp, err := gateway.Verify(r.Context(), *payload)
			if err != nil {
				log.Printf("acp: verification error: %v", err)
				return c.JSON(http.StatusPaymentRequired, errorResponse{
					Code:    string(core.ErrVerificationFailed),
					Message: "payment verification failed: " + err.Error(),
				})
			}
			if !verifyResp.Valid {
				reason := "payment verification failed"
				if verifyResp.Reason != "" {
					reason = verifyResp.Reason
				}
				return c.JSON(http.StatusPaymentRequired, errorResponse{
					Code:    string(core.ErrVerificationFailed),
					Message: reason,
				})
			}

			// Verified -- settle.
			settleResp, err := gateway.Settle(r.Context(), *payload)
			if err != nil {
				log.Printf("acp: settlement error: %v", err)
				return c.JSON(http.StatusPaymentRequired, errorResponse{
					Code:    string(core.ErrSettlementFailed),
					Message: "settlement failed: " + err.Error(),
				})
			}
			if !settleResp.Success {
				return c.JSON(http.StatusPaymentRequired, errorResponse{
					Code:    string(core.ErrSettlementFailed),
					Message: "settlement unsuccessful",
				})
			}

			// Write settlement receipt header, then proceed.
			encoded, err := acphttp.EncodeHeader(settleResp)
			if err != nil {
				log.Printf("acp: failed to encode payment response header: %v", err)
			} else {
				c.Response().Header().Set(acphttp.HeaderPaymentResponse, encoded)
			}
			return next(c)
		}
	}
}
