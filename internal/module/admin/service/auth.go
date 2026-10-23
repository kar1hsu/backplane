package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	jwtpkg "github.com/kar1hsu/backplane/internal/pkg/jwt"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"github.com/kar1hsu/backplane/internal/repository"
	"gorm.io/gorm"
)

// Fallbacks for the security.login_* settings when unset or invalid.
const (
	defaultLoginFailLimit   = 5
	defaultLoginLockMinutes = 15
)

type AuthService struct {
	userRepo *repository.UserRepo
}

func NewAuthService() *AuthService {
	return &AuthService{userRepo: repository.NewUserRepo()}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// Login verifies credentials under a per (username, IP) attempt limit.
//
// Unknown usernames and wrong passwords get the same message and the same
// bcrypt cost, and the disabled-account check runs only after the password
// matches, so responses do not reveal which usernames exist.
func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ip string) (*LoginResponse, error) {
	limit, window := loginLimits()

	// Count the attempt before checking it (atomic INCR), so concurrent
	// requests cannot all pass a separate "is it locked?" read first.
	attempts, err := cache.IncrLoginFail(req.Username, ip, window)
	if err != nil {
		app.Log.Warnw("login limiter unavailable", "err", err)
	}
	if attempts > limit {
		minutes := int(cache.GetLoginLockTTL(req.Username, ip).Minutes()) + 1
		return nil, fmt.Errorf("登录失败次数过多，请 %d 分钟后重试", minutes)
	}

	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		cache.DecrLoginFail(req.Username, ip) // a DB failure is not a wrong guess
		app.Log.Errorw("login: load user failed", "err", err)
		return nil, errors.New("登录服务暂时不可用，请稍后重试")
	}

	passwordHash := dummyPasswordHash()
	if user != nil {
		passwordHash = user.Password
	}
	if !utils.CheckPassword(req.Password, passwordHash) || user == nil {
		if remaining := limit - attempts; remaining > 0 {
			return nil, fmt.Errorf("用户名或密码错误，还可尝试 %d 次", remaining)
		}
		return nil, errors.New("登录失败次数过多，账户已被临时锁定")
	}

	if user.Status != 1 {
		return nil, errors.New("用户已被禁用")
	}

	cache.ClearLoginFail(req.Username, ip)

	token, err := jwtpkg.GenerateToken(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return nil, errors.New("生成 Token 失败")
	}

	return &LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	claims, err := jwtpkg.ParseToken(token)
	if err != nil {
		return nil // 无效/已过期的 token 无需拉黑
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	// 只按 token 的剩余寿命拉黑，避免黑名单条目长期占用 Redis
	return cache.BlacklistToken(token, ttl)
}

// loginLimits reads the lockout policy from the runtime settings.
func loginLimits() (int64, time.Duration) {
	limit := setting.GetInt(setting.LoginFailLimitKey)
	if limit <= 0 {
		limit = defaultLoginFailLimit
	}
	minutes := setting.GetInt(setting.LoginLockMinutesKey)
	if minutes <= 0 {
		minutes = defaultLoginLockMinutes
	}
	return int64(limit), time.Duration(minutes) * time.Minute
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// dummyPasswordHash is a bcrypt hash of a random secret, compared against when
// the username does not exist so that path costs the same as a wrong password.
func dummyPasswordHash() string {
	dummyHashOnce.Do(func() {
		secret := make([]byte, 32)
		_, _ = rand.Read(secret)
		dummyHash, _ = utils.HashPassword(fmt.Sprintf("%x", secret))
	})
	return dummyHash
}
