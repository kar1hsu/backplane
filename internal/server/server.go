package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/pkg/response"
)

type Module interface {
	Name() string
	RegisterRoutes(rg *gin.RouterGroup)
}

func NewRouter(adminDist embed.FS, modules ...Module) *gin.Engine {
	gin.SetMode(app.Cfg.Server.Mode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())

	apiPrefixes := make([]string, 0, len(modules))
	for _, m := range modules {
		group := r.Group("/" + m.Name())
		m.RegisterRoutes(group)
		apiPrefixes = append(apiPrefixes, "/"+m.Name()+"/")
		app.Log.Infof("module [%s] registered at /%s", m.Name(), m.Name())
	}

	setupStaticFiles(r, adminDist, apiPrefixes)

	return r
}

func setupStaticFiles(r *gin.Engine, adminDist embed.FS, apiPrefixes []string) {
	subFS, err := fs.Sub(adminDist, "web/admin/dist")
	if err != nil {
		app.Log.Warnf("embed admin dist not found: %v", err)
		return
	}

	staticHandler := http.FileServer(http.FS(subFS))

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// Unmatched API routes must return JSON 404, not the SPA index.html.
		for _, prefix := range apiPrefixes {
			if strings.HasPrefix(reqPath, prefix) {
				response.NotFound(c, "接口不存在")
				return
			}
		}

		// Sanitize the path before hitting the filesystem, then try to serve
		// the requested static asset directly.
		name := strings.TrimPrefix(path.Clean("/"+reqPath), "/")
		if name != "" && fs.ValidPath(name) {
			if f, err := subFS.Open(name); err == nil {
				f.Close()
				staticHandler.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// Fallback to index.html for SPA client-side routing.
		c.Request.URL.Path = "/"
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})

	app.Log.Info("admin panel static files registered")
}
