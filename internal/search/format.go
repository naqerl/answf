package search

import (
	"fmt"
	"strings"
)

func formatSearchResults(results []result, showEngine bool) string {
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
