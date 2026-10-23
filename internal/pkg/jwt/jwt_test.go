package jwt

import (
	"testing"
	"time"

	"github.com/kar1hsu/backplane/internal/app"
)

func TestGenerateAndParseToken(t *testing.T) {
	setTestJWTConfig(t, 120)

	token, err := GenerateToken(42, "alice", 3)
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
	if claims.Issuer != "backplane-test" {
		t.Fatalf("issuer = %q", claims.Issuer)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Fatalf("expires_at = %v", claims.ExpiresAt)
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	setTestJWTConfig(t, 60)

	token, err := GenerateToken(1, "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	tampered := token[:len(token)-1] + "x"
	if _, err := ParseToken(tampered); err == nil {
		t.Fatal("ParseToken() accepted a tampered token")
	}

	app.Cfg.JWT.Expire = -1
	expired, err := GenerateToken(1, "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken() expired token error = %v", err)
	}
	if _, err := ParseToken(expired); err == nil {
		t.Fatal("ParseToken() accepted an expired token")
	}
}

// An HS256 signature is 32 bytes = 43 base64url chars, so the last char holds
// 2 unused bits. Lenient decoding accepts 3 other spellings of the same
// signature, which would bypass the logout blacklist keyed by token text.
func TestParseTokenRejectsNonCanonicalSignature(t *testing.T) {
	setTestJWTConfig(t, 60)

	token, err := GenerateToken(1, "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for i := 0; i < len(alphabet); i++ {
		if alphabet[i] == token[len(token)-1] {
			continue
		}
		variant := token[:len(token)-1] + string(alphabet[i])
		if _, err := ParseToken(variant); err == nil {
			t.Fatalf("ParseToken() accepted non-canonical variant ending in %q", alphabet[i])
		}
	}
}

func TestParseTokenRejectsWrongIssuer(t *testing.T) {
	setTestJWTConfig(t, 60)

	app.Cfg.JWT.Issuer = "another-service"
	token, err := GenerateToken(1, "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	app.Cfg.JWT.Issuer = "backplane-test"
	if _, err := ParseToken(token); err == nil {
		t.Fatal("ParseToken() accepted a token from another issuer")
	}
}

func setTestJWTConfig(t *testing.T, expire int64) {
	t.Helper()

	original := app.Cfg.JWT
	t.Cleanup(func() { app.Cfg.JWT = original })
	app.Cfg.JWT = app.JWTConfig{
		Secret: "unit-test-secret",
		Expire: expire,
		Issuer: "backplane-test",
	}
}
