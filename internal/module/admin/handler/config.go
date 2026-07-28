package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
)

type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{svc: service.NewConfigService()}
}

func (h *ConfigHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("group"))
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ConfigHandler) Create(c *gin.Context) {
	var req service.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Create(c.Request.Context(), &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

type batchUpdateConfigReq struct {
	Items []service.ConfigItem `json:"items" binding:"required"`
}

func (h *ConfigHandler) BatchUpdate(c *gin.Context) {
	var req batchUpdateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.BatchUpdate(c.Request.Context(), req.Items); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}
	var req service.UpdateConfigRequest
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

func (h *ConfigHandler) Delete(c *gin.Context) {
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

// Refresh re-syncs the cache. ?key=xxx refreshes one key; no key refreshes all.
func (h *ConfigHandler) Refresh(c *gin.Context) {
	if err := h.svc.Refresh(c.Request.Context(), c.Query("key")); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
