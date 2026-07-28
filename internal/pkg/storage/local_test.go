package storage

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestLocalSaveUsesDateFolderByDefault(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads")
	local, err := NewLocal(LocalConfig{
		Directory:    root,
		PublicURL:    "/uploads/",
		MaxBytes:     1024,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	local.now = func() time.Time {
		return time.Date(2026, time.July, 28, 12, 0, 0, 0, time.Local)
	}

	content := []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00}
	header := newFileHeader(t, "Photo.JPG", "application/octet-stream", content)
	result, err := local.Save(header)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	pathPattern := regexp.MustCompile(`^2026/07/28/[0-9a-f]{32}\.jpg$`)
	if !pathPattern.MatchString(result.RelativePath) {
		t.Fatalf("RelativePath = %q, want date folder and random filename", result.RelativePath)
	}
	if result.URL != "/uploads/"+result.RelativePath {
		t.Fatalf("URL = %q, want %q", result.URL, "/uploads/"+result.RelativePath)
	}
	if result.OriginalName != "Photo.JPG" {
		t.Fatalf("OriginalName = %q", result.OriginalName)
	}
	if result.Size != int64(len(content)) {
		t.Fatalf("Size = %d", result.Size)
	}
	if result.ContentType != "image/jpeg" {
		t.Fatalf("ContentType = %q, want image/jpeg", result.ContentType)
	}

	savedPath := filepath.Join(root, filepath.FromSlash(result.RelativePath))
	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(savedContent, content) {
		t.Fatalf("saved content = %v", savedContent)
	}
	info, err := os.Stat(savedPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("file permissions = %o, want 644", info.Mode().Perm())
	}
}

func TestLocalSaveUsesCustomFolder(t *testing.T) {
	local, err := NewLocal(LocalConfig{
		Directory:    t.TempDir(),
		PublicURL:    "https://cdn.example.com/uploads",
		MaxBytes:     1024,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	result, err := local.Save(
		newFileHeader(
			t,
			"avatar.png",
			"image/png",
			[]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00},
		),
		"users/avatars",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !strings.HasPrefix(result.RelativePath, "users/avatars/") {
		t.Fatalf("RelativePath = %q", result.RelativePath)
	}
	if result.URL != "https://cdn.example.com/uploads/"+result.RelativePath {
		t.Fatalf("URL = %q", result.URL)
	}
}

func TestLocalSaveRejectsInvalidFolder(t *testing.T) {
	local, err := NewLocal(LocalConfig{
		Directory:    t.TempDir(),
		PublicURL:    "/uploads",
		MaxBytes:     1024,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	header := newFileHeader(t, "file.txt", "text/plain", []byte("content"))

	for _, folder := range []string{
		"../secret",
		"users/../secret",
		"/absolute",
		`..\secret`,
		"users//avatars",
		"users avatars",
	} {
		t.Run(folder, func(t *testing.T) {
			_, err := local.Save(header, folder)
			if !errors.Is(err, ErrInvalidFolder) {
				t.Fatalf("Save() error = %v, want ErrInvalidFolder", err)
			}
		})
	}
}

func TestLocalSaveChecksActualSize(t *testing.T) {
	root := t.TempDir()
	local, err := NewLocal(LocalConfig{
		Directory:    root,
		PublicURL:    "/uploads",
		MaxBytes:     4,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	header := newFileHeader(
		t,
		"file.jpg",
		"image/jpeg",
		[]byte{0xff, 0xd8, 0xff, 0xdb, 0x00},
	)
	header.Size = 0
	_, err = local.Save(header, "files")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("Save() error = %v, want ErrFileTooLarge", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "files"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("upload folder contains %d files after rejected upload", len(entries))
	}
}

func TestLocalSaveRejectsNilFile(t *testing.T) {
	local, err := NewLocal(LocalConfig{
		Directory:    t.TempDir(),
		PublicURL:    "/uploads",
		MaxBytes:     1024,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	_, err = local.Save(nil, "")
	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("Save() error = %v, want ErrEmptyFile", err)
	}
}

func TestLocalSaveRejectsDisguisedFileTypes(t *testing.T) {
	local, err := NewLocal(LocalConfig{
		Directory:    t.TempDir(),
		PublicURL:    "/uploads",
		MaxBytes:     1024,
		AllowedTypes: testAllowedTypes(),
	})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "script.php",
			content: []byte{0xff, 0xd8, 0xff, 0xdb, 0x00},
		},
		{
			name:    "renamed.jpg",
			content: []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a},
		},
		{
			name:    "page.jpg",
			content: []byte("<html><script>alert(1)</script></html>"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := newFileHeader(t, test.name, "image/jpeg", test.content)
			_, err := local.Save(header)
			if !errors.Is(err, ErrInvalidFileType) {
				t.Fatalf("Save() error = %v, want ErrInvalidFileType", err)
			}
		})
	}
}

func testAllowedTypes() map[string]string {
	return map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
	}
}

func newFileHeader(t *testing.T, name, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	headers := make(map[string][]string)
	headers["Content-Disposition"] = []string{`form-data; name="file"; filename="` + name + `"`}
	headers["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(headers)
	if err != nil {
		t.Fatalf("CreatePart() error = %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatalf("ParseMultipartForm() error = %v", err)
	}
	t.Cleanup(func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	})
	return request.MultipartForm.File["file"][0]
}
