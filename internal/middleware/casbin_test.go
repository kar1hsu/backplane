package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
)

const testRBACModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act || r.sub == "admin"
`

func TestCasbinRBAC(t *testing.T) {
	model, err := casbinmodel.NewModelFromString(testRBACModel)
	if err != nil {
		t.Fatalf("NewModelFromString() error = %v", err)
	}
	enforcer, err := casbin.NewEnforcer(model)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	if _, err := enforcer.AddPolicy("editor", "/projects/:id", http.MethodGet); err != nil {
		t.Fatalf("AddPolicy(editor) error = %v", err)
	}
	if _, err := enforcer.AddPolicy("writer", "/projects/:id", http.MethodPost); err != nil {
		t.Fatalf("AddPolicy(writer) error = %v", err)
	}

	previous := app.Enforcer
	app.Enforcer = enforcer
	t.Cleanup(func() { app.Enforcer = previous })

	tests := []struct {
		name       string
		roles      []string
		method     string
		wantStatus int
	}{
		{name: "matching role and keyMatch2 path", roles: []string{"editor"}, method: http.MethodGet, wantStatus: http.StatusNoContent},
		{name: "one of multiple roles is allowed", roles: []string{"viewer", "writer"}, method: http.MethodPost, wantStatus: http.StatusNoContent},
		{name: "method is denied", roles: []string{"editor"}, method: http.MethodDelete, wantStatus: http.StatusForbidden},
		{name: "admin bypasses policies", roles: []string{"admin"}, method: http.MethodDelete, wantStatus: http.StatusNoContent},
		{name: "missing roles are denied", roles: nil, method: http.MethodGet, wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(CtxRoleCodesKey, test.roles)
				c.Next()
			})
			router.Use(CasbinRBAC())
			router.Any("/projects/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			request := httptest.NewRequest(test.method, "/projects/42", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
		})
	}
}
