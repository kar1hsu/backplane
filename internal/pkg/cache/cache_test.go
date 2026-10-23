package cache

import (
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestUserPermissionAndConfigCache(t *testing.T) {
	server := setupRedisStore(t)

	wantPermissions := []string{"system:user:list", "system:user:add"}
	if err := SetUserPermissions(7, wantPermissions); err != nil {
		t.Fatalf("SetUserPermissions() error = %v", err)
	}
	gotPermissions, ok := GetUserPermissions(7)
	if !ok || !reflect.DeepEqual(gotPermissions, wantPermissions) {
		t.Fatalf("GetUserPermissions() = %v, %v", gotPermissions, ok)
	}

	ClearUserPermissions(7)
	if _, ok := GetUserPermissions(7); ok {
		t.Fatal("ClearUserPermissions() left a cached value")
	}

	if err := SetUserPermissions(8, []string{"a"}); err != nil {
		t.Fatalf("SetUserPermissions(8) error = %v", err)
	}
	if err := SetUserPermissions(9, []string{"b"}); err != nil {
		t.Fatalf("SetUserPermissions(9) error = %v", err)
	}
	ClearAllPermissionCache()
	if _, ok := GetUserPermissions(8); ok {
		t.Fatal("ClearAllPermissionCache() left user 8 cached")
	}
	if _, ok := GetUserPermissions(9); ok {
		t.Fatal("ClearAllPermissionCache() left user 9 cached")
	}

	if err := RebuildConfigCache(map[string]string{"site.name": "Backplane", "feature.enabled": "true"}); err != nil {
		t.Fatalf("RebuildConfigCache() error = %v", err)
	}
	if value, ok := GetConfigCache("site.name"); !ok || value != "Backplane" {
		t.Fatalf("GetConfigCache() = %q, %v", value, ok)
	}
	if err := DelConfigCache("site.name"); err != nil {
		t.Fatalf("DelConfigCache() error = %v", err)
	}
	if _, ok := GetConfigCache("site.name"); ok {
		t.Fatal("DelConfigCache() left a cached value")
	}

	if server.DB(0).Keys() == nil {
		t.Fatal("test Redis should contain the remaining config hash")
	}
}

func TestLoginLimiterAndTokenBlacklist(t *testing.T) {
	server := setupRedisStore(t)
	const window = 15 * time.Minute

	for attempt := int64(1); attempt <= 3; attempt++ {
		count, err := IncrLoginFail("admin", "127.0.0.1", window)
		if err != nil {
			t.Fatalf("IncrLoginFail() attempt %d error = %v", attempt, err)
		}
		if count != attempt {
			t.Fatalf("attempt %d count = %d", attempt, count)
		}
	}
	if ttl := GetLoginLockTTL("admin", "127.0.0.1"); ttl <= 0 || ttl > window {
		t.Fatalf("login lock TTL = %v", ttl)
	}

	// Case and padding variants share the counter.
	if count, _ := IncrLoginFail(" ADMIN ", "127.0.0.1", window); count != 4 {
		t.Fatalf("normalized username count = %d, want 4", count)
	}
	// Another IP has its own budget.
	if count, _ := IncrLoginFail("admin", "10.0.0.1", window); count != 1 {
		t.Fatalf("other IP count = %d, want 1", count)
	}

	DecrLoginFail("admin", "127.0.0.1")
	if count, _ := IncrLoginFail("admin", "127.0.0.1", window); count != 4 {
		t.Fatalf("count after DecrLoginFail = %d, want 4", count)
	}

	server.FastForward(window + time.Second)
	if count, _ := IncrLoginFail("admin", "127.0.0.1", window); count != 1 {
		t.Fatalf("count after the window expired = %d, want 1", count)
	}
	ClearLoginFail("admin", "127.0.0.1")
	if ttl := GetLoginLockTTL("admin", "127.0.0.1"); ttl > 0 {
		t.Fatalf("ClearLoginFail() left TTL %v", ttl)
	}

	if err := BlacklistToken("signed-token", time.Minute); err != nil {
		t.Fatalf("BlacklistToken() error = %v", err)
	}
	if !IsTokenBlacklisted("signed-token") {
		t.Fatal("blacklisted token was not found")
	}
	server.FastForward(time.Minute + time.Second)
	if IsTokenBlacklisted("signed-token") {
		t.Fatal("blacklisted token remained after its TTL expired")
	}
}

func setupRedisStore(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	previous := GetStore()
	InitStore(NewRedisStore(client, "test:"))
	t.Cleanup(func() {
		InitStore(previous)
		_ = client.Close()
	})
	return server
}
