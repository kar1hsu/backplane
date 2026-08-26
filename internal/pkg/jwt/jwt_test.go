package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/kar1hsu/backplane/internal/app"
)

func TestGenerateAndParseToken(t *testing.T) {
	original := app.Cfg.JWT
	t.Cleanup(func() { app.Cfg.JWT = original })
	app.Cfg.JWT = app.JWTConfig{
		Secret: "unit-test-secret",
		Expire: 120,
		Issuer: "backplane-test",
	}

	token, err := GenerateToken(42, "alice", []string{"editor", "auditor"}, 3)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.Username != "alice" || claims.TokenVersion != 3 {
		t.Fatalf("claims = %+v", claims)
	}
	if strings.Join(claims.RoleCodes, ",") != "editor,auditor" {
		t.Fatalf("role codes = %v", claims.RoleCodes)
	}
	if claims.Issuer != "backplane-test" {
		t.Fatalf("issuer = %q", claims.Issuer)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Fatalf("expires_at = %v", claims.ExpiresAt)
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	original := app.Cfg.JWT
	t.Cleanup(func() { app.Cfg.JWT = original })
	app.Cfg.JWT = app.JWTConfig{Secret: "unit-test-secret", Expire: 60, Issuer: "backplane-test"}

	token, err := GenerateToken(1, "admin", []string{"admin"}, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	tampered := token[:len(token)-1] + "x"
	if _, err := ParseToken(tampered); err == nil {
		t.Fatal("ParseToken() accepted a tampered token")
	}

	app.Cfg.JWT.Expire = -1
	expired, err := GenerateToken(1, "admin", []string{"admin"}, 1)
	if err != nil {
		t.Fatalf("GenerateToken() expired token error = %v", err)
	}
	if _, err := ParseToken(expired); err == nil {
		t.Fatal("ParseToken() accepted an expired token")
	}
}
