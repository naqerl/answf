package fetch

import (
	"fmt"

	"github.com/naqerl/answf/internal/netx"
	playwright "github.com/playwright-community/playwright-go"
)

func fetchWithPlaywright(cfg Config, target string) (string, bool, error) {
	wsEndpoint, err := netx.NormalizeWSEndpoint(cfg.WSEndpoint)
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
		Timeout:   playwright.Float(float64(cfg.Timeout.Milliseconds())),
	}); err != nil {
		return "", false, fmt.Errorf("navigate to %q: %w", target, err)
	}

	html, err := page.Content()
	if err != nil {
		return "", false, fmt.Errorf("read page content: %w", err)
	}

	return html, true, nil
}
