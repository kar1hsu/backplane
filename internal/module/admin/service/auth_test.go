package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"go.uber.org/zap"
)

func setupLoginTest(t *testing.T) *AuthService {
	t.Helper()
	setupRolePolicyTestEnvironment(t)
	previousLog, previousJWT := app.Log, app.Cfg.JWT
	app.Log = zap.NewNop().Sugar()
	app.Cfg.JWT = app.JWTConfig{Secret: strings.Repeat("s", 32), Expire: 60, Issuer: "backplane-test"}
	t.Cleanup(func() {
		app.Log = previousLog
		app.Cfg.JWT = previousJWT
	})

	hash, err := utils.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	mustCreate(t, &model.SysUser{Username: "alice", Password: hash, Status: 1})
	disabled := model.SysUser{Username: "dave", Password: hash, Status: 1}
	mustCreate(t, &disabled)
	if err := app.DB.Model(&disabled).Update("status", 0).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	return NewAuthService()
}

func login(svc *AuthService, username, password, ip string) error {
	_, err := svc.Login(context.Background(), &LoginRequest{Username: username, Password: password}, ip)
	return err
}

// Unknown users, wrong passwords and disabled accounts with a wrong password
// must be indistinguishable to the caller.
func TestLoginDoesNotRevealWhichUsernamesExist(t *testing.T) {
	svc := setupLoginTest(t)

	unknown := login(svc, "nobody", "whatever", "10.0.0.1")
	wrong := login(svc, "alice", "whatever", "10.0.0.2")
	disabledWrong := login(svc, "dave", "whatever", "10.0.0.3")
	if unknown == nil || wrong == nil || disabledWrong == nil {
		t.Fatalf("errors = %v / %v / %v, want all non-nil", unknown, wrong, disabledWrong)
	}
	if unknown.Error() != wrong.Error() || wrong.Error() != disabledWrong.Error() {
		t.Fatalf("messages differ: %q / %q / %q", unknown, wrong, disabledWrong)
	}

	if err := login(svc, "dave", "correct-password", "10.0.0.4"); err == nil || !strings.Contains(err.Error(), "禁用") {
		t.Fatalf("disabled user with the right password error = %v", err)
	}
	if err := login(svc, "alice", "correct-password", "10.0.0.5"); err != nil {
		t.Fatalf("valid login error = %v", err)
	}
}

func TestLoginLockoutFollowsSettings(t *testing.T) {
	svc := setupLoginTest(t)
	mustCreate(t, &model.SysConfig{Key: setting.LoginFailLimitKey, Value: "3", Type: "int"})

	for i := 1; i <= 3; i++ {
		// Case and padding variants count against the same budget.
		username := []string{"alice", "ALICE", " alice "}[i-1]
		if err := login(svc, username, "wrong", "10.0.0.9"); err == nil {
			t.Fatalf("attempt %d succeeded with a wrong password", i)
		}
	}
	err := login(svc, "alice", "correct-password", "10.0.0.9")
	if err == nil || !strings.Contains(err.Error(), "分钟后重试") {
		t.Fatalf("login after the limit error = %v, want locked", err)
	}
	if err := login(svc, "alice", "correct-password", "10.0.0.10"); err != nil {
		t.Fatalf("login from another IP error = %v", err)
	}
}
