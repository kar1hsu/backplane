package backplane

import (
	"io/fs"
	"path/filepath"
	"testing"
)

func TestAdminDistIncludesEveryBuildFile(t *testing.T) {
	const distDir = "web/admin/dist"

	err := filepath.WalkDir(distDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		embeddedPath := filepath.ToSlash(filePath)
		if _, err := fs.Stat(AdminDist, embeddedPath); err != nil {
			t.Errorf("frontend build file %q is not embedded: %v", embeddedPath, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk frontend build directory: %v", err)
	}
}
