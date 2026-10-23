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

// purgeBatchSize bounds each hard-delete so purging a large backlog does not
// hold one huge transaction and table lock.
const purgeBatchSize = 5000

// DeleteBefore hard-deletes (Unscoped) logs created before t — the retention job
// must reclaim space, not just soft-delete. Returns the number of rows removed.
func (d *OperationLogRepo) DeleteBefore(ctx context.Context, t time.Time) (int64, error) {
	return d.purge(ctx, "created_at < ?", t)
}

// Clear hard-deletes (Unscoped) all operation logs so "清空" truly empties the
// table.
func (d *OperationLogRepo) Clear(ctx context.Context) error {
	_, err := d.purge(ctx, "1 = 1")
	return err
}

// purge hard-deletes matching rows in id batches until none are left.
func (d *OperationLogRepo) purge(ctx context.Context, query string, args ...interface{}) (int64, error) {
	var total int64
	for {
		var ids []uint
		if err := dbFrom(ctx).Unscoped().Model(&model.SysOperationLog{}).
			Where(query, args...).Order("id").Limit(purgeBatchSize).Pluck("id", &ids).Error; err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}
		res := dbFrom(ctx).Unscoped().Where("id IN ?", ids).Delete(&model.SysOperationLog{})
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if len(ids) < purgeBatchSize {
			return total, nil
		}
	}
}
