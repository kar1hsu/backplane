package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	jwtpkg "github.com/kar1hsu/backplane/internal/pkg/jwt"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"github.com/kar1hsu/backplane/internal/repository"
)

const (
	CtxUserIDKey    = "user_id"
	CtxUsernameKey  = "username"
	CtxRoleCodesKey = "role_codes"
)

func AdminAuth() gin.HandlerFunc {
	userRepo := repository.NewUserRepo()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Authorization 格式错误")
			return
		}

		tokenString := parts[1]
		if cache.IsTokenBlacklisted(tokenString) {
			response.Unauthorized(c, "Token 已失效，请重新登录")
			return
		}

		claims, err := jwtpkg.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			return
		}

		// 会话撤销：比对 token 版本与用户当前版本
		// 改密 / 禁用 / 改角色 / 删除用户后，旧 token 立即失效
		version, err := userRepo.GetTokenVersion(c.Request.Context(), claims.UserID)
		if err != nil || version != claims.TokenVersion {
			response.Unauthorized(c, "登录状态已失效，请重新登录")
			return
		}

		c.Set(CtxUserIDKey, claims.UserID)
		c.Set(CtxUsernameKey, claims.Username)
		c.Set(CtxRoleCodesKey, claims.RoleCodes)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	val, exists := c.Get(CtxUserIDKey)
	if !exists {
		return 0
	}
	id, _ := val.(uint)
	return id
}

func GetUsername(c *gin.Context) string {
	val, exists := c.Get(CtxUsernameKey)
	if !exists {
		return ""
	}
	name, _ := val.(string)
	return name
}

func GetRoleCodes(c *gin.Context) []string {
	val, exists := c.Get(CtxRoleCodesKey)
	if !exists {
		return nil
	}
	codes, _ := val.([]string)
	return codes
}
