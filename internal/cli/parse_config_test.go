package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLoadsConfigDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("playwright_url: wss://browserless.example\nsearx_url: https://searx.example\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse([]string{"--config", cfgPath, "-search", "systemd sandboxing"}, os.Getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.PlaywrightURL != "wss://browserless.example" {
		t.Fatalf("unexpected ws endpoint: %q", cfg.PlaywrightURL)
	}
	if cfg.SearXURL != "https://searx.example" {
		t.Fatalf("unexpected searx url: %q", cfg.SearXURL)
	}
}

func TestParseCLIOverridesConfigDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("searx_url: https://searx.example\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse([]string{"--config", cfgPath, "-search", "q", "--searx-url", "https://override.example"}, os.Getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.SearXURL != "https://override.example" {
		t.Fatalf("expected CLI override, got %q", cfg.SearXURL)
	}
}

func TestParseUsesXDGConfigPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	xdgBase := filepath.Join(dir, "xdg")
	cfgDir := filepath.Join(xdgBase, "answf")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("searx_url: https://searx.example\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	getenv := func(key string) string {
		if key == "XDG_CONFIG" {
			return xdgBase
		}
		return ""
	}

	cfg, err := Parse([]string{"-search", "q"}, getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.SearXURL != "https://searx.example" {
		t.Fatalf("expected xdg config value, got %q", cfg.SearXURL)
	}
}
