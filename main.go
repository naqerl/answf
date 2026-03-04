package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	playwright "github.com/playwright-community/playwright-go"
)

func main() {
	cfg := parseFlags()

	html, err := renderHTML(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(html)
}

type config struct {
	FetchURL   string
	TargetURL  string
	Markdown   bool
	WSEndpoint string
	TimeoutMS  float64
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.FetchURL, "fetch", "", "Fetch and render content from URL")
	flag.BoolVar(&cfg.Markdown, "md", false, "Output markdown instead of raw HTML")
	flag.StringVar(&cfg.WSEndpoint, "ws-endpoint", firstNonEmpty(os.Getenv("BROWSERLESS_WS_ENDPOINT"), "wss://browserless.aishift.co"), "Browserless websocket endpoint")
	flag.Float64Var(&cfg.TimeoutMS, "timeout-ms", 30000, "Navigation timeout in milliseconds")
	flag.Parse()

	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: only one positional URL argument is supported")
		os.Exit(2)
	}

	cfg.FetchURL = strings.TrimSpace(cfg.FetchURL)
	switch {
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
