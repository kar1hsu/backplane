package app

import (
	"strings"
	"testing"
)

func TestValidateConfigJWTSecret(t *testing.T) {
	strongSecret := strings.Repeat("s", minJWTSecretLen)

	tests := []struct {
		name    string
		mode    string
		secret  string
		wantErr bool
	}{
		{name: "empty secret", mode: "debug", secret: "", wantErr: true},
		{name: "default secret in debug mode", mode: "debug", secret: defaultJWTSecret, wantErr: true},
		{name: "default secret in release mode", mode: "release", secret: defaultJWTSecret, wantErr: true},
		{name: "short secret in debug mode", mode: "debug", secret: strongSecret[:minJWTSecretLen-1], wantErr: true},
		{name: "strong secret in debug mode", mode: "debug", secret: strongSecret},
		{name: "strong secret in release mode", mode: "release", secret: strongSecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidConfig(t)
			Cfg.Server.Mode = tt.mode
			Cfg.JWT.Secret = tt.secret

			err := validateConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies []string
		wantErr bool
	}{
		{name: "none", proxies: nil},
		{name: "ip and cidr", proxies: []string{"127.0.0.1", "10.0.0.0/8", "::1"}},
		{name: "invalid entry", proxies: []string{"nginx"}, wantErr: true},
		{name: "invalid cidr", proxies: []string{"10.0.0.0/33"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidConfig(t)
			Cfg.Server.TrustedProxies = tt.proxies

			err := validateConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func setValidConfig(t *testing.T) {
	t.Helper()

	previous := Cfg
	t.Cleanup(func() { Cfg = previous })
	Cfg = Config{
		Server:   ServerConfig{Mode: "debug"},
		Database: DatabaseConfig{Driver: "mysql", DBName: "backplane"},
		JWT:      JWTConfig{Secret: strings.Repeat("s", minJWTSecretLen)},
	}
}
