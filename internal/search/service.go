package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/naqerl/answf/internal/cache"
	"github.com/naqerl/answf/internal/netx"
)

type Config struct {
	Query    string
	SearXURL string
	Timeout  time.Duration
	Verbose  bool
	Top      int
}

type response struct {
	Results []result `json:"results"`
}

type result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

func Run(cfg Config, c cache.Manager) (string, error) {
	baseURL, err := netx.NormalizeHTTPURL(cfg.SearXURL)
	if err != nil {
		return "", fmt.Errorf("invalid searx-url: %w", err)
	}

	cacheQuery := fmt.Sprintf("%s|top=%d|verbose=%t", cfg.Query, cfg.Top, cfg.Verbose)
	cacheKey := cache.KeyForSearch(cacheQuery, baseURL)
	if cached, ok, err := c.Get(cacheKey, time.Hour); err == nil && ok {
		return cached, nil
	} else if err != nil {
		return "", fmt.Errorf("read search cache: %w", err)
	}

	searchURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse searx-url: %w", err)
	}
	searchURL.Path = "/search"
	values := searchURL.Query()
	values.Set("format", "json")
	values.Set("q", cfg.Query)
	searchURL.RawQuery = values.Encode()

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest(http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform search request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("searx request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode searx response: %w", err)
	}

	ranked := rankResults(parsed.Results, cfg.Query)
	limited := applyTop(ranked, cfg.Top)
	out := formatSearchResults(limited, cfg.Verbose)
	if err := c.Set(cacheKey, out); err != nil {
		return "", fmt.Errorf("write search cache: %w", err)
	}
	return out, nil
}
