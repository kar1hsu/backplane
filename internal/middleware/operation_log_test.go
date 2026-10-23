package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/model"
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

func TestFitColumnsTruncatesByCharacter(t *testing.T) {
	entry := &model.SysOperationLog{
		Username:  strings.Repeat("管", 100),
		UserAgent: strings.Repeat("a", 1000),
		Method:    "POST",
	}
	fitColumns(entry)

	if got := utf8.RuneCountInString(entry.Username); got != 64 || !utf8.ValidString(entry.Username) {
		t.Fatalf("username runes = %d (valid %v), want 64", got, utf8.ValidString(entry.Username))
	}
	if len(entry.UserAgent) != 255 || entry.Method != "POST" {
		t.Fatalf("user agent len = %d, method = %q", len(entry.UserAgent), entry.Method)
	}
}
