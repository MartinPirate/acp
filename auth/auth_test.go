package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testSigningKey = "test-secret-key-for-signing-jwts"

func newTestIssuerAndValidator() (*TokenIssuer, *JWTValidator) {
	key := []byte(testSigningKey)
	issuer := NewTokenIssuer(key, "test-issuer", "test-audience")
	validator := NewJWTValidator(JWTConfig{
		SigningKey: key,
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	})
	return issuer, validator
}

func TestIssueAndValidate(t *testing.T) {
	issuer, validator := newTestIssuerAndValidator()

	perms := []Permission{
		{
			Resource:  "/api/*",
			Methods:   []string{"card", "mock"},
			MaxAmount: "100.00",
			Currency:  "USD",
		},
	}

	tokenStr, err := issuer.Issue("agent-1", "user-1", perms, time.Hour)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	token, err := validator.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if token.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", token.AgentID, "agent-1")
	}
	if token.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", token.UserID, "user-1")
	}
	if len(token.Permissions) != 1 {
		t.Fatalf("Permissions len = %d, want 1", len(token.Permissions))
	}
	if token.Permissions[0].Resource != "/api/*" {
		t.Errorf("Permission resource = %q, want %q", token.Permissions[0].Resource, "/api/*")
	}
	if token.Permissions[0].MaxAmount != "100.00" {
		t.Errorf("Permission maxAmount = %q, want %q", token.Permissions[0].MaxAmount, "100.00")
	}
	if token.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
	if token.IssuedAt.IsZero() {
		t.Error("IssuedAt should not be zero")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	issuer, validator := newTestIssuerAndValidator()

	tokenStr, err := issuer.Issue("agent-1", "user-1", nil, -time.Hour)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	_, err = validator.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateWrongKey(t *testing.T) {
	issuer := NewTokenIssuer([]byte("key-a"), "test-issuer", "test-audience")
	validator := NewJWTValidator(JWTConfig{
		SigningKey: []byte("key-b"),
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	})

	tokenStr, err := issuer.Issue("agent-1", "user-1", nil, time.Hour)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	_, err = validator.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestValidateWrongIssuer(t *testing.T) {
	issuer := NewTokenIssuer([]byte(testSigningKey), "issuer-a", "test-audience")
	validator := NewJWTValidator(JWTConfig{
		SigningKey: []byte(testSigningKey),
		Issuer:    "issuer-b",
		Audience:  "test-audience",
	})

	tokenStr, err := issuer.Issue("agent-1", "user-1", nil, time.Hour)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	_, err = validator.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestValidateGarbage(t *testing.T) {
	validator := NewJWTValidator(JWTConfig{SigningKey: []byte(testSigningKey)})
	_, err := validator.Validate("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestAuthMiddleware(t *testing.T) {
	issuer, validator := newTestIssuerAndValidator()

	tokenStr, err := issuer.Issue("agent-1", "user-1", []Permission{
		{Resource: "/api/*", Methods: []string{"mock"}},
	}, time.Hour)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	var capturedToken *AgentToken
	handler := AuthMiddleware(validator)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedToken = ContextAgentToken(r.Context())
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Valid token.
	req := httptest.NewRequest("GET", "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if capturedToken == nil {
		t.Fatal("expected token in context")
	}
	if capturedToken.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", capturedToken.AgentID, "agent-1")
	}
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	_, validator := newTestIssuerAndValidator()
	handler := AuthMiddleware(validator)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	_, validator := newTestIssuerAndValidator()
	handler := AuthMiddleware(validator)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	_, validator := newTestIssuerAndValidator()
	handler := AuthMiddleware(validator)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestContextAgentTokenNil(t *testing.T) {
	tok := ContextAgentToken(context.Background())
	if tok != nil {
		t.Error("expected nil token from empty context")
	}
}
