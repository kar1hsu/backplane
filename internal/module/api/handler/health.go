package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/pkg/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	response.OK(c, gin.H{
		"status": "ok",
	})
}
