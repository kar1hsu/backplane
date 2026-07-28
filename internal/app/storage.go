package app

import (
	"fmt"

	"github.com/kar1hsu/backplane/internal/pkg/storage"
)

const bytesPerMegabyte int64 = 1024 * 1024

var Uploader *storage.Local

func InitStorage() error {
	uploader, err := storage.NewLocal(storage.LocalConfig{
		Directory:    Cfg.Storage.Directory,
		PublicURL:    Cfg.Storage.PublicURL,
		MaxBytes:     Cfg.Storage.MaxSize * bytesPerMegabyte,
		AllowedTypes: Cfg.Storage.AllowedTypes,
	})
	if err != nil {
		return fmt.Errorf("initialize local storage: %w", err)
	}
	Uploader = uploader
	return nil
}
