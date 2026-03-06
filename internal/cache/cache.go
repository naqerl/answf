package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manager struct {
	Dir      string
	Disabled bool
	Now      func() time.Time
}

func KeyForFetch(url string, markdown bool) string {
	return hashKey("fetch|" + strings.TrimSpace(url) + fmt.Sprintf("|md=%t", markdown))
}

func KeyForSearch(query string, searxURL string) string {
	return hashKey("search|" + strings.TrimSpace(query) + "|searx=" + strings.TrimSpace(searxURL))
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (c Manager) nowFunc() func() time.Time {
	if c.Now != nil {
		return c.Now
	}
	return time.Now
}

func (c Manager) cachePath(key string) string {
	return filepath.Join(c.Dir, key+".txt")
}

func (c Manager) Get(key string, ttl time.Duration) (string, bool, error) {
	if c.Disabled {
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

func (c Manager) Set(key string, content string) error {
	if c.Disabled {
		return nil
	}
	if strings.TrimSpace(c.Dir) == "" {
		return fmt.Errorf("cache directory is empty")
	}
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}
	path := c.cachePath(key)
	return os.WriteFile(path, []byte(content), 0o644)
}
