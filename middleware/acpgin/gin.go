// Package acpgin provides Gin-compatible middleware for the Agentic Commerce Protocol.
package acpgin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/core"
	"github.com/paideia-ai/acp/transport/acphttp"
)

// Paywall returns a Gin middleware handler that requires payment before
// allowing access to downstream handlers.
func Paywall(gateway *acp.Gateway, price acp.Price) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to read payment from header.
		payload, err := acphttp.ReadPaymentPayload(c.Request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    string(core.ErrInvalidPayload),
				"message": "invalid ACP-Payment header: " + err.Error(),
			})
			c.Abort()
			return
		}

		// No payment present -- return 402.
		if payload == nil {
			resource := core.Resource{
				URL:         c.Request.URL.String(),
				Description: c.Request.URL.Path,
				MimeType:    c.Request.Header.Get("Accept"),
			}
			pr, err := gateway.BuildPaymentRequired(resource, price)
			if err != nil {
				log.Printf("acp: failed to build payment requirements: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    string(core.ErrMethodUnavailable),
					"message": "no payment methods available",
				})
				c.Abort()
				return
			}
			encoded, err := acphttp.EncodeHeader(pr)
			if err != nil {
				log.Printf("acp: failed to encode payment required header: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    string(core.ErrMethodUnavailable),
					"message": "failed to encode payment requirements",
				})
				c.Abort()
				return
			}
			c.Header(acphttp.HeaderPaymentRequired, encoded)
			c.JSON(http.StatusPaymentRequired, pr)
			c.Abort()
			return
		}

		// Payment present -- verify.
		verifyResp, err := gateway.Verify(c.Request.Context(), *payload)
		if err != nil {
			log.Printf("acp: verification error: %v", err)
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    string(core.ErrVerificationFailed),
				"message": "payment verification failed: " + err.Error(),
			})
			c.Abort()
			return
		}
		if !verifyResp.Valid {
			reason := "payment verification failed"
			if verifyResp.Reason != "" {
				reason = verifyResp.Reason
			}
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    string(core.ErrVerificationFailed),
				"message": reason,
			})
			c.Abort()
			return
		}

		// Verified -- settle.
		settleResp, err := gateway.Settle(c.Request.Context(), *payload)
		if err != nil {
			log.Printf("acp: settlement error: %v", err)
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    string(core.ErrSettlementFailed),
				"message": "settlement failed: " + err.Error(),
			})
			c.Abort()
			return
		}
		if !settleResp.Success {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    string(core.ErrSettlementFailed),
				"message": "settlement unsuccessful",
			})
			c.Abort()
			return
		}

		// Write settlement receipt header, then proceed.
		encoded, err := acphttp.EncodeHeader(settleResp)
		if err != nil {
			log.Printf("acp: failed to encode payment response header: %v", err)
		} else {
			c.Header(acphttp.HeaderPaymentResponse, encoded)
		}
		c.Next()
	}
}
