package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCaptureRequestRedactsSecrets(t *testing.T) {
	const secret = "PlainSecret123"

	tests := []struct {
		name        string
		target      string
		contentType string
		body        string
		wantKept    string
	}{
		{
			name:        "json body",
			target:      "/admin/users",
			contentType: "application/json",
			body:        `{"username":"bob","new_password":"` + secret + `"}`,
			wantKept:    "bob",
		},
		{
			name:        "json body cut off at the capture limit",
			target:      "/admin/users",
			contentType: "application/json",
			body:        `{"password":"` + secret + `","remark":"` + strings.Repeat("x", maxBodyCapture) + `"}`,
		},
		{
			name:        "urlencoded body",
			target:      "/admin/users",
			contentType: "application/x-www-form-urlencoded",
			body:        "username=bob&password=" + secret,
			wantKept:    "bob",
		},
		{
			name:     "query string",
			target:   "/admin/users?page=2&access_token=" + secret,
			wantKept: "page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			if tt.contentType != "" {
				c.Request.Header.Set("Content-Type", tt.contentType)
			}

			got := captureRequest(c)
			if strings.Contains(got, secret) {
				t.Fatalf("captured params leak the secret: %s", got)
			}
			if tt.wantKept != "" && !strings.Contains(got, tt.wantKept) {
				t.Fatalf("captured params %s lost %q", got, tt.wantKept)
			}
		})
	}
}
