package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		app.Log.Infow("request",
			"status", status,
			"method", method,
			"path", path,
			"query", query,
			"ip", clientIP,
			"latency", latency.String(),
			"errors", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}
