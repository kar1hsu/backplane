package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
)

type OperationLogHandler struct {
	svc *service.OperationLogService
}

func NewOperationLogHandler() *OperationLogHandler {
	return &OperationLogHandler{svc: service.NewOperationLogService()}
}

func (h *OperationLogHandler) List(c *gin.Context) {
	p := utils.GetPagination(c)

	var success *bool
	if s := c.Query("success"); s != "" {
		b := s == "true" || s == "1"
		success = &b
	}

	list, total, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &service.ListOperationLogRequest{
		Username:  c.Query("username"),
		Module:    c.Query("module"),
		ClientIP:  c.Query("client_ip"),
		Success:   success,
		Keyword:   c.Query("keyword"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
	})
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{
		"list":  list,
		"total": total,
		"page":  p.Page,
		"size":  p.PageSize,
	})
}

func (h *OperationLogHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误")
		return
	}
	log, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, "日志不存在")
		return
	}
	response.OK(c, log)
}

func (h *OperationLogHandler) Delete(c *gin.Context) {
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

// Clear removes all operation logs (admin maintenance action).
func (h *OperationLogHandler) Clear(c *gin.Context) {
	if err := h.svc.Clear(c.Request.Context()); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
