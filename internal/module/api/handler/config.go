package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"github.com/kar1hsu/backplane/internal/repository"
)

type ConfigHandler struct {
	repo *repository.ConfigRepo
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{repo: repository.NewConfigRepo()}
}

// Public returns is_public configs as a key→value map for the unauthenticated
// SPA bootstrap (e.g. site name / logo on the login page).
func (h *ConfigHandler) Public(c *gin.Context) {
	list, err := h.repo.ListPublic(c.Request.Context())
	if err != nil {
		// Unauthenticated endpoint: log the cause, don't echo DB errors.
		app.Log.Errorw("list public configs failed", "err", err)
		response.Fail(c, errcode.ErrServer, "服务器内部错误")
		return
	}
	out := make(map[string]string, len(list))
	for i := range list {
		out[list[i].Key] = list[i].Value
	}
	response.OK(c, out)
}
