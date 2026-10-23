package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── 查询选项 ──

// QueryOptions builds a dynamic query.
//
// SECURITY: field names in Where/Search/Order/Select/Group are interpolated into
// SQL (GORM does not parameterize identifiers). Build them only from trusted
// server-side code, never directly from client input, to avoid SQL injection.
type QueryOptions struct {
	Where    map[string]interface{} // equality conditions (ANDed)
	Search   map[string]string      // field -> keyword, rendered as "field LIKE %kw%"
	Conds    []Cond                 // raw conditions for ranges/ORs Where/Search can't express
	Order    []string               // e.g. ["created_at DESC", "id ASC"]
	Preloads []string               // associations to preload
	Select   []string               // projection columns
	Group    string
	Having   map[string]interface{}

	// OrderColumns sort by columns GORM quotes per dialect; use it for names
	// that are reserved words (e.g. "group"). Applied before Order.
	OrderColumns []clause.OrderByColumn
}

// Cond is a raw query fragment for predicates that Where (equality) and Search
// (single-field LIKE) can't express — ranges, multi-field ORs, IN, etc. Query is
// passed verbatim to GORM's Where; like the rest of QueryOptions it is trusted
// server-side SQL, never raw client input.
type Cond struct {
	Query string
	Args  []interface{}
}

// applyFilter applies only the row-filtering parts (shared by List and Count).
func (q *QueryOptions) applyFilter(db *gorm.DB) *gorm.DB {
	if q == nil {
		return db
	}
	if len(q.Where) > 0 {
		db = db.Where(q.Where)
	}
	for field, kw := range q.Search {
		if kw != "" {
			db = db.Where(field+" LIKE ?", "%"+kw+"%")
		}
	}
	for _, c := range q.Conds {
		if c.Query != "" {
			db = db.Where(c.Query, c.Args...)
		}
	}
	if q.Group != "" {
		db = db.Group(q.Group)
	}
	if len(q.Having) > 0 {
		db = db.Having(q.Having)
	}
	return db
}

// applyAll adds projection/ordering/preloads on top of the filter.
func (q *QueryOptions) applyAll(db *gorm.DB) *gorm.DB {
	db = q.applyFilter(db)
	if q == nil {
		return db
	}
	if len(q.Select) > 0 {
		db = db.Select(q.Select)
	}
	for _, o := range q.OrderColumns {
		db = db.Order(o)
	}
	for _, o := range q.Order {
		db = db.Order(o)
	}
	for _, p := range q.Preloads {
		db = db.Preload(p)
	}
	return db
}

// ── 泛型基础仓储 ──

// BaseRepo provides generic, context-aware CRUD for model T. Embed it in a
// concrete repo; all methods honor transactions carried in ctx (see Transaction).
type BaseRepo[T any] struct{}

func (r BaseRepo[T]) Create(ctx context.Context, m *T) error {
	return dbFrom(ctx).Create(m).Error
}

// GetByID loads by primary key, optionally preloading associations. The error
// (incl. gorm.ErrRecordNotFound) is returned as-is, not masked.
func (r BaseRepo[T]) GetByID(ctx context.Context, id uint, preloads ...string) (*T, error) {
	q := dbFrom(ctx)
	for _, p := range preloads {
		q = q.Preload(p)
	}
	var m T
	if err := q.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetOne returns the first row matching q (honoring Where/Search/Order/Preload/
// Select). Like GetByID it returns gorm.ErrRecordNotFound when nothing matches,
// so callers can errors.Is it — handy for "fetch by some unique column".
func (r BaseRepo[T]) GetOne(ctx context.Context, q *QueryOptions) (*T, error) {
	var m T
	if err := q.applyAll(dbFrom(ctx).Model(new(T))).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Update writes the named struct fields (e.g. "Name","Status"). Passing explicit
// fields avoids Save's implicit association write-back. With no fields it Saves
// the full row.
func (r BaseRepo[T]) Update(ctx context.Context, m *T, fields ...string) error {
	if len(fields) == 0 {
		return dbFrom(ctx).Save(m).Error
	}
	return dbFrom(ctx).Model(m).Select(fields).Updates(m).Error
}

func (r BaseRepo[T]) Delete(ctx context.Context, id uint) error {
	var m T
	return dbFrom(ctx).Delete(&m, id).Error
}

func (r BaseRepo[T]) HardDelete(ctx context.Context, id uint) error {
	var m T
	return dbFrom(ctx).Unscoped().Delete(&m, id).Error
}

func (r BaseRepo[T]) Count(ctx context.Context, q *QueryOptions) (int64, error) {
	var total int64
	if err := q.applyFilter(dbFrom(ctx).Model(new(T))).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// Exists reports whether any row matches q — a thin Count>0 wrapper, handy for
// uniqueness checks without loading the row.
func (r BaseRepo[T]) Exists(ctx context.Context, q *QueryOptions) (bool, error) {
	total, err := r.Count(ctx, q)
	if err != nil {
		return false, err
	}
	return total > 0, nil
}

// CountIDs returns how many of ids exist (soft-deleted rows excluded), so
// callers can reject unknown IDs before Association.Replace would insert them
// as blank records.
func (r BaseRepo[T]) CountIDs(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var total int64
	if err := dbFrom(ctx).Model(new(T)).Where("id IN ?", ids).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r BaseRepo[T]) List(ctx context.Context, q *QueryOptions) ([]T, error) {
	var list []T
	if err := q.applyAll(dbFrom(ctx).Model(new(T))).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r BaseRepo[T]) PageList(ctx context.Context, page, pageSize int, q *QueryOptions) ([]T, int64, error) {
	total, err := r.Count(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	var list []T
	db := q.applyAll(dbFrom(ctx).Model(new(T)))
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
