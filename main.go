package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	playwright "github.com/playwright-community/playwright-go"
)

func main() {
	cfg := parseFlags()

	out, err := run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(out)
}

type config struct {
	FetchURL        string
	Search          string
	TargetURL       string
	Markdown        bool
	WSEndpoint      string
	SearXURL        string
	TimeoutMS       float64
	FallbackTextise bool
	TextiseBaseURL  string
	Verbose         bool
	Top             int
	CacheDir        string
	NoCache         bool
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.FetchURL, "fetch", "", "Fetch and render content from URL")
	flag.StringVar(&cfg.Search, "search", "", "Search query to run against SearXNG and print results")
	flag.StringVar(&cfg.Search, "s", "", "Alias for -search")
	flag.BoolVar(&cfg.Markdown, "md", false, "Output markdown instead of raw HTML")
	flag.StringVar(&cfg.WSEndpoint, "ws-endpoint", firstNonEmpty(os.Getenv("BROWSERLESS_WS_ENDPOINT"), "wss://browserless.aishift.co"), "Browserless websocket endpoint")
	flag.StringVar(&cfg.SearXURL, "searx-url", firstNonEmpty(os.Getenv("SEARX_URL"), "https://searx.aishift.co"), "SearXNG base URL")
	flag.Float64Var(&cfg.TimeoutMS, "timeout-ms", 30000, "Navigation timeout in milliseconds")
	flag.BoolVar(&cfg.FallbackTextise, "fallback-textise", true, "Fallback to textise endpoint when browser fetch fails")
	flag.StringVar(&cfg.TextiseBaseURL, "textise-base-url", "https://r.jina.ai/http://", "Textise fallback base URL")
	flag.BoolVar(&cfg.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	flag.IntVar(&cfg.Top, "top", 0, "Limit search results to top N (0 means all)")
	flag.StringVar(&cfg.CacheDir, "cache-dir", defaultCacheDir(), "Cache directory")
	flag.BoolVar(&cfg.NoCache, "no-cache", false, "Disable local cache reads/writes")
	flag.Parse()

	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: only one positional argument is supported")
		os.Exit(2)
	}

	cfg.FetchURL = strings.TrimSpace(cfg.FetchURL)
	cfg.Search = strings.TrimSpace(cfg.Search)
	if cfg.Top < 0 {
		fmt.Fprintln(os.Stderr, "error: -top must be >= 0")
		os.Exit(2)
	}

	expandedCacheDir, err := expandPath(cfg.CacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid cache-dir: %v\n", err)
		os.Exit(2)
	}
	cfg.CacheDir = expandedCacheDir

	switch {
	case cfg.FetchURL != "" && cfg.Search != "":
		fmt.Fprintln(os.Stderr, "error: use either -fetch or -search/-s, not both")
		os.Exit(2)
	case cfg.Search != "":
		if len(args) > 0 {
			fmt.Fprintln(os.Stderr, "error: positional argument is not supported with -search/-s")
			os.Exit(2)
		}
	case cfg.FetchURL != "" && len(args) > 0:
		fmt.Fprintln(os.Stderr, "error: provide URL either via -fetch or as positional argument, not both")
		os.Exit(2)
	case cfg.FetchURL != "":
		cfg.TargetURL = cfg.FetchURL
	case len(args) == 1:
		cfg.TargetURL = strings.TrimSpace(args[0])
	default:
		cfg.TargetURL = "https://google.com/search?q=helloworld"
	}

	return cfg
}

func run(cfg config) (string, error) {
	cache := cacheManager{
		dir:      cfg.CacheDir,
		disabled: cfg.NoCache,
		now:      time.Now,
	}

	if cfg.Search != "" {
		return runSearch(cfg, cache)
	}

	return fetchWithFallback(cfg, cache)
}

func fetchWithFallback(cfg config, cache cacheManager) (string, error) {
	target, err := normalizeHTTPURL(cfg.TargetURL)
	if err != nil {
		return "", err
	}

	cacheKey := keyForFetch(target, cfg.Markdown)
	if cached, ok, err := cache.Get(cacheKey, 24*time.Hour); err == nil && ok {
		return cached, nil
	} else if err != nil {
		return "", fmt.Errorf("read fetch cache: %w", err)
	}

	content, isHTML, err := fetchWithPlaywright(cfg, target)
	if err != nil {
		if !cfg.FallbackTextise {
			return "", err
		}
		fallbackContent, fallbackErr := fetchViaTextise(target, time.Duration(cfg.TimeoutMS)*time.Millisecond, cfg.TextiseBaseURL)
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
	if err := cache.Set(cacheKey, output); err != nil {
		return "", fmt.Errorf("write fetch cache: %w", err)
	}
	return output, nil
}

func fetchWithPlaywright(cfg config, target string) (string, bool, error) {
	wsEndpoint, err := normalizeWSEndpoint(cfg.WSEndpoint)
	if err != nil {
		return "", false, err
	}

	if err := playwright.Install(&playwright.RunOptions{
		SkipInstallBrowsers: true,
		Verbose:             false,
	}); err != nil {
		return "", false, fmt.Errorf("install playwright driver: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return "", false, fmt.Errorf("start playwright: %w", err)
	}
	defer func() {
		_ = pw.Stop()
	}()

	browser, err := pw.Chromium.ConnectOverCDP(wsEndpoint)
	if err != nil {
		return "", false, fmt.Errorf("connect to browserless endpoint %q: %w", wsEndpoint, err)
	}
	defer func() {
		_ = browser.Close()
	}()

	contexts := browser.Contexts()
	var context playwright.BrowserContext
	if len(contexts) > 0 {
		context = contexts[0]
	} else {
		context, err = browser.NewContext()
		if err != nil {
			return "", false, fmt.Errorf("create context: %w", err)
		}
	}

	page, err := context.NewPage()
	if err != nil {
		return "", false, fmt.Errorf("create page: %w", err)
	}
	defer func() {
		_ = page.Close()
	}()

	if _, err := page.Goto(target, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(cfg.TimeoutMS),
	}); err != nil {
		return "", false, fmt.Errorf("navigate to %q: %w", target, err)
	}

	html, err := page.Content()
	if err != nil {
		return "", false, fmt.Errorf("read page content: %w", err)
	}

	return html, true, nil
}

func fetchViaTextise(targetURL string, timeout time.Duration, textiseBase string) (string, error) {
	target := buildTextiseURL(textiseBase, targetURL)
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", fmt.Errorf("build textise request: %w", err)
	}
	req.Header.Set("Accept", "text/plain,text/markdown;q=0.9,text/html;q=0.8,*/*;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform textise request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("textise request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read textise response: %w", err)
	}
	return string(body), nil
}

