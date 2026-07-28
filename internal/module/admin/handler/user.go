package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"github.com/kar1hsu/backplane/internal/tasks"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{svc: service.NewUserService()}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.Create(c.Request.Context(), &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}

	// Example: enqueue a welcome email task after user creation.
	// Failure here must not fail the request — the user has already been created.
	if req.Email != "" {
		if _, err := app.TaskMgr.Client.Enqueue(tasks.TypeEmailSend, tasks.EmailPayload{
			To:      req.Email,
			Subject: "欢迎注册",
			Body:    "您的账号 " + req.Username + " 已创建成功。",
		}); err != nil {
			app.Log.Errorw("enqueue welcome email failed", "username", req.Username, "err", err)
		}
	}

	response.OK(c, nil)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, errcode.ErrUserNotFound, "用户不存在")
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id), middleware.GetUserID(c)); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) List(c *gin.Context) {
	p := utils.GetPagination(c)
	users, total, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{
		"list":  users,
		"total": total,
		"page":  p.Page,
		"size":  p.PageSize,
	})
}
