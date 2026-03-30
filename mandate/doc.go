// Package mandate provides pre-authorized spending authority for AI agents.
//
// A [Mandate] defines what an agent is allowed to spend, on which payment
// methods, for which intents, and against which resource URL scopes. The
// [Enforcer] validates payments against mandates and tracks cumulative spend
// using arbitrary-precision arithmetic.
//
// # Key Types
//
//   - [Mandate] -- the authorization specification (amount, currency, methods,
//     intents, scopes, and expiry).
//   - [Enforcer] -- validates a payment against a mandate and tracks spend.
//   - [Store] -- persistence interface for mandates (see [MemoryStore]).
//
// # Usage
//
// Create a mandate and enforce it:
//
//	m := mandate.NewMandate(
//	    mandate.WithID("m-001"),
//	    mandate.WithAgentID("agent-42"),
//	    mandate.WithMaxAmount("100.00"),
//	    mandate.WithCurrency(core.USD),
//	    mandate.WithMaxPerRequest("10.00"),
//	    mandate.WithAllowedMethods("card", "mock"),
//	    mandate.WithAllowedIntents(core.IntentCharge),
//	    mandate.WithScope("/api/*"),
//	    mandate.WithExpiry(time.Now().Add(24*time.Hour)),
//	)
//
//	enforcer := mandate.NewEnforcer()
//	if err := enforcer.Check(m, payload, resource); err != nil {
//	    // payment not allowed
//	}
//	enforcer.Record(m, payload.Accepted.Amount)
package mandate
