package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheHitWithinTTL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	c := Manager{
		Dir: dir,
		Now: func() time.Time { return now },
	}

	key := "k"
	if err := c.Set(key, "value"); err != nil {
		t.Fatalf("cache set failed: %v", err)
	}
	path := filepath.Join(dir, key+".txt")
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("set file times failed: %v", err)
	}

	got, ok, err := c.Get(key, time.Hour)
	if err != nil {
		t.Fatalf("cache get failed: %v", err)
	}
	if !ok || got != "value" {
		t.Fatalf("expected cache hit with value, got ok=%t value=%q", ok, got)
	}
}

func TestCacheMissAfterTTL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	c := Manager{
		Dir: dir,
		Now: func() time.Time { return now },
	}

	key := "k"
	if err := c.Set(key, "value"); err != nil {
		t.Fatalf("cache set failed: %v", err)
	}
	old := now.Add(-2 * time.Hour)
	path := filepath.Join(dir, key+".txt")
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("set file times failed: %v", err)
	}

	got, ok, err := c.Get(key, time.Hour)
	if err != nil {
		t.Fatalf("cache get failed: %v", err)
	}
	if ok || got != "" {
		t.Fatalf("expected cache miss, got ok=%t value=%q", ok, got)
	}
}

func TestCacheKeyDeterministic(t *testing.T) {
	t.Parallel()

	k1 := KeyForFetch("https://example.com", true)
	k2 := KeyForFetch("https://example.com", true)
	if k1 != k2 {
		t.Fatalf("expected stable key, got %q and %q", k1, k2)
	}

	s1 := KeyForSearch("systemd", "https://searx.example")
	s2 := KeyForSearch("systemd", "https://searx.example")
	if s1 != s2 {
		t.Fatalf("expected stable search key, got %q and %q", s1, s2)
	}
}

func TestNoCacheBypass(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := Manager{
		Dir:      dir,
		Disabled: true,
	}

	if err := c.Set("k", "value"); err != nil {
		t.Fatalf("disabled cache set should not fail: %v", err)
	}
	got, ok, err := c.Get("k", time.Hour)
	if err != nil {
		t.Fatalf("disabled cache get should not fail: %v", err)
	}
	if ok || got != "" {
		t.Fatalf("disabled cache should miss, got ok=%t value=%q", ok, got)
	}
}
