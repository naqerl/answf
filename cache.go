package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type cacheManager struct {
	dir      string
	disabled bool
	now      func() time.Time
}

func keyForFetch(url string, markdown bool) string {
	return hashKey("fetch|" + strings.TrimSpace(url) + fmt.Sprintf("|md=%t", markdown))
}

func keyForSearch(query string, searxURL string) string {
	return hashKey("search|" + strings.TrimSpace(query) + "|searx=" + strings.TrimSpace(searxURL))
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (c cacheManager) nowFunc() func() time.Time {
	if c.now != nil {
		return c.now
	}
	return time.Now
}

func (c cacheManager) cachePath(key string) string {
	return filepath.Join(c.dir, key+".txt")
}

func (c cacheManager) Get(key string, ttl time.Duration) (string, bool, error) {
	if c.disabled {
		return "", false, nil
	}
	path := c.cachePath(key)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	if ttl > 0 && c.nowFunc()().Sub(info.ModTime()) > ttl {
		return "", false, nil
	}

	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(body), true, nil
}

func (c cacheManager) Set(key string, content string) error {
	if c.disabled {
		return nil
	}
	if strings.TrimSpace(c.dir) == "" {
		return fmt.Errorf("cache directory is empty")
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	path := c.cachePath(key)
	return os.WriteFile(path, []byte(content), 0o644)
}
