package execute

import (
	"testing"
	"time"

	"github.com/zanechee/metaprompter/internal/config"
	"github.com/zanechee/metaprompter/internal/github"
)

func TestShouldCreateSubissue(t *testing.T) {
	if !shouldCreateSubissue("Blocked by dependency issue") {
		t.Fatal("expected dependency blocker to trigger subissue")
	}
	if shouldCreateSubissue("simple compile error") {
		t.Fatal("did not expect generic error to trigger subissue")
	}
}

func TestSelectFramework(t *testing.T) {
	issue := github.Issue{Labels: []github.IssueLabel{{Name: "framework:claude"}}}
	name := selectFramework(issue, "codex", []string{"codex", "claude"})
	if name != "claude" {
		t.Fatalf("expected claude, got %s", name)
	}
}

func TestFilterQueue(t *testing.T) {
	r := Runner{Config: &config.Config{GitHub: config.GitHubConfig{Labels: config.LabelConfig{Todo: "todo", InProgress: "in_progress", Blocked: "blocked", Done: "done"}}}, MaxIssues: 1}
	issues := []github.Issue{
		{Number: 1, CreatedAt: time.Now()},
		{Number: 2, CreatedAt: time.Now().Add(time.Minute), State: "CLOSED"},
	}
	queue := r.filterQueue(issues)
	if len(queue) != 1 {
		t.Fatalf("expected 1 issue in queue, got %d", len(queue))
	}
}

func TestRequiredFrameworks(t *testing.T) {
	out := requiredFrameworks([]string{"codex", "claude", "codex"}, "codex")
	if len(out) != 2 {
		t.Fatalf("expected 2 frameworks, got %v", out)
	}
	out = requiredFrameworks(nil, "opencode")
	if len(out) != 1 || out[0] != "opencode" {
		t.Fatalf("expected fallback default framework, got %v", out)
	}
}
