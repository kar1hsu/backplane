package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kar1hsu/backplane/internal/pkg/cache"
	jwtpkg "github.com/kar1hsu/backplane/internal/pkg/jwt"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"github.com/kar1hsu/backplane/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepo
}

func NewAuthService() *AuthService {
	return &AuthService{userRepo: repository.NewUserRepo()}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ip string) (*LoginResponse, error) {
	if cache.IsLoginLocked(req.Username, ip) {
		ttl := cache.GetLoginLockTTL(req.Username, ip)
		minutes := int(ttl.Minutes()) + 1
		return nil, fmt.Errorf("登录失败次数过多，请 %d 分钟后重试", minutes)
	}

	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		_, _ = cache.IncrLoginFail(req.Username, ip)
		return nil, errors.New("用户名或密码错误")
	}

	if user.Status != 1 {
		return nil, errors.New("用户已被禁用")
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		count, _ := cache.IncrLoginFail(req.Username, ip)
		remaining := int64(5) - count
		if remaining > 0 {
			return nil, fmt.Errorf("用户名或密码错误，还可尝试 %d 次", remaining)
		}
		return nil, errors.New("登录失败次数过多，账户已被临时锁定")
	}

	cache.ClearLoginFail(req.Username, ip)

	roleCodes := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleCodes = append(roleCodes, r.Code)
	}

	token, err := jwtpkg.GenerateToken(user.ID, user.Username, roleCodes, user.TokenVersion)
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
