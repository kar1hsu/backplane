package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
)

type MenuHandler struct {
	svc *service.MenuService
}

func NewMenuHandler() *MenuHandler {
	return &MenuHandler{svc: service.NewMenuService()}
}

func (h *MenuHandler) Create(c *gin.Context) {
	var req service.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if req.Visible == 0 {
		req.Visible = 1
	}

	if err := h.svc.Create(c.Request.Context(), &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *MenuHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	menu, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, errcode.ErrMenuNotFound, "菜单不存在")
		return
	}
	response.OK(c, menu)
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	var req service.UpdateMenuRequest
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

func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.svc.GetTree(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, tree)
}

func (h *MenuHandler) GetUserMenuTree(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tree, err := h.svc.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, tree)
}
