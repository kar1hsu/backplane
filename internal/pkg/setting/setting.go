// Package setting is the runtime accessor for admin-editable system config.
//
// Source of truth is the sys_config table; a shared Redis hash is the cache.
// Because every instance reads the same Redis, multi-instance deployments stay
// consistent without pub/sub — refreshing the cache (single key or all) is an
// explicit admin action. Reads degrade gracefully: Redis hit → DB → compiled
// default, so a Redis/DB blip never takes the app down.
package setting

import (
	"context"
	"strconv"

	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/kar1hsu/backplane/internal/repository"
)

var repo = repository.NewConfigRepo()

// Get resolves a config value: Redis cache → DB (backfilling cache) → the
// compiled-in default. Never errors — missing/unknown keys yield "".
func Get(key string) string {
	if v, ok := cache.GetConfigCache(key); ok {
		return v
	}
	c, err := repo.GetByKey(context.Background(), key)
	if err != nil {
		return defaultValue(key)
	}
	_ = cache.SetConfigCache(key, c.Value) // backfill; no-op if Redis is down
	return c.Value
}

func GetString(key string) string { return Get(key) }

func GetInt(key string) int {
	n, _ := strconv.Atoi(Get(key))
	return n
}

func GetInt64(key string) int64 {
	n, _ := strconv.ParseInt(Get(key), 10, 64)
	return n
}

func GetBool(key string) bool {
	v := Get(key)
	return v == "true" || v == "1"
}

func GetFloat(key string) float64 {
	f, _ := strconv.ParseFloat(Get(key), 64)
	return f
}

// Set writes to the DB (source of truth) then write-through the cache.
func Set(ctx context.Context, key, value string) error {
	if err := repo.UpdateValue(ctx, key, value); err != nil {
		return err
	}
	_ = cache.SetConfigCache(key, value)
	return nil
}

// RefreshKey reloads one key from DB into the cache; if the key no longer exists
// in DB its stale cache field is dropped. (Powers "单独刷新缓存".)
func RefreshKey(ctx context.Context, key string) error {
	c, err := repo.GetByKey(ctx, key)
	if err != nil {
		return cache.DelConfigCache(key)
	}
	return cache.SetConfigCache(key, c.Value)
}

// RefreshAll rebuilds the whole cache hash from DB. (Powers "一键刷新全部缓存"
// and the startup warm-up.)
func RefreshAll(ctx context.Context) error {
	list, err := repo.ListAll(ctx)
	if err != nil {
		return err
	}
	kv := make(map[string]string, len(list))
	for i := range list {
		kv[list[i].Key] = list[i].Value
	}
	return cache.RebuildConfigCache(kv)
}

// Init seeds any missing default configs into DB and warms the cache. Call from
// main after AutoMigrate (the sys_config table must exist).
func Init(ctx context.Context) error {
	if err := seedDefaults(ctx); err != nil {
		return err
	}
	return RefreshAll(ctx)
}
