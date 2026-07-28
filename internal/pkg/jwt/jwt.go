package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kar1hsu/backplane/internal/app"
)

type Claims struct {
	UserID       uint     `json:"user_id"`
	Username     string   `json:"username"`
	RoleCodes    []string `json:"role_codes"`
	TokenVersion int      `json:"token_version"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username string, roleCodes []string, tokenVersion int) (string, error) {
	cfg := app.Cfg.JWT
	claims := Claims{
		UserID:       userID,
		Username:     username,
		RoleCodes:    roleCodes,
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
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(app.Cfg.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
