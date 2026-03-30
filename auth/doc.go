// Package auth provides OAuth 2.0 agent token issuance and validation
// based on the IETF draft-oauth-ai-agents-on-behalf-of-user pattern.
//
// It supports issuing JWT tokens that encode an agent's identity, the
// delegating user, scopes, and fine-grained [Permission] rules (resource
// patterns, allowed methods, and per-request amount caps).
//
// # Key Types
//
//   - [AgentToken] -- a validated token carrying agent/user identity and
//     permissions.
//   - [TokenValidator] -- interface for validating bearer tokens.
//   - [JWTValidator] -- HMAC-signed JWT validator with issuer/audience checks.
//   - [TokenIssuer] -- issues signed JWT agent tokens with a configurable TTL.
//   - [Permission] -- resource-scoped spending constraint embedded in the token.
//
// # Usage
//
// Issue a token for an agent acting on behalf of a user:
//
//	issuer := auth.NewTokenIssuer([]byte("secret"), "acp-server", "acp-api")
//	token, err := issuer.Issue("agent-42", "user-7", []auth.Permission{
//	    {Resource: "/api/*", Methods: []string{"card"}, MaxAmount: "50.00", Currency: "USD"},
//	}, 1*time.Hour)
//
// Protect routes with the auth middleware:
//
//	validator := auth.NewJWTValidator(auth.JWTConfig{
//	    SigningKey: []byte("secret"),
//	    Issuer:    "acp-server",
//	})
//	mux.Handle("/api/", auth.AuthMiddleware(validator)(handler))
//
// Retrieve the token from the request context:
//
//	tok := auth.ContextAgentToken(r.Context())
package auth
