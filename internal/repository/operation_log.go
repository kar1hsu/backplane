package repository

import (
	"context"
	"time"

	"github.com/kar1hsu/backplane/internal/model"
)

type OperationLogRepo struct {
	BaseRepo[model.SysOperationLog]
}

func NewOperationLogRepo() *OperationLogRepo {
	return &OperationLogRepo{}
}

// List/PageList/GetByID/Create come from BaseRepo. Time-range and keyword-OR
// filters are expressed via QueryOptions.Conds (see the service layer).

// DeleteBefore hard-deletes (Unscoped) logs created before t — the retention job
// must reclaim space, not just soft-delete. Returns the number of rows removed.
func (d *OperationLogRepo) DeleteBefore(ctx context.Context, t time.Time) (int64, error) {
	res := dbFrom(ctx).Unscoped().Where("created_at < ?", t).Delete(&model.SysOperationLog{})
	return res.RowsAffected, res.Error
}

// Clear hard-deletes (Unscoped) all operation logs so "清空" truly empties the
// table. The explicit WHERE satisfies GORM's block-global-delete guard.
func (d *OperationLogRepo) Clear(ctx context.Context) error {
	return dbFrom(ctx).Unscoped().Where("1 = 1").Delete(&model.SysOperationLog{}).Error
}
