package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractConfigPath(t *testing.T) {
	t.Parallel()

	got, ok, err := extractConfigPath([]string{"--config", "./a.yml", "-s", "q"})
	if err != nil {
		t.Fatalf("extractConfigPath error: %v", err)
	}
	if !ok || got != "./a.yml" {
		t.Fatalf("unexpected result ok=%t value=%q", ok, got)
	}

	got, ok, err = extractConfigPath([]string{"--config=./b.yml", "-s", "q"})
	if err != nil {
		t.Fatalf("extractConfigPath error: %v", err)
	}
	if !ok || got != "./b.yml" {
		t.Fatalf("unexpected result ok=%t value=%q", ok, got)
	}
}

func TestResolveConfigPathFromXDGConfig(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	getenv := func(key string) string {
		if key == "XDG_CONFIG" {
			return base
		}
		return ""
	}

	got, explicit, err := resolveConfigPath(nil, getenv)
	if err != nil {
		t.Fatalf("resolveConfigPath error: %v", err)
	}
	if explicit {
		t.Fatalf("expected non-explicit path")
	}
	want := filepath.Join(base, "answf", "config.yml")
	if got != want {
		t.Fatalf("unexpected config path: got %q want %q", got, want)
	}
}

func TestReadConfigFileExplicitMissing(t *testing.T) {
	t.Parallel()

	_, _, err := readConfigFile(filepath.Join(t.TempDir(), "missing.yml"), true)
	if err == nil {
		t.Fatalf("expected error for missing explicit config")
	}
}

func TestReadConfigFileParsesYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := "fetch:\n  playwright_url: wss://browserless.example\n  timeout_ms: 12000\n  fallback_textise: false\n  format: md\nsearch:\n  searx_url: https://searx.example\n  timeout_ms: 15000\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg, ok, err := readConfigFile(path, true)
	if err != nil {
		t.Fatalf("readConfigFile error: %v", err)
	}
	if !ok {
		t.Fatalf("expected config file to be loaded")
	}
	if cfg.Fetch == nil || cfg.Fetch.PlaywrightURL == nil || *cfg.Fetch.PlaywrightURL != "wss://browserless.example" {
		t.Fatalf("unexpected fetch.playwright_url")
	}
	if cfg.Fetch.TimeoutMS == nil || *cfg.Fetch.TimeoutMS != 12000 {
		t.Fatalf("unexpected fetch.timeout_ms: %+v", cfg.Fetch.TimeoutMS)
	}
	if cfg.Fetch.Format == nil || *cfg.Fetch.Format != "md" {
		t.Fatalf("unexpected fetch.format: %+v", cfg.Fetch.Format)
	}
	if cfg.Search == nil || cfg.Search.SearXURL == nil || *cfg.Search.SearXURL != "https://searx.example" {
		t.Fatalf("unexpected search.searx_url")
	}
	if cfg.Search.TimeoutMS == nil || *cfg.Search.TimeoutMS != 15000 {
		t.Fatalf("unexpected search.timeout_ms: %+v", cfg.Search.TimeoutMS)
	}
}
