package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/storage"
)

type uploadResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *UploadResult `json:"data"`
}

func TestUploadHandlerUpload(t *testing.T) {
	root := t.TempDir()
	uploader := newTestUploader(t, root, 1024)
	router := newUploadRouter(uploader)
	request := newUploadRequest(
		t,
		"avatar.png",
		"users/avatars",
		[]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00},
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	result := decodeUploadResponse(t, recorder)
	if result.Code != 0 {
		t.Fatalf("code = %d, message = %q", result.Code, result.Message)
	}
	if result.Data == nil {
		t.Fatal("data is nil")
	}
	if !strings.HasPrefix(result.Data.RelativePath, "users/avatars/") {
		t.Fatalf("path = %q", result.Data.RelativePath)
	}
	if result.Data.ResourceDomain != "https://cdn.example.com" {
		t.Fatalf("resource_domain = %q", result.Data.ResourceDomain)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(result.Data.RelativePath))); err != nil {
		t.Fatalf("uploaded file not found: %v", err)
	}
}

func TestUploadHandlerRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		folder      string
		content     []byte
		maxBytes    int64
		wantMessage string
	}{
		{
			name:        "invalid type",
			fileName:    "script.php",
			content:     []byte("<?php echo 1;"),
			maxBytes:    1024,
			wantMessage: "不支持的文件类型",
		},
		{
			name:        "invalid folder",
			fileName:    "image.png",
			folder:      "../outside",
			content:     []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a},
			maxBytes:    1024,
			wantMessage: "文件夹名称不合法",
		},
		{
			name:        "file too large",
			fileName:    "image.jpg",
			content:     []byte{0xff, 0xd8, 0xff, 0xdb, 0x00},
			maxBytes:    4,
			wantMessage: "文件大小超过限制",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uploader := newTestUploader(t, t.TempDir(), test.maxBytes)
			router := newUploadRouter(uploader)
			request := newUploadRequest(t, test.fileName, test.folder, test.content)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			result := decodeUploadResponse(t, recorder)
			if result.Code != errcode.ErrParam {
				t.Fatalf("code = %d, want %d", result.Code, errcode.ErrParam)
			}
			if result.Message != test.wantMessage {
				t.Fatalf("message = %q, want %q", result.Message, test.wantMessage)
			}
		})
	}
}

func TestUploadHandlerRequiresFile(t *testing.T) {
	uploader := newTestUploader(t, t.TempDir(), 1024)
	router := newUploadRouter(uploader)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("folder", "avatars"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	result := decodeUploadResponse(t, recorder)
	if result.Code != errcode.ErrParam {
		t.Fatalf("code = %d, want %d", result.Code, errcode.ErrParam)
	}
	if result.Message != "请选择上传文件" {
		t.Fatalf("message = %q", result.Message)
	}
}

func newTestUploader(t *testing.T, root string, maxBytes int64) *storage.Local {
	t.Helper()

	uploader, err := storage.NewLocal(storage.LocalConfig{
		Directory: root,
		PublicURL: "/uploads",
		MaxBytes:  maxBytes,
		AllowedTypes: map[string]string{
			"jpg":  "image/jpeg",
			"jpeg": "image/jpeg",
			"png":  "image/png",
			"gif":  "image/gif",
			"webp": "image/webp",
		},
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	return uploader
}

func newUploadRouter(uploader fileUploader) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := &UploadHandler{
		uploader: uploader,
		getResourceDomain: func() string {
			return " https://cdn.example.com/ "
		},
	}
	router.POST("/api/upload", handler.Upload)
	return router
}

func newUploadRequest(
	t *testing.T,
	fileName string,
	folder string,
	content []byte,
) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if folder != "" {
		if err := writer.WriteField("folder", folder); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func decodeUploadResponse(t *testing.T, recorder *httptest.ResponseRecorder) uploadResponse {
	t.Helper()

	var result uploadResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}
