package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
)

func Cors() gin.HandlerFunc {

	// in production; an empty list falls back to "*".
	origins := app.Cfg.Server.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	return cors.New(cors.Config{
		AllowOrigins:  origins,
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	})
}
