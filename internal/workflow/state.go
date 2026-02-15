package workflow

import (
	"github.com/zanechee/metaprompter/internal/config"
	"github.com/zanechee/metaprompter/internal/github"
)

func WorkflowLabels(labels config.LabelConfig) []string {
	return []string{labels.Todo, labels.InProgress, labels.Blocked, labels.Done}
}

func LabelForState(state string, labels config.LabelConfig) string {
	switch state {
	case "todo":
		return labels.Todo
	case "in_progress":
		return labels.InProgress
	case "blocked":
		return labels.Blocked
	case "done":
		return labels.Done
	default:
		return labels.Todo
	}
}

func StateFromIssue(issue github.Issue, labels config.LabelConfig) string {
	has := map[string]bool{}
	for _, l := range issue.Labels {
		has[l.Name] = true
	}

	switch {
	case has[labels.InProgress]:
		return "in_progress"
	case has[labels.Blocked]:
		return "blocked"
	case has[labels.Done]:
		return "done"
	case has[labels.Todo]:
		return "todo"
	default:
		if issue.State == "CLOSED" || issue.State == "closed" {
			return "done"
		}
		return "todo"
	}
}
