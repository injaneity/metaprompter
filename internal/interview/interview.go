package interview

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/zanechee/metaprompter/internal/artifacts"
	"github.com/zanechee/metaprompter/internal/config"
	"github.com/zanechee/metaprompter/internal/github"
	"github.com/zanechee/metaprompter/internal/prompts"
	"github.com/zanechee/metaprompter/internal/workflow"
)

type Runner struct {
	Config    *config.Config
	GitHub    github.Client
	DryRun    bool
	Framework string
	Stdout    io.Writer
}

func (r *Runner) Run(ctx context.Context) error {
	if r.Config == nil {
		return fmt.Errorf("config is required")
	}
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}

	framework := selectInterviewFramework(r.Framework, r.Config.Frameworks.Default, r.Config.Frameworks.Enabled)

	tpl := prompts.LoadTemplateOrFallback("prompts/interview/system.md", prompts.DefaultInterviewSystemTemplate)
	rendered, err := prompts.RenderTemplate(tpl, map[string]string{
		"framework":   framework,
		"repo_path":   mustGetwd(),
		"github_repo": r.Config.GitHub.Repo,
	})
	if err != nil {
		return fmt.Errorf("render interview template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "metaprompter-interview-*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(rendered); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	launchCmd, ok := interviewLaunchCommand(framework)
	if !ok {
		return fmt.Errorf("unsupported interview framework %q", framework)
	}
	if r.DryRun {
		fmt.Fprintf(r.Stdout, "[dry-run] no agent process will be started.\n")
		fmt.Fprintf(r.Stdout, "[dry-run] would run interview using framework=%s\n", framework)
		fmt.Fprintf(r.Stdout, "[dry-run] launch command: %s\n", launchCmd)
		fmt.Fprintf(r.Stdout, "[dry-run] prompt file: %s\n", tmpFile.Name())
		fmt.Fprintf(r.Stdout, "[dry-run] interview prompt preview:\n%s\n", truncate(rendered, 600))
		return nil
	}

	cmd := exec.CommandContext(ctx, "bash", "-lc", launchCmd)
	cmd.Env = append(os.Environ(),
		"PROMPT_FILE="+tmpFile.Name(),
		"REPO_PATH="+mustGetwd(),
		"GITHUB_REPO="+r.Config.GitHub.Repo,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("interview worker failed: %w", err)
	}

	if err := r.refreshTasksSnapshot(ctx); err != nil {
		fmt.Fprintf(r.Stdout, "warning: unable to rebuild tasks.md after interview: %v\n", err)
	}
	fmt.Fprintln(r.Stdout, "Interview run complete.")
	return nil
}

func (r *Runner) refreshTasksSnapshot(ctx context.Context) error {
	if strings.TrimSpace(r.Config.GitHub.Repo) == "" || r.GitHub == nil {
		return nil
	}
	issues, err := r.GitHub.ListIssues(ctx)
	if err != nil {
		return err
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].CreatedAt.Equal(issues[j].CreatedAt) {
			return issues[i].Number < issues[j].Number
		}
		return issues[i].CreatedAt.Before(issues[j].CreatedAt)
	})
	items := make([]artifacts.TaskItem, 0, len(issues))
	for _, issue := range issues {
		items = append(items, artifacts.TaskItem{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     workflow.StateFromIssue(issue, r.Config.GitHub.Labels),
			CreatedAt: issue.CreatedAt,
			URL:       issue.URL,
		})
	}
	return artifacts.WriteTasksSnapshot("tasks.md", items, time.Now().UTC())
}

func selectInterviewFramework(override string, defaultFramework string, enabled []string) string {
	override = strings.TrimSpace(override)
	if override != "" {
		for _, e := range enabled {
			if e == override {
				return override
			}
		}
	}
	for _, e := range enabled {
		if e == defaultFramework {
			return defaultFramework
		}
	}
	if len(enabled) > 0 {
		return enabled[0]
	}
	if override != "" {
		return override
	}
	if defaultFramework != "" {
		return defaultFramework
	}
	return "codex"
}

func interviewLaunchCommand(framework string) (string, bool) {
	switch strings.TrimSpace(framework) {
	case "codex":
		return `codex "$(cat "$PROMPT_FILE")"`, true
	case "claude":
		return `claude "$(cat "$PROMPT_FILE")"`, true
	case "opencode":
		return `opencode --prompt "$(cat "$PROMPT_FILE")"`, true
	default:
		return "", false
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func truncate(value string, max int) string {
	if max < 4 || len(value) <= max {
		return value
	}
	return value[:max-3] + "..."
}
