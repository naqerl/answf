package fetch

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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
