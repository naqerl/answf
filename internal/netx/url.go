package netx

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func NormalizeHTTPURL(raw string) (string, error) {
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

func NormalizeWSEndpoint(raw string) (string, error) {
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
