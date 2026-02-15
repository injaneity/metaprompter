package artifacts

import (
	"strings"
	"testing"
	"time"
)

func TestFormatLogEvent(t *testing.T) {
	ts := time.Date(2026, 2, 15, 10, 30, 45, 0, time.UTC)
	line := FormatLogEvent(ts, 17, "START")
	expected := "2026-02-15T10:30:45Z [17] START"
	if line != expected {
		t.Fatalf("expected %q, got %q", expected, line)
	}
}

func TestBuildTasksMarkdownSorted(t *testing.T) {
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	content := BuildTasksMarkdown([]TaskItem{
		{Number: 2, Title: "Second", State: "todo", CreatedAt: now.Add(time.Hour)},
		{Number: 1, Title: "First", State: "done", CreatedAt: now},
	}, now)

	firstIndex := strings.Index(content, "#1")
	secondIndex := strings.Index(content, "#2")
	if firstIndex == -1 || secondIndex == -1 || firstIndex > secondIndex {
		t.Fatalf("tasks not sorted by created time:\n%s", content)
	}
}