func buildTextiseURL(base, target string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "https://r.jina.ai/http://"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	return base + target
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

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

func runSearch(cfg config, cache cacheManager) (string, error) {
	baseURL, err := normalizeHTTPURL(cfg.SearXURL)
	if err != nil {
		return "", fmt.Errorf("invalid searx-url: %w", err)
	}

	cacheQuery := fmt.Sprintf("%s|top=%d|verbose=%t", cfg.Search, cfg.Top, cfg.Verbose)
	cacheKey := keyForSearch(cacheQuery, baseURL)
	if cached, ok, err := cache.Get(cacheKey, time.Hour); err == nil && ok {
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
	values.Set("q", cfg.Search)
	searchURL.RawQuery = values.Encode()

	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	client := &http.Client{Timeout: timeout}
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

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode searx response: %w", err)
	}

	ranked := rankResults(parsed.Results, cfg.Search)
	limited := applyTop(ranked, cfg.Top)
	out := formatSearchResults(limited, cfg.Verbose)
	if err := cache.Set(cacheKey, out); err != nil {
		return "", fmt.Errorf("write search cache: %w", err)
	}
	return out, nil
}

func applyTop(results []searchResult, top int) []searchResult {
	if top <= 0 || top >= len(results) {
		return results
	}
	return results[:top]
}

func rankResults(results []searchResult, query string) []searchResult {
	type scoredResult struct {
		result searchResult
		score  int
		index  int
	}
	scored := make([]scoredResult, 0, len(results))
	for i, r := range results {
		scored = append(scored, scoredResult{
			result: r,
			score:  scoreSearchResult(r, query),
			index:  i,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].index < scored[j].index
	})

	ranked := make([]searchResult, 0, len(results))
	for _, item := range scored {
		ranked = append(ranked, item.result)
	}
	return ranked
}

func scoreSearchResult(r searchResult, query string) int {
	score := 0
	host := normalizedResultHost(r.URL)
	title := strings.ToLower(strings.TrimSpace(r.Title))
	full := host + " " + title

	switch {
	case strings.Contains(host, "stackoverflow.com"), strings.Contains(host, "stackexchange.com"):
		score += 25
	case strings.Contains(host, "github.com"):
		score += 10
	}

	docsSignals := []string{"docs.", "wiki.", "readthedocs.io", "developer."}
	for _, signal := range docsSignals {
		if strings.Contains(host, signal) {
			score += 40
			break
		}
	}

	lowSignalHosts := []string{
		"pinterest.",
		"quora.com",
		"fandom.com",
		"medium.com",
	}
	for _, signal := range lowSignalHosts {
		if strings.Contains(host, signal) {
			score -= 20
			break
		}
	}

	for _, token := range strings.Fields(strings.ToLower(query)) {
		if token != "" && strings.Contains(full, token) {
			score += 5
		}
	}

	return score
}

func normalizedResultHost(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(rawURL))
	}
	return strings.ToLower(u.Hostname())
}

func formatSearchResults(results []searchResult, showEngine bool) string {
	if len(results) == 0 {
		return "No results\n"
	}

	var out strings.Builder
	for i, r := range results {
		if i > 0 {
			out.WriteString("\n")
		}
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = "(untitled)"
		}
		out.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
		out.WriteString(strings.TrimSpace(r.URL))
		out.WriteString("\n")

		content := strings.TrimSpace(r.Content)
		if content != "" {
			out.WriteString(content)
			out.WriteString("\n")
		}

		engine := strings.TrimSpace(r.Engine)
		if showEngine && engine != "" {
			out.WriteString("engine: ")
			out.WriteString(engine)
			out.WriteString("\n")
		}
	}

	return out.String()
}

func normalizeHTTPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url is required")
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid url %q: %w", raw, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("url scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return "", fmt.Errorf("url host is required: %q", raw)
	}

	return u.String(), nil
}

func normalizeWSEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("ws-endpoint is required")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid ws-endpoint %q: %w", raw, err)
	}

	switch u.Scheme {
	case "ws", "wss":
		// valid
	case "http":
		u.Scheme = "ws"
	case "https", "":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("ws-endpoint scheme must be ws/wss/http/https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return "", fmt.Errorf("ws-endpoint host is required: %q", raw)
	}

	return u.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
