package artifacts

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type TaskItem struct {
	Number    int
	Title     string
	State     string
	CreatedAt time.Time
	URL       string
}

func BuildTasksMarkdown(items []TaskItem, generatedAt time.Time) string {
	sorted := make([]TaskItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].Number < sorted[j].Number
		}
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})

	var b strings.Builder
	b.WriteString("# Tasks Snapshot\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", generatedAt.UTC().Format(time.RFC3339)))
	for _, it := range sorted {
		created := "unknown"
		if !it.CreatedAt.IsZero() {
			created = it.CreatedAt.UTC().Format(time.RFC3339)
		}
		line := fmt.Sprintf("- #%d [%s] %s (created %s)", it.Number, it.State, it.Title, created)
		if strings.TrimSpace(it.URL) != "" {
			line += " - " + it.URL
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func WriteTasksSnapshot(path string, items []TaskItem, generatedAt time.Time) error {
	content := BuildTasksMarkdown(items, generatedAt)
	return os.WriteFile(path, []byte(content), 0o644)
}
