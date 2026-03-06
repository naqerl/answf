package fetch

import (
	"fmt"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/naqerl/answf/internal/cache"
	"github.com/naqerl/answf/internal/netx"
)

type Config struct {
	TargetURL       string
	WSEndpoint      string
	Timeout         time.Duration
	Markdown        bool
	FallbackTextise bool
	TextiseBaseURL  string
}

func Run(cfg Config, c cache.Manager) (string, error) {
	target, err := netx.NormalizeHTTPURL(cfg.TargetURL)
	if err != nil {
		return "", err
	}

	cacheKey := cache.KeyForFetch(target, cfg.Markdown)
	if cached, ok, err := c.Get(cacheKey, 24*time.Hour); err == nil && ok {
		return cached, nil
	} else if err != nil {
		return "", fmt.Errorf("read fetch cache: %w", err)
	}

	content, isHTML, err := fetchWithPlaywright(cfg, target)
	if err != nil {
		if !cfg.FallbackTextise {
			return "", err
		}
		fallbackContent, fallbackErr := fetchViaTextise(target, cfg.Timeout, cfg.TextiseBaseURL)
		if fallbackErr != nil {
			return "", fmt.Errorf("playwright fetch failed: %v; textise fallback failed: %w", err, fallbackErr)
		}
		content = fallbackContent
		isHTML = false
	}

	output, err := finalizeFetchedContent(content, isHTML, cfg.Markdown)
	if err != nil {
		return "", err
	}
	if err := c.Set(cacheKey, output); err != nil {
		return "", fmt.Errorf("write fetch cache: %w", err)
	}
	return output, nil
}

func finalizeFetchedContent(content string, isHTML, markdown bool) (string, error) {
	if markdown && isHTML {
		markdownOut, err := htmltomarkdown.ConvertString(content)
		if err != nil {
			return "", fmt.Errorf("convert html to markdown: %w", err)
		}
		return markdownOut, nil
	}
	return content, nil
}
