package server

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"go.uber.org/zap"
)

type clientIPModule struct{}

func (clientIPModule) Name() string { return "probe" }

func (clientIPModule) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
}

func TestNewRouterClientIPTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies []string
		want    string
	}{
		{name: "forwarded headers ignored by default", proxies: nil, want: "203.0.113.7"},
		{name: "forwarded headers honored from trusted proxy", proxies: []string{"203.0.113.0/24"}, want: "198.51.100.9"},
		{name: "forwarded headers ignored from other proxies", proxies: []string{"10.0.0.0/8"}, want: "203.0.113.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRouterTestConfig(t, tt.proxies)
			router := NewRouter(embed.FS{}, clientIPModule{})

			request := httptest.NewRequest(http.MethodGet, "/probe/ip", nil)
			request.RemoteAddr = "203.0.113.7:51234"
			request.Header.Set("X-Forwarded-For", "198.51.100.9")
			request.Header.Set("X-Real-IP", "198.51.100.9")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if got := recorder.Body.String(); got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func setRouterTestConfig(t *testing.T, proxies []string) {
	t.Helper()

	previousCfg, previousLog := app.Cfg, app.Log
	t.Cleanup(func() {
		app.Cfg = previousCfg
		app.Log = previousLog
	})
	app.Cfg.Server.Mode = gin.TestMode
	app.Cfg.Server.TrustedProxies = proxies
	app.Log = zap.NewNop().Sugar()
}
