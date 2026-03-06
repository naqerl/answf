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
	content := "playwright_url: wss://browserless.example\nsearx_url: https://searx.example\nplaywright_timeout_ms: 12000\nsearch_timeout_ms: 15000\nfallback_textise: false\n"
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
	if cfg.PlaywrightURL == nil || *cfg.PlaywrightURL != "wss://browserless.example" {
		t.Fatalf("unexpected playwright_url: %+v", cfg.PlaywrightURL)
	}
	if cfg.SearXURL == nil || *cfg.SearXURL != "https://searx.example" {
		t.Fatalf("unexpected searx_url: %+v", cfg.SearXURL)
	}
	if cfg.PlaywrightTimeoutMS == nil || *cfg.PlaywrightTimeoutMS != 12000 {
		t.Fatalf("unexpected playwright_timeout_ms: %+v", cfg.PlaywrightTimeoutMS)
	}
	if cfg.SearchTimeoutMS == nil || *cfg.SearchTimeoutMS != 15000 {
		t.Fatalf("unexpected search_timeout_ms: %+v", cfg.SearchTimeoutMS)
	}
	if cfg.FallbackTextise == nil || *cfg.FallbackTextise {
		t.Fatalf("unexpected fallback_textise: %+v", cfg.FallbackTextise)
	}
}
