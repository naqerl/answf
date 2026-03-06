package search

import (
	"strings"
	"testing"
)

func TestFormatSearchResultsHideEngine(t *testing.T) {
	t.Parallel()

	out := formatSearchResults([]result{
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

	out := formatSearchResults([]result{
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
