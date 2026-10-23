package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/echo", BodyLimit(16), func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusOK, string(body))
	})

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "within limit", body: strings.Repeat("a", 16), want: http.StatusOK},
		{name: "over limit", body: strings.Repeat("a", 17), want: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(tt.body)))
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}
