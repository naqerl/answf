package search

import (
	"net/url"
	"sort"
	"strings"
)

func applyTop(results []result, top int) []result {
	if top <= 0 || top >= len(results) {
		return results
	}
	return results[:top]
}

func rankResults(results []result, query string) []result {
	type scoredResult struct {
		result result
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

	ranked := make([]result, 0, len(results))
	for _, item := range scored {
		ranked = append(ranked, item.result)
	}
	return ranked
}

func scoreSearchResult(r result, query string) int {
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
