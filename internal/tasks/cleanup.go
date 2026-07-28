package tasks

import (
	"context"
	"time"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/repository"
)

func HandleCleanup(ctx context.Context, payload []byte) error {
	retainDays := setting.GetInt64("log.operation_retain_days")
	if retainDays <= 0 {
		app.Log.Infow("[task] operation log cleanup skipped", "retain_days", retainDays)
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -int(retainDays))
	deleted, err := repository.NewOperationLogRepo().DeleteBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	app.Log.Infow("[task] operation log cleanup completed", "retain_days", retainDays, "cutoff", cutoff, "deleted", deleted)
	return nil
}
