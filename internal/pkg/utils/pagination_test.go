package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPagination(t *testing.T) {
	tests := []struct {
		query      string
		page, size int
	}{
		{query: "", page: 1, size: 10},
		{query: "page=3&page_size=20", page: 3, size: 20},
		{query: "page=0&page_size=0", page: 1, size: 10},
		{query: "page_size=500", page: 1, size: maxPageSize},
		{query: "page=999999", page: maxPage, size: 10},
	}
	for _, tt := range tests {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?"+tt.query, nil)
		got := GetPagination(c)
		if got.Page != tt.page || got.PageSize != tt.size {
			t.Fatalf("GetPagination(%q) = %d/%d, want %d/%d", tt.query, got.Page, got.PageSize, tt.page, tt.size)
		}
	}
}
