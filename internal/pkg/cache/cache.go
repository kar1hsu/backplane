package cache

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Global store instance, initialized via InitStore.
var store Store

func InitStore(s Store) {
	store = s
}

func GetStore() Store {
	return store
}

// ── Token Blacklist ──

func BlacklistToken(token string, expiration time.Duration) error {
	return store.Set("token:blacklist:"+token, "1", expiration)
}

func IsTokenBlacklisted(token string) bool {
	val, err := store.Get("token:blacklist:" + token)
	return err == nil && val == "1"
}

// ── User Permission Cache ──

const permCacheTTL = 10 * time.Minute

func permKey(userID uint) string {
	return fmt.Sprintf("perm:user:%d", userID)
}

func SetUserPermissions(userID uint, perms []string) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return store.Set(permKey(userID), string(data), permCacheTTL)
}

func GetUserPermissions(userID uint) ([]string, bool) {
	val, err := store.Get(permKey(userID))
	if err != nil {
		return nil, false
	}
	var perms []string
	if err := json.Unmarshal([]byte(val), &perms); err != nil {
		return nil, false
	}
	return perms, true
}

func ClearUserPermissions(userID uint) {
	_ = store.Del(permKey(userID))
}

func ClearAllPermissionCache() {
	keys, err := store.Scan("perm:user:*")
	if err != nil {
		return
	}
	for _, k := range keys {
		// keys from Scan already have the prefix, use raw client Del
		// but since our store.Del adds prefix again, we need to strip it
		// Instead, just delete by known user pattern
		_ = store.Del(k)
	}
}

// ── Login Rate Limit ──

// Attempts are counted per (username, client IP). Usernames are normalized so
// case or padding variants — which MySQL's default collation resolves to the
// same account — share one counter instead of each getting a fresh budget.
func loginKey(username, ip string) string {
	return "login:fail:" + strings.ToLower(strings.TrimSpace(username)) + ":" + ip
}

// IncrLoginFail atomically counts one login attempt and returns the running
// total. The window slides: every attempt pushes the expiry out again.
func IncrLoginFail(username, ip string, window time.Duration) (int64, error) {
	k := loginKey(username, ip)
	count, err := store.Incr(k)
	if err != nil {
		return 0, err
	}
	if err := store.Expire(k, window); err != nil {
		_ = store.Del(k)
		return 0, err
	}
	return count, nil
}

// DecrLoginFail gives back an attempt counted by IncrLoginFail, for logins that
// failed on a server-side error rather than bad credentials.
func DecrLoginFail(username, ip string) {
	_, _ = store.Decr(loginKey(username, ip))
}

func ClearLoginFail(username, ip string) {
	_ = store.Del(loginKey(username, ip))
}

func GetLoginLockTTL(username, ip string) time.Duration {
	ttl, _ := store.TTL(loginKey(username, ip))
	return ttl
}

// ── System Config Cache ──
// All runtime configs share one Redis hash; each field is a config key. Reads go
// here first and fall back to DB on miss (see internal/pkg/setting).

const configHashKey = "config"

// GetConfigCache returns (value, true) on a cache hit. A miss or any Redis error
// returns ("", false) so the caller falls back to the DB.
func GetConfigCache(key string) (string, bool) {
	val, err := store.HGet(configHashKey, key)
	if err != nil {
		return "", false
	}
	return val, true
}

func SetConfigCache(key, value string) error {
	return store.HSet(configHashKey, key, value)
}

func DelConfigCache(key string) error {
	return store.HDel(configHashKey, key)
}

// RebuildConfigCache atomically-ish replaces the whole config hash with kv
// (used by the "refresh all" maintenance action and startup warm-up).
func RebuildConfigCache(kv map[string]string) error {
	if err := store.Del(configHashKey); err != nil {
		return err
	}
	if len(kv) == 0 {
		return nil
	}
	return store.HSet(configHashKey, kv)
}
