package main

import (
	"strings"
	"testing"
)

func TestNormalizeHTTPURL(t *testing.T) {
	t.Parallel()

	got, err := normalizeHTTPURL("example.com/a")
	if err != nil {
		t.Fatalf("normalizeHTTPURL returned error: %v", err)
	}
	if got != "https://example.com/a" {
		t.Fatalf("unexpected URL: got %q", got)
	}
}

func TestNormalizeWSEndpoint(t *testing.T) {
	t.Parallel()

	got, err := normalizeWSEndpoint("https://browserless.example")
	if err != nil {
		t.Fatalf("normalizeWSEndpoint returned error: %v", err)
	}
	if got != "wss://browserless.example" {
		t.Fatalf("unexpected endpoint: got %q", got)
	}
}

func TestFormatSearchResultsHideEngine(t *testing.T) {
	t.Parallel()

	out := formatSearchResults([]searchResult{
		{
			Title:   "Systemd Sandbox",
			URL:     "https://wiki.archlinux.org/title/Systemd/Sandboxing",
			Content: "content",
			Engine:  "startpage",
		},
	}, false)
	if strings.Contains(out, "engine:") {
		t.Fatalf("engine line should be hidden, got %q", out)
	}
}

func TestFormatSearchResultsShowEngine(t *testing.T) {
	t.Parallel()

	out := formatSearchResults([]searchResult{
		{
			Title:   "Systemd Sandbox",
			URL:     "https://wiki.archlinux.org/title/Systemd/Sandboxing",
			Content: "content",
			Engine:  "startpage",
		},
	}, true)
	if !strings.Contains(out, "engine: startpage") {
		t.Fatalf("engine line should be visible, got %q", out)
	}
}

func TestRankResults(t *testing.T) {
	t.Parallel()

	in := []searchResult{
		{
			Title: "A random blog about sandboxing",
			URL:   "https://medium.com/some-post",
		},
		{
			Title: "Systemd Sandboxing - ArchWiki",
			URL:   "https://wiki.archlinux.org/title/Systemd/Sandboxing",
		},
	}

	ranked := rankResults(in, "systemd sandboxing")
	if len(ranked) != 2 {
		t.Fatalf("unexpected ranked result len: %d", len(ranked))
	}
	if ranked[0].URL != "https://wiki.archlinux.org/title/Systemd/Sandboxing" {
		t.Fatalf("expected wiki result first, got %q", ranked[0].URL)
	}
}

func TestApplyTop(t *testing.T) {
	t.Parallel()

	in := []searchResult{{Title: "1"}, {Title: "2"}, {Title: "3"}}

	all := applyTop(in, 0)
	if len(all) != 3 {
		t.Fatalf("top=0 should keep all, got %d", len(all))
	}

	one := applyTop(in, 1)
	if len(one) != 1 {
		t.Fatalf("top=1 should keep one, got %d", len(one))
	}

	big := applyTop(in, 10)
	if len(big) != 3 {
		t.Fatalf("top > len should keep all, got %d", len(big))
	}
}
