package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var ErrUsage = errors.New("usage requested")

func Parse(args []string, getenv func(string) string) (Config, error) {
	d, err := loadDefaults(args, getenv)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	fs := newFlagSet(&cfg, d, os.Stderr)
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	remaining := fs.Args()
	cfg.FetchURL = strings.TrimSpace(cfg.FetchURL)
	cfg.Search = strings.TrimSpace(cfg.Search)
	cfg.PlaywrightURL = strings.TrimSpace(cfg.PlaywrightURL)
	cfg.SearXURL = strings.TrimSpace(cfg.SearXURL)

	if cfg.Top < 0 {
		return Config{}, errors.New("-top must be >= 0")
	}
	if cfg.PlaywrightTimeoutMS <= 0 {
		return Config{}, errors.New("-playwright-timeout-ms must be > 0")
	}
	if cfg.SearchTimeoutMS <= 0 {
		return Config{}, errors.New("-search-timeout-ms must be > 0")
	}

	expandedCacheDir, err := expandPath(cfg.CacheDir)
	if err != nil {
		return Config{}, fmt.Errorf("invalid cache-dir: %w", err)
	}
	cfg.CacheDir = expandedCacheDir

	switch {
	case cfg.FetchURL != "" && cfg.Search != "":
		return Config{}, errors.New("use either -fetch or -search/-s, not both")
	case cfg.Search != "":
		if len(remaining) > 0 {
			return Config{}, errors.New("positional argument is not supported with -search/-s")
		}
	case cfg.FetchURL != "" && len(remaining) > 0:
		return Config{}, errors.New("provide URL either via -fetch or as positional argument, not both")
	case cfg.FetchURL != "":
		cfg.TargetURL = cfg.FetchURL
	default:
		if len(remaining) == 0 {
			return Config{}, ErrUsage
		}
		positional := strings.TrimSpace(strings.Join(remaining, " "))
		if positional == "" {
			return Config{}, ErrUsage
		}
		if looksLikeURL(positional) {
			cfg.TargetURL = positional
		} else {
			cfg.Search = positional
		}
	}

	return cfg, nil
}

func PrintUsage(w io.Writer) {
	d, err := loadDefaults(nil, os.Getenv)
	if err != nil {
		d = defaults{
			ConfigPath:          "$HOME/.config/answf/config.yml",
			PlaywrightTimeoutMS: 30000,
			SearchTimeoutMS:     30000,
			FallbackTextise:     true,
			TextiseBaseURL:      "https://r.jina.ai",
			CacheDir:            defaultCacheDir(),
		}
	}

	var cfg Config
	fs := newFlagSet(&cfg, d, w)
	fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
	fs.PrintDefaults()
}

func newFlagSet(cfg *Config, d defaults, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("answf", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.ConfigPath, "config", d.ConfigPath, "Path to answf config.yml")
	fs.StringVar(&cfg.FetchURL, "fetch", "", "Fetch and render content from URL")
	fs.StringVar(&cfg.Search, "search", "", "Search query to run against SearXNG and print results")
	fs.StringVar(&cfg.Search, "s", "", "Alias for -search")
	fs.BoolVar(&cfg.Markdown, "md", false, "Output markdown instead of raw HTML")
	fs.StringVar(&cfg.PlaywrightURL, "playwright-url", d.PlaywrightURL, "Playwright Browserless websocket URL")
	fs.StringVar(&cfg.PlaywrightURL, "playwright-ws-endpoint", d.PlaywrightURL, "Alias for -playwright-url")
	fs.StringVar(&cfg.PlaywrightURL, "ws-endpoint", d.PlaywrightURL, "Alias for -playwright-url")
	fs.StringVar(&cfg.SearXURL, "searx-url", d.SearXURL, "SearXNG base URL")
	fs.Float64Var(&cfg.PlaywrightTimeoutMS, "playwright-timeout-ms", d.PlaywrightTimeoutMS, "Playwright fetch timeout in milliseconds")
	fs.Float64Var(&cfg.SearchTimeoutMS, "search-timeout-ms", d.SearchTimeoutMS, "Search request timeout in milliseconds")
	fs.BoolVar(&cfg.FallbackTextise, "fallback-textise", d.FallbackTextise, "Fallback to textise endpoint when browser fetch fails")
	fs.StringVar(&cfg.TextiseBaseURL, "textise-base-url", d.TextiseBaseURL, "Textise fallback base URL")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	fs.IntVar(&cfg.Top, "top", 0, "Limit search results to top N (0 means all)")
	fs.StringVar(&cfg.CacheDir, "cache-dir", d.CacheDir, "Cache directory")
	fs.BoolVar(&cfg.NoCache, "no-cache", false, "Disable cache for -fetch mode only")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fs.PrintDefaults()
	}
	return fs
}

func looksLikeURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return false
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		return err == nil && strings.TrimSpace(u.Host) != ""
	}

	if strings.HasPrefix(raw, "localhost") {
		return true
	}

	return strings.Contains(raw, ".") || strings.Contains(raw, "/")
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".answf-cache"
	}
	return filepath.Join(home, ".cache", "answf")
}

func expandPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("path is required")
	}
	if raw == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~/"))
	}
	return filepath.Clean(raw), nil
}
