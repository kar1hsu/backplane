package api

import (
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/module/api/handler"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "api"
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewConfigHandler()

	rg.GET("/health", healthHandler.Health)
	rg.GET("/configs/public", configHandler.Public)
}
