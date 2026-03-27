package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// IdempotencyKey generates a deterministic idempotency key from payment parameters.
func IdempotencyKey(agentID, resource, method, amount string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%s", agentID, resource, method, amount)))
	return hex.EncodeToString(h[:16])
}
