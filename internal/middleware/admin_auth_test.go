package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	jwtpkg "github.com/kar1hsu/backplane/internal/pkg/jwt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Roles must come from the DB on every request: only enabled roles count, and
// a disabled account or bumped token version locks the session out.
func TestAdminAuthLoadsSessionFromDB(t *testing.T) {
	db := setupAdminAuthEnv(t)

	editor := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	auditor := model.SysRole{Name: "Auditor", Code: "auditor", Status: 1}
	deleted := model.SysRole{Name: "Gone", Code: "gone", Status: 1}
	for _, r := range []*model.SysRole{&editor, &auditor, &deleted} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create role: %v", err)
		}
	}
	user := model.SysUser{Username: "alice", Password: "x", Status: 1, TokenVersion: 2,
		Roles: []model.SysRole{editor, auditor, deleted}}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("soft-delete role: %v", err)
	}
	// Create skips the zero status in favor of the column default (1).
	if err := db.Model(&auditor).Update("status", 0).Error; err != nil {
		t.Fatalf("disable role: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/whoami", AdminAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, strings.Join(GetRoleCodes(c), ","))
	})
	call := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/whoami", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	token := mustToken(t, user.ID, "alice", 2)
	if got := call(token); got.Code != http.StatusOK || got.Body.String() != "editor" {
		t.Fatalf("roles = %d %q, want 200 \"editor\"", got.Code, got.Body.String())
	}

	if got := call(mustToken(t, user.ID, "alice", 1)); got.Code != http.StatusUnauthorized {
		t.Fatalf("stale token_version status = %d, want 401", got.Code)
	}

	if err := db.Model(&user).Update("status", 0).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if got := call(token); got.Code != http.StatusUnauthorized {
		t.Fatalf("disabled user status = %d, want 401", got.Code)
	}
}

func setupAdminAuthEnv(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:admin_auth_test?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysRole{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	previousDB, previousJWT := app.DB, app.Cfg.JWT
	app.DB = db
	app.Cfg.JWT = app.JWTConfig{Secret: strings.Repeat("s", 32), Expire: 60, Issuer: "backplane-test"}
	t.Cleanup(func() {
		app.DB = previousDB
		app.Cfg.JWT = previousJWT
	})

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	previousStore := cache.GetStore()
	cache.InitStore(cache.NewRedisStore(client, "auth-test:"))
	t.Cleanup(func() {
		cache.InitStore(previousStore)
		_ = client.Close()
	})
	return db
}

func mustToken(t *testing.T, userID uint, username string, version int) string {
	t.Helper()
	token, err := jwtpkg.GenerateToken(userID, username, version)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	return token
}
