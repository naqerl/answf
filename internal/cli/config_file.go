package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Fetch  *fetchFileConfig  `yaml:"fetch"`
	Search *searchFileConfig `yaml:"search"`
}

type fetchFileConfig struct {
	PlaywrightURL   *string  `yaml:"playwright_url"`
	TimeoutMS       *float64 `yaml:"timeout_ms"`
	FallbackTextise *bool    `yaml:"fallback_textise"`
	TextiseBaseURL  *string  `yaml:"textise_base_url"`
	Format          *string  `yaml:"format"`
}

type searchFileConfig struct {
	SearXURL  *string  `yaml:"searx_url"`
	TimeoutMS *float64 `yaml:"timeout_ms"`
}

type defaults struct {
	ConfigPath          string
	PlaywrightURL       string
	SearXURL            string
	PlaywrightTimeoutMS float64
	SearchTimeoutMS     float64
	FallbackTextise     bool
	TextiseBaseURL      string
	CacheDir            string
	FetchMarkdown       bool
}

func loadDefaults(args []string, getenv func(string) string) (defaults, error) {
	d := defaults{
		PlaywrightTimeoutMS: 30000,
		SearchTimeoutMS:     30000,
		FallbackTextise:     true,
		TextiseBaseURL:      "https://r.jina.ai",
		CacheDir:            defaultCacheDir(),
		FetchMarkdown:       false,
	}

	resolvedPath, explicitPath, err := resolveConfigPath(args, getenv)
	if err != nil {
		return defaults{}, err
	}
	d.ConfigPath = resolvedPath

	parsed, ok, err := readConfigFile(resolvedPath, explicitPath)
	if err != nil {
		return defaults{}, err
	}
	if !ok {
		return d, nil
	}

	if parsed.Fetch != nil {
		if parsed.Fetch.PlaywrightURL != nil {
			d.PlaywrightURL = strings.TrimSpace(*parsed.Fetch.PlaywrightURL)
		}
		if parsed.Fetch.TimeoutMS != nil {
			d.PlaywrightTimeoutMS = *parsed.Fetch.TimeoutMS
		}
		if parsed.Fetch.FallbackTextise != nil {
			d.FallbackTextise = *parsed.Fetch.FallbackTextise
		}
		if parsed.Fetch.TextiseBaseURL != nil {
			d.TextiseBaseURL = strings.TrimSpace(*parsed.Fetch.TextiseBaseURL)
		}
		if parsed.Fetch.Format != nil {
			format := strings.ToLower(strings.TrimSpace(*parsed.Fetch.Format))
			switch format {
			case "html":
				d.FetchMarkdown = false
			case "md", "markdown":
				d.FetchMarkdown = true
			default:
				return defaults{}, fmt.Errorf("parse config file %q: fetch.format must be one of: html, md", resolvedPath)
			}
		}
	}

	if parsed.Search != nil {
		if parsed.Search.SearXURL != nil {
			d.SearXURL = strings.TrimSpace(*parsed.Search.SearXURL)
		}
		if parsed.Search.TimeoutMS != nil {
			d.SearchTimeoutMS = *parsed.Search.TimeoutMS
		}
	}

	return d, nil
}

func resolveConfigPath(args []string, getenv func(string) string) (string, bool, error) {
	explicitPath, hasExplicitPath, err := extractConfigPath(args)
	if err != nil {
		return "", false, err
	}
	if hasExplicitPath {
		expanded, err := expandPath(explicitPath)
		if err != nil {
			return "", false, fmt.Errorf("invalid config path: %w", err)
		}
		return expanded, true, nil
	}

	if base := strings.TrimSpace(getenv("XDG_CONFIG")); base != "" {
		return filepath.Join(base, "answf", "config.yml"), false, nil
	}
	if base := strings.TrimSpace(getenv("XDG_CONFIG_HOME")); base != "" {
		return filepath.Join(base, "answf", "config.yml"), false, nil
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".config/answf/config.yml", false, nil
	}
	return filepath.Join(home, ".config", "answf", "config.yml"), false, nil
}

func extractConfigPath(args []string) (string, bool, error) {
	var value string
	found := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			if found {
				return "", false, errors.New("--config provided multiple times")
			}
			value = strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
			found = true
			continue
		}
		if arg == "--config" {
			if found {
				return "", false, errors.New("--config provided multiple times")
			}
			if i+1 >= len(args) {
				return "", false, errors.New("--config requires a value")
			}
			value = strings.TrimSpace(args[i+1])
			found = true
			i++
		}
	}

	if !found {
		return "", false, nil
	}
	if value == "" {
		return "", false, errors.New("--config requires a non-empty value")
	}
	return value, true, nil
}

func readConfigFile(path string, explicit bool) (fileConfig, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return fileConfig{}, false, nil
		}
		if os.IsNotExist(err) && explicit {
			return fileConfig{}, false, fmt.Errorf("config file not found: %s", path)
		}
		return fileConfig{}, false, fmt.Errorf("open config file %q: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var cfg fileConfig
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return fileConfig{}, true, nil
		}
		return fileConfig{}, false, fmt.Errorf("parse config file %q: %w", path, err)
	}

	if cfg.Fetch != nil && cfg.Fetch.TimeoutMS != nil && *cfg.Fetch.TimeoutMS <= 0 {
		return fileConfig{}, false, fmt.Errorf("parse config file %q: fetch.timeout_ms must be > 0", path)
	}
	if cfg.Search != nil && cfg.Search.TimeoutMS != nil && *cfg.Search.TimeoutMS <= 0 {
		return fileConfig{}, false, fmt.Errorf("parse config file %q: search.timeout_ms must be > 0", path)
	}

	return cfg, true, nil
}
