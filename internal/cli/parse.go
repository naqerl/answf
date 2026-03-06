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

	cleanArgs, err := stripConfigArgs(args)
	if err != nil {
		return Config{}, err
	}
	if len(cleanArgs) == 0 {
		PrintUsage(os.Stderr)
		return Config{}, ErrUsage
	}

	command := strings.TrimSpace(cleanArgs[0])
	subArgs := cleanArgs[1:]

	switch command {
	case "help", "-h", "--help":
		PrintUsage(os.Stderr)
		return Config{}, ErrUsage
	case "fetch":
		cfg, err := parseFetch(subArgs, d, os.Stderr)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return Config{}, ErrUsage
			}
			return Config{}, err
		}
		return cfg, nil
	case "search":
		cfg, err := parseSearch(subArgs, d, os.Stderr)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return Config{}, ErrUsage
			}
			return Config{}, err
		}
		return cfg, nil
	default:
		PrintUsage(os.Stderr)
		return Config{}, fmt.Errorf("unknown command %q (expected: fetch or search)", command)
	}
}

func parseFetch(args []string, d defaults, output io.Writer) (Config, error) {
	cfg := baseConfigFromDefaults(d)
	fs := flag.NewFlagSet("answf fetch", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&cfg.Markdown, "md", d.FetchMarkdown, "Output markdown")
	forceHTML := false
	fs.BoolVar(&forceHTML, "html", false, "Output HTML")
	fs.BoolVar(&cfg.NoCache, "no-cache", false, "Disable cache")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: answf fetch [flags] <url>\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	mdExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "md" {
			mdExplicit = true
		}
	})
	if forceHTML && mdExplicit {
		return Config{}, errors.New("use only one of -md or -html")
	}
	if forceHTML {
		cfg.Markdown = false
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return Config{}, errors.New("fetch URL is required")
	}
	if len(remaining) > 1 {
		return Config{}, errors.New("fetch expects a single URL argument")
	}

	target := strings.TrimSpace(remaining[0])
	if target == "" {
		return Config{}, errors.New("fetch URL is required")
	}

	expandedCacheDir, err := expandPath(cfg.CacheDir)
	if err != nil {
		return Config{}, fmt.Errorf("invalid cache-dir: %w", err)
	}
	cfg.CacheDir = expandedCacheDir
	cfg.FetchURL = target
	cfg.TargetURL = target
	cfg.Search = ""
	return cfg, nil
}

func parseSearch(args []string, d defaults, output io.Writer) (Config, error) {
	cfg := baseConfigFromDefaults(d)
	fs := flag.NewFlagSet("answf search", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	fs.IntVar(&cfg.Top, "top", 0, "Limit results to top N (0 means all)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: answf search [flags] <query>\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Top < 0 {
		return Config{}, errors.New("-top must be >= 0")
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return Config{}, errors.New("search query is required")
	}
	query := strings.TrimSpace(strings.Join(remaining, " "))
	if query == "" {
		return Config{}, errors.New("search query is required")
	}

	expandedCacheDir, err := expandPath(cfg.CacheDir)
	if err != nil {
		return Config{}, fmt.Errorf("invalid cache-dir: %w", err)
	}
	cfg.CacheDir = expandedCacheDir
	cfg.Search = query
	cfg.FetchURL = ""
	cfg.TargetURL = ""
	cfg.NoCache = false
	cfg.Markdown = d.FetchMarkdown
	return cfg, nil
}

func baseConfigFromDefaults(d defaults) Config {
	return Config{
		ConfigPath:          d.ConfigPath,
		PlaywrightURL:       strings.TrimSpace(d.PlaywrightURL),
		SearXURL:            strings.TrimSpace(d.SearXURL),
		PlaywrightTimeoutMS: d.PlaywrightTimeoutMS,
		SearchTimeoutMS:     d.SearchTimeoutMS,
		FallbackTextise:     d.FallbackTextise,
		TextiseBaseURL:      d.TextiseBaseURL,
		CacheDir:            d.CacheDir,
		Markdown:            d.FetchMarkdown,
	}
}

func stripConfigArgs(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
			if value == "" {
				return nil, errors.New("--config requires a non-empty value")
			}
			continue
		}
		if arg == "--config" {
			if i+1 >= len(args) {
				return nil, errors.New("--config requires a value")
			}
			if strings.TrimSpace(args[i+1]) == "" {
				return nil, errors.New("--config requires a non-empty value")
			}
			i++
			continue
		}
		out = append(out, arg)
	}
	return out, nil
}

func PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  answf [--config path] fetch [flags] <url>")
	fmt.Fprintln(w, "  answf [--config path] search [flags] <query>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  fetch   Fetch and render a URL")
	fmt.Fprintln(w, "  search  Search query through SearXNG")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'answf <command> -h' for command-specific flags.")
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
