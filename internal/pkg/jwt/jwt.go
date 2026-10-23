package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kar1hsu/backplane/internal/app"
)

// Claims identifies the session only. Roles are deliberately not embedded:
// AdminAuth loads the user's enabled roles from the DB on every request, so a
// role change, disabled role or deleted role takes effect immediately and a
// token can never carry more authority than the DB grants.
type Claims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username string, tokenVersion int) (string, error) {
	cfg := app.Cfg.JWT
	claims := Claims{
		UserID:       userID,
		Username:     username,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func ParseToken(tokenString string) (*Claims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		// Reject non-canonical base64. Otherwise the unused low bits of the
		// last signature character can be flipped and the token still
		// verifies, which slips past the logout blacklist keyed by token text.
		jwt.WithStrictDecoding(),
	}
	if issuer := app.Cfg.JWT.Issuer; issuer != "" {
		opts = append(opts, jwt.WithIssuer(issuer))
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(app.Cfg.JWT.Secret), nil
	}, opts...)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
