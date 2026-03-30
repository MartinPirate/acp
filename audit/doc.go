// Package audit provides payment audit trail logging and receipt storage
// for the Agentic Commerce Protocol.
//
// Every payment verify and settle operation can be recorded as an [Entry]
// with structured fields for the agent, resource, method, amount, status,
// and provider transaction ID.
//
// # Key Types
//
//   - [Entry] -- a single audit log record with status tracking
//     ([StatusPending], [StatusVerified], [StatusSettled], [StatusFailed]).
//   - [Logger] -- interface for recording and querying audit entries.
//   - [MemoryLogger] -- in-memory [Logger] implementation.
//   - [AuditedGateway] -- decorator that wraps any [GatewayInterface] and
//     logs all verify/settle calls automatically.
//   - [Filter] -- query criteria for retrieving entries by agent, method,
//     status, resource pattern, or time range.
//
// # Usage
//
// Wrap a gateway with audit logging:
//
//	logger := audit.NewMemoryLogger()
//	audited := audit.NewAuditedGateway(gateway, logger)
//
//	// Use audited in place of gateway -- all payments are logged.
//	resp, err := audited.Verify(ctx, payload)
//
//	// Query the audit trail.
//	entries, _ := logger.Query(audit.Filter{
//	    AgentID: "agent-42",
//	    Status:  audit.StatusSettled,
//	    Limit:   50,
//	})
package audit
