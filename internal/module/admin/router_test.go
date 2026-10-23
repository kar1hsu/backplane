package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The login endpoint is unauthenticated, so it must not buffer oversized bodies.
func TestLoginRejectsOversizedBody(t *testing.T) {
	setupRouterTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/admin"))

	body := `{"username":"` + strings.Repeat("a", int(maxRequestBodyBytes)) + `","password":"x"}`
	request := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var result response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v (body %q)", err, recorder.Body.String())
	}
	if result.Code != errcode.ErrParam || !strings.Contains(result.Message, "too large") {
		t.Fatalf("response = %+v, want ErrParam about body too large", result)
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
