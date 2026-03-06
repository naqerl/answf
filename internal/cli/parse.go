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
	var cfg Config
	fs := flag.NewFlagSet("answf", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fs.PrintDefaults()
	}

	fs.StringVar(&cfg.FetchURL, "fetch", "", "Fetch and render content from URL")
	fs.StringVar(&cfg.Search, "search", "", "Search query to run against SearXNG and print results")
	fs.StringVar(&cfg.Search, "s", "", "Alias for -search")
	fs.BoolVar(&cfg.Markdown, "md", false, "Output markdown instead of raw HTML")
	fs.StringVar(&cfg.WSEndpoint, "ws-endpoint", firstNonEmpty(getenv("BROWSERLESS_WS_ENDPOINT"), "wss://browserless.aishift.co"), "Browserless websocket endpoint")
	fs.StringVar(&cfg.SearXURL, "searx-url", firstNonEmpty(getenv("SEARX_URL"), "https://searx.aishift.co"), "SearXNG base URL")
	fs.Float64Var(&cfg.TimeoutMS, "timeout-ms", 30000, "Navigation timeout in milliseconds")
	fs.BoolVar(&cfg.FallbackTextise, "fallback-textise", true, "Fallback to textise endpoint when browser fetch fails")
	fs.StringVar(&cfg.TextiseBaseURL, "textise-base-url", "https://r.jina.ai/http://", "Textise fallback base URL")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	fs.IntVar(&cfg.Top, "top", 0, "Limit search results to top N (0 means all)")
	fs.StringVar(&cfg.CacheDir, "cache-dir", defaultCacheDir(), "Cache directory")
	fs.BoolVar(&cfg.NoCache, "no-cache", false, "Disable local cache reads/writes")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	remaining := fs.Args()
	cfg.FetchURL = strings.TrimSpace(cfg.FetchURL)
	cfg.Search = strings.TrimSpace(cfg.Search)

	if cfg.Top < 0 {
		return Config{}, errors.New("-top must be >= 0")
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
	fs := flag.NewFlagSet("answf", flag.ContinueOnError)
	fs.SetOutput(w)
	fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
	fs.String("fetch", "", "Fetch and render content from URL")
	fs.String("search", "", "Search query to run against SearXNG and print results")
	fs.String("s", "", "Alias for -search")
	fs.Bool("md", false, "Output markdown instead of raw HTML")
	fs.String("ws-endpoint", "wss://browserless.aishift.co", "Browserless websocket endpoint")
	fs.String("searx-url", "https://searx.aishift.co", "SearXNG base URL")
	fs.Float64("timeout-ms", 30000, "Navigation timeout in milliseconds")
	fs.Bool("fallback-textise", true, "Fallback to textise endpoint when browser fetch fails")
	fs.String("textise-base-url", "https://r.jina.ai/http://", "Textise fallback base URL")
	fs.Bool("v", false, "Verbose output")
	fs.Bool("verbose", false, "Verbose output")
	fs.Int("top", 0, "Limit search results to top N (0 means all)")
	fs.String("cache-dir", defaultCacheDir(), "Cache directory")
	fs.Bool("no-cache", false, "Disable local cache reads/writes")
	fs.PrintDefaults()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
