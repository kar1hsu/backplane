package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/app"
	"gorm.io/gorm"
)

// ── 事务：通过 context 传递 ──

type txKey struct{}

// Transaction runs fn inside a DB transaction. Any repo call that uses the ctx
// handed to fn automatically joins the same transaction — so multiple repos can
// be composed atomically without passing *gorm.DB around.
func Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// 已在事务中（ctx 已携带 tx）则复用外层事务，避免嵌套调用各开一个独立
	// 事务，导致外层回滚而内层已提交的不一致。
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return app.DB.Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// dbFrom returns the active *gorm.DB: the transaction stored in ctx if present,
// otherwise the global app.DB. The ctx is always attached for cancellation.
func dbFrom(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return app.DB.WithContext(ctx)
}
