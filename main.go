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
	FetchURL   string
	Search     string
	TargetURL  string
	Markdown   bool
	WSEndpoint string
	SearXURL   string
	TimeoutMS  float64
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
	flag.Parse()

	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: only one positional argument is supported")
		os.Exit(2)
	}

	cfg.FetchURL = strings.TrimSpace(cfg.FetchURL)
	cfg.Search = strings.TrimSpace(cfg.Search)
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
	if cfg.Search != "" {
		return runSearch(cfg)
	}

	return renderHTML(cfg)
}

func renderHTML(cfg config) (string, error) {
	target, err := normalizeHTTPURL(cfg.TargetURL)
	if err != nil {
		return "", err
	}

	wsEndpoint, err := normalizeWSEndpoint(cfg.WSEndpoint)
	if err != nil {
		return "", err
	}

	if err := playwright.Install(&playwright.RunOptions{
		SkipInstallBrowsers: true,
		Verbose:             false,
	}); err != nil {
		return "", fmt.Errorf("install playwright driver: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("start playwright: %w", err)
	}
	defer func() {
		_ = pw.Stop()
	}()

	browser, err := pw.Chromium.ConnectOverCDP(wsEndpoint)
	if err != nil {
		return "", fmt.Errorf("connect to browserless endpoint %q: %w", wsEndpoint, err)
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
			return "", fmt.Errorf("create context: %w", err)
		}
	}

	page, err := context.NewPage()
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer func() {
		_ = page.Close()
	}()

	if _, err := page.Goto(target, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(cfg.TimeoutMS),
	}); err != nil {
		return "", fmt.Errorf("navigate to %q: %w", target, err)
	}

	html, err := page.Content()
	if err != nil {
		return "", fmt.Errorf("read page content: %w", err)
	}

	if cfg.Markdown {
		markdown, err := htmltomarkdown.ConvertString(html)
		if err != nil {
			return "", fmt.Errorf("convert html to markdown: %w", err)
		}
		return markdown, nil
	}

	return html, nil
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

func runSearch(cfg config) (string, error) {
	baseURL, err := normalizeHTTPURL(cfg.SearXURL)
	if err != nil {
		return "", fmt.Errorf("invalid searx-url: %w", err)
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

	return formatSearchResults(parsed.Results), nil
}

func formatSearchResults(results []searchResult) string {
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
		if engine != "" {
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
