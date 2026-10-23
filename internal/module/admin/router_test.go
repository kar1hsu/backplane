package admin

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Every admin route, including the unauthenticated login, must reject bodies
// over the limit instead of buffering them. A probe route on the same group
// inherits the group middleware, so it observes exactly what handlers get.
func TestAdminRoutesLimitRequestBody(t *testing.T) {
	setupRouterTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/admin")
	New().RegisterRoutes(group)
	group.POST("/probe", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name string
		size int64
		want int
	}{
		{name: "within limit", size: maxRequestBodyBytes, want: http.StatusOK},
		{name: "over limit", size: maxRequestBodyBytes + 1, want: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(strings.Repeat("a", int(tt.size)))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/admin/probe", body))
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

// setupRouterTestDB provides the sys_api table that OperationLog reads when the
// routes are registered.
func setupRouterTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:admin_router_test?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysAPI{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	previous := app.DB
	app.DB = db
	t.Cleanup(func() { app.DB = previous })
}
