package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{svc: service.NewAuthService()}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}

	res, err := h.svc.Login(c.Request.Context(), &req, c.ClientIP())
	if err != nil {
		// 审计失败登录：用尝试的用户名，userID 记 0
		middleware.RecordLogin(c, "登录", 0, req.Username, false, err.Error())
		response.Fail(c, errcode.ErrPasswordWrong, err.Error())
		return
	}

	middleware.RecordLogin(c, "登录", res.UserID, req.Username, true, "")
	response.OK(c, res)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 {
		h.svc.Logout(c.Request.Context(), parts[1])
	}

	userID := middleware.GetUserID(c)
	cache.ClearUserPermissions(userID)

	middleware.RecordLogin(c, "登出", userID, middleware.GetUsername(c), true, "")
	response.OK(c, nil)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userSvc := service.NewUserService()
	user, err := userSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, errcode.ErrUserNotFound, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AuthHandler) GetPermissions(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Try cache first
	if perms, ok := cache.GetUserPermissions(userID); ok {
		response.OK(c, perms)
		return
	}

	menuSvc := service.NewMenuService()
	perms, err := menuSvc.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}

	cache.SetUserPermissions(userID, perms)
	response.OK(c, perms)
}
