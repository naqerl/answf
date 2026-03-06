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
	content := "fetch:\n  playwright_url: wss://browserless.example\n  format: md\nsearch:\n  searx_url: https://searx.example\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse([]string{"--config", cfgPath, "search", "systemd sandboxing"}, os.Getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.PlaywrightURL != "wss://browserless.example" {
		t.Fatalf("unexpected ws endpoint: %q", cfg.PlaywrightURL)
	}
	if cfg.SearXURL != "https://searx.example" {
		t.Fatalf("unexpected searx url: %q", cfg.SearXURL)
	}
	if !cfg.Markdown {
		t.Fatalf("expected markdown default from fetch.format=md")
	}
}

func TestParseUsesConfigValuesForOperationalSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	content := "fetch:\n  playwright_url: wss://browserless.example\n  timeout_ms: 12000\nsearch:\n  searx_url: https://searx.example\n  timeout_ms: 15000\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse([]string{"--config", cfgPath, "search", "q"}, os.Getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.PlaywrightURL != "wss://browserless.example" {
		t.Fatalf("unexpected playwright url: %q", cfg.PlaywrightURL)
	}
	if cfg.PlaywrightTimeoutMS != 12000 {
		t.Fatalf("unexpected fetch timeout: %v", cfg.PlaywrightTimeoutMS)
	}
	if cfg.SearXURL != "https://searx.example" {
		t.Fatalf("unexpected searx url: %q", cfg.SearXURL)
	}
	if cfg.SearchTimeoutMS != 15000 {
		t.Fatalf("unexpected search timeout: %v", cfg.SearchTimeoutMS)
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
	if err := os.WriteFile(cfgPath, []byte("search:\n  searx_url: https://searx.example\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	getenv := func(key string) string {
		if key == "XDG_CONFIG" {
			return xdgBase
		}
		return ""
	}

	cfg, err := Parse([]string{"search", "q"}, getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.SearXURL != "https://searx.example" {
		t.Fatalf("expected xdg config value, got %q", cfg.SearXURL)
	}
}

func TestParseHTMLOverridesMarkdownDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("fetch:\n  format: md\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse([]string{"--config", cfgPath, "fetch", "-html", "example.com"}, os.Getenv)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.Markdown {
		t.Fatalf("expected -html to force HTML output")
	}
}

func TestParseRejectsMDAndHTMLTogether(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"fetch", "-md", "-html", "example.com"}, os.Getenv)
	if err == nil {
		t.Fatalf("expected error when both -md and -html are set")
	}
}
