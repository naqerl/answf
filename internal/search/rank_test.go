package search

import "testing"

func TestRankResults(t *testing.T) {
	t.Parallel()

	in := []result{
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

	in := []result{{Title: "1"}, {Title: "2"}, {Title: "3"}}

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
