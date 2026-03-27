// Package auth provides OAuth 2.0 agent token issuance and validation
// based on the IETF draft-oauth-ai-agents-on-behalf-of-user pattern.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Permission defines what a token holder can access.
type Permission struct {
	Resource  string   `json:"resource"`  // URL pattern (glob)
	Methods   []string `json:"methods"`   // allowed payment methods
	MaxAmount string   `json:"maxAmount"` // max amount per request
	Currency  string   `json:"currency"`
}

// AgentToken represents a validated agent token.
type AgentToken struct {
	AgentID     string
	UserID      string
	ClientID    string
	Scopes      []string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Permissions []Permission
}

// TokenValidator validates agent tokens.
type TokenValidator interface {
	Validate(token string) (*AgentToken, error)
}

// --- JWT Validator ---

// JWTConfig configures the JWT validator.
type JWTConfig struct {
	SigningKey []byte
	Issuer    string
	Audience  string
}

// agentClaims are the JWT claims for an agent token.
type agentClaims struct {
	jwt.RegisteredClaims
	AgentID     string       `json:"agent_id"`
	UserID      string       `json:"user_id"`
	ClientID    string       `json:"client_id"`
	Scopes      []string     `json:"scopes"`
	Permissions []Permission `json:"permissions"`
}

// JWTValidator validates JWT agent tokens.
type JWTValidator struct {
	config JWTConfig
}

// NewJWTValidator creates a new JWT validator.
func NewJWTValidator(cfg JWTConfig) *JWTValidator {
	return &JWTValidator{config: cfg}
}

// Validate parses and validates a JWT agent token.
func (v *JWTValidator) Validate(tokenStr string) (*AgentToken, error) {
	claims := &agentClaims{}

	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}),
	}
	if v.config.Issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(v.config.Issuer))
	}
	if v.config.Audience != "" {
		parserOpts = append(parserOpts, jwt.WithAudience(v.config.Audience))
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.config.SigningKey, nil
	}, parserOpts...)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token validation failed")
	}

	return &AgentToken{
		AgentID:     claims.AgentID,
		UserID:      claims.UserID,
		ClientID:    claims.ClientID,
		Scopes:      claims.Scopes,
		IssuedAt:    claims.IssuedAt.Time,
		ExpiresAt:   claims.ExpiresAt.Time,
		Permissions: claims.Permissions,
	}, nil
}

// --- Token Issuer ---

// TokenIssuer issues JWT agent tokens.
type TokenIssuer struct {
	signingKey []byte
	issuer     string
	audience   string
}

// NewTokenIssuer creates a new token issuer.
func NewTokenIssuer(signingKey []byte, issuer, audience string) *TokenIssuer {
	return &TokenIssuer{
		signingKey: signingKey,
		issuer:     issuer,
		audience:   audience,
	}
}

// Issue creates a signed JWT agent token.
func (ti *TokenIssuer) Issue(agentID, userID string, perms []Permission, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := agentClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ti.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		AgentID:     agentID,
		UserID:      userID,
		Permissions: perms,
	}
	if ti.audience != "" {
		claims.Audience = jwt.ClaimStrings{ti.audience}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(ti.signingKey)
}

// --- HTTP Middleware ---

type contextKey string

const agentTokenKey contextKey = "acp-agent-token"

// AuthMiddleware returns HTTP middleware that extracts and validates agent tokens
// from the Authorization header (Bearer scheme).
func AuthMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			agentToken, err := validator.Validate(parts[1])
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), agentTokenKey, agentToken)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContextAgentToken extracts the AgentToken from the request context.
// Returns nil if no token is present.
func ContextAgentToken(ctx context.Context) *AgentToken {
	tok, _ := ctx.Value(agentTokenKey).(*AgentToken)
	return tok
}
