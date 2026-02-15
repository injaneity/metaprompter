package execute

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zanechee/metaprompter/internal/adapters"
	"github.com/zanechee/metaprompter/internal/artifacts"
	"github.com/zanechee/metaprompter/internal/config"
	"github.com/zanechee/metaprompter/internal/github"
	"github.com/zanechee/metaprompter/internal/prompts"
	"github.com/zanechee/metaprompter/internal/workflow"
)

type Runner struct {
	Config      *config.Config
	GitHub      github.Client
	ConfigPath  string
	DryRun      bool
	Issue       int
	MaxIssues   int
	ChildWorker bool
	Stdout      io.Writer
}

type runResult struct {
	Output string
	Err    error
}

func (r *Runner) Run(ctx context.Context) error {
	if r.Config == nil {
		return fmt.Errorf("config is required")
	}
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}
	if err := r.ensureAdapterScripts(); err != nil {
		return err
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

	tasks := make([]artifacts.TaskItem, 0, len(issues))
	for _, issue := range issues {
		tasks = append(tasks, artifacts.TaskItem{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     workflow.StateFromIssue(issue, r.Config.GitHub.Labels),
			CreatedAt: issue.CreatedAt,
			URL:       issue.URL,
		})
	}

	now := time.Now().UTC()
	if r.DryRun {
		fmt.Fprintln(r.Stdout, "[dry-run] would rebuild tasks.md and reset log.md")
	} else if !r.ChildWorker {
		if err := artifacts.WriteTasksSnapshot("tasks.md", tasks, now); err != nil {
			return fmt.Errorf("write tasks snapshot: %w", err)
		}
		if err := artifacts.ResetLog("log.md", now); err != nil {
			return fmt.Errorf("reset log: %w", err)
		}
	}

	queue := r.filterQueue(issues)
	if len(queue) == 0 {
		fmt.Fprintln(r.Stdout, "No eligible issues found for execution.")
		return nil
	}

	for _, issue := range queue {
		if err := r.processIssue(ctx, issue); err != nil {
			fmt.Fprintf(r.Stdout, "Issue #%d ended with error: %v\n", issue.Number, err)
		}
	}
	return nil
}

func (r *Runner) filterQueue(issues []github.Issue) []github.Issue {
	queue := make([]github.Issue, 0, len(issues))
	for _, issue := range issues {
		state := workflow.StateFromIssue(issue, r.Config.GitHub.Labels)
		if r.Issue > 0 && issue.Number != r.Issue {
			continue
		}
		if state == "done" {
			continue
		}
		queue = append(queue, issue)
	}
	if r.MaxIssues > 0 && len(queue) > r.MaxIssues {
		queue = queue[:r.MaxIssues]
	}
	return queue
}

func (r *Runner) processIssue(ctx context.Context, issue github.Issue) error {
	workflowLabels := workflow.WorkflowLabels(r.Config.GitHub.Labels)
	inProgress := workflow.LabelForState("in_progress", r.Config.GitHub.Labels)
	blocked := workflow.LabelForState("blocked", r.Config.GitHub.Labels)
	done := workflow.LabelForState("done", r.Config.GitHub.Labels)

	if !r.DryRun {
		if err := r.GitHub.SetWorkflowState(ctx, issue.Number, inProgress, workflowLabels, r.Config.GitHub.Labels.Done); err != nil {
			return fmt.Errorf("set issue #%d in_progress: %w", issue.Number, err)
		}
		if err := ensureIssueBranch(ctx, issue); err != nil {
			return fmt.Errorf("prepare branch for issue #%d: %w", issue.Number, err)
		}
	}
	if err := r.log(issue.Number, "START"); err != nil {
		return err
	}

	maxAttempts := r.Config.Execute.RetryAttempts + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastOutput string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err := r.log(issue.Number, "RETRY"); err != nil {
				return err
			}
		}

		result := r.runAgentForIssue(ctx, issue)
		lastOutput = result.Output
		if result.Err == nil {
			if err := r.runChecks(ctx, issue.Number); err != nil {
				lastOutput = lastOutput + "\nchecks failed: " + err.Error()
			} else {
				prURL, prErr := r.openPR(ctx, issue)
				if prErr != nil {
					lastOutput = lastOutput + "\npr failed: " + prErr.Error()
				} else {
					if err := r.log(issue.Number, fmt.Sprintf("PR_OPENED %s", prURL)); err != nil {
						return err
					}
					if !r.DryRun {
						if err := r.GitHub.SetWorkflowState(ctx, issue.Number, done, workflowLabels, r.Config.GitHub.Labels.Done); err != nil {
							return fmt.Errorf("set issue #%d done: %w", issue.Number, err)
						}
					}
					if err := r.log(issue.Number, "DONE"); err != nil {
						return err
					}
					return nil
				}
			}
		}

		reason := failureReason(result)
		if r.Config.Execute.AllowSubissueFanout && shouldCreateSubissue(reason) {
			subRef, subErr := r.createSubissue(ctx, issue, reason)
			if subErr == nil {
				_ = r.log(issue.Number, fmt.Sprintf("SUBISSUE_CREATED #%d", subRef.Number))
				r.spawnChildWorker(ctx, subRef.Number)
			}
		}

		if attempt == maxAttempts {
			if !r.DryRun {
				if err := r.GitHub.SetWorkflowState(ctx, issue.Number, blocked, workflowLabels, r.Config.GitHub.Labels.Done); err != nil {
					return fmt.Errorf("set issue #%d blocked: %w", issue.Number, err)
				}
			}
			followup, err := r.createBlockedFollowUp(ctx, issue, reason)
			if err == nil {
				_ = r.log(issue.Number, fmt.Sprintf("SUBISSUE_CREATED #%d", followup.Number))
			}
			if err := r.log(issue.Number, "BLOCKED"); err != nil {
				return err
			}
			return fmt.Errorf("issue #%d blocked after retries", issue.Number)
		}
	}

	return fmt.Errorf("issue #%d failed: %s", issue.Number, strings.TrimSpace(lastOutput))
}

func (r *Runner) runAgentForIssue(ctx context.Context, issue github.Issue) runResult {
	framework := selectFramework(issue, r.Config.Frameworks.Default, r.Config.Frameworks.Enabled)
	adapterPath := filepath.Join(r.Config.Commands.GeneratedDir, framework+".sh")

	tpl := prompts.LoadTemplateOrFallback("prompts/execute/worker.md", prompts.DefaultExecuteWorkerTemplate)
	rendered, err := prompts.RenderTemplate(tpl, map[string]string{
		"issue_number": strconv.Itoa(issue.Number),
		"issue_title":  issue.Title,
		"issue_body":   safe(issue.Body),
		"issue_url":    issue.URL,
		"framework":    framework,
	})
	if err != nil {
		return runResult{Err: err}
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("metaprompter-issue-%d-*.md", issue.Number))
	if err != nil {
		return runResult{Err: err}
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(rendered); err != nil {
		_ = tmpFile.Close()
		return runResult{Err: err}
	}
	if err := tmpFile.Close(); err != nil {
		return runResult{Err: err}
	}

	if r.DryRun {
		msg := fmt.Sprintf("[dry-run] would run %s for issue #%d with prompt file %s", framework, issue.Number, tmpFile.Name())
		fmt.Fprintln(r.Stdout, msg)
		return runResult{Output: msg}
	}
	if _, err := os.Stat(adapterPath); err != nil {
		return runResult{Err: fmt.Errorf("adapter script missing for %s at %s", framework, adapterPath)}
	}

	cmd := exec.CommandContext(ctx, adapterPath)
	cmd.Env = append(os.Environ(),
		"PROMPT_FILE="+tmpFile.Name(),
		"ISSUE_NUMBER="+strconv.Itoa(issue.Number),
		"ISSUE_TITLE="+issue.Title,
		"ISSUE_BODY="+safe(issue.Body),
		"ISSUE_URL="+issue.URL,
		"REPO_PATH="+mustGetwd(),
	)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	return runResult{Output: output.String(), Err: err}
}

func (r *Runner) runChecks(ctx context.Context, issueNumber int) error {
	if r.DryRun {
		fmt.Fprintf(r.Stdout, "[dry-run] would run checks for issue #%d\n", issueNumber)
		return nil
	}
	if strings.TrimSpace(r.Config.Checks.TestCmd) != "" {
		if err := runShell(ctx, r.Config.Checks.TestCmd); err != nil {
			_ = r.log(issueNumber, "CHECK_TEST_FAILED")
			return fmt.Errorf("test command failed: %w", err)
		}
	}
	if strings.TrimSpace(r.Config.Checks.LintCmd) != "" {
		if err := runShell(ctx, r.Config.Checks.LintCmd); err != nil {
			_ = r.log(issueNumber, "CHECK_LINT_FAILED")
			return fmt.Errorf("lint command failed: %w", err)
		}
	}
	return nil
}

func (r *Runner) openPR(ctx context.Context, issue github.Issue) (string, error) {
	branch := issueBranchName(issue)
	if r.DryRun {
		return "[dry-run]", nil
	}
	return r.GitHub.CreatePR(ctx, github.CreatePRRequest{
		Title: fmt.Sprintf("Issue #%d: %s", issue.Number, issue.Title),
		Body:  fmt.Sprintf("Resolves #%d", issue.Number),
		Head:  branch,
	})
}

func (r *Runner) createSubissue(ctx context.Context, parent github.Issue, reason string) (github.IssueRef, error) {
	if r.DryRun {
		return github.IssueRef{Number: 0, URL: ""}, nil
	}
	body := fmt.Sprintf("## Parent\n#%d\n\n## Reason\n%s\n", parent.Number, reason)
	return r.GitHub.CreateIssue(ctx, github.CreateIssueRequest{
		Title:  fmt.Sprintf("[Subtask] %s", parent.Title),
		Body:   body,
		Labels: []string{r.Config.GitHub.Labels.Todo},
	})
}

func (r *Runner) createBlockedFollowUp(ctx context.Context, parent github.Issue, reason string) (github.IssueRef, error) {
	if r.DryRun {
		return github.IssueRef{Number: 0, URL: ""}, nil
	}
	body := fmt.Sprintf("## Parent\n#%d\n\n## Blocker\n%s\n\n## Next steps\n- Resolve blocker\n- Resume parent issue\n", parent.Number, reason)
	return r.GitHub.CreateIssue(ctx, github.CreateIssueRequest{
		Title:  fmt.Sprintf("[Blocked Follow-up] %s", parent.Title),
		Body:   body,
		Labels: []string{r.Config.GitHub.Labels.Todo},
	})
}

func (r *Runner) spawnChildWorker(ctx context.Context, issueNumber int) {
	if !r.Config.Execute.AllowSubissueFanout || r.DryRun {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	args := []string{"execute", "--issue", strconv.Itoa(issueNumber), "--max-issues", "1", "--child-worker"}
	if strings.TrimSpace(r.ConfigPath) != "" {
		args = append(args, "--config", r.ConfigPath)
	}
	cmd := exec.CommandContext(context.Background(), exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
}

func (r *Runner) log(issueNumber int, action string) error {
	if r.DryRun {
		fmt.Fprintf(r.Stdout, "[dry-run] %s\n", artifacts.FormatLogEvent(time.Now().UTC(), issueNumber, action))
		return nil
	}
	return artifacts.AppendLogEvent("log.md", time.Now().UTC(), issueNumber, action)
}

func ensureIssueBranch(ctx context.Context, issue github.Issue) error {
	branch := issueBranchName(issue)
	if err := runExec(ctx, "git", "rev-parse", "--verify", branch); err == nil {
		return runExec(ctx, "git", "checkout", branch)
	}
	return runExec(ctx, "git", "checkout", "-b", branch)
}

func issueBranchName(issue github.Issue) string {
	slug := sanitizeSlug(issue.Title)
	return fmt.Sprintf("issue-%d-%s", issue.Number, slug)
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "task"
	}
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

func runShell(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runExec(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func failureReason(result runResult) string {
	if result.Err == nil {
		return ""
	}
	output := strings.TrimSpace(result.Output)
	if output == "" {
		return result.Err.Error()
	}
	return output
}

func shouldCreateSubissue(reason string) bool {
	reason = strings.ToLower(reason)
	keywords := []string{
		"blocked by",
		"dependency",
		"context rot",
		"lost context",
		"too large",
		"token",
	}
	for _, k := range keywords {
		if strings.Contains(reason, k) {
			return true
		}
	}
	return false
}

func selectFramework(issue github.Issue, defaultFramework string, enabled []string) string {
	enabledSet := map[string]bool{}
	for _, e := range enabled {
		enabledSet[e] = true
	}
	for _, l := range issue.Labels {
		if strings.HasPrefix(l.Name, "framework:") {
			candidate := strings.TrimPrefix(l.Name, "framework:")
			if enabledSet[candidate] {
				return candidate
			}
		}
	}
	if enabledSet[defaultFramework] {
		return defaultFramework
	}
	if len(enabled) > 0 {
		return enabled[0]
	}
	return "codex"
}

func safe(value string) string {
	return strings.ReplaceAll(value, "\x00", "")
}

func (r *Runner) ensureAdapterScripts() error {
	if r.DryRun {
		return nil
	}
	required := requiredFrameworks(r.Config.Frameworks.Enabled, r.Config.Frameworks.Default)
	if len(required) == 0 {
		return nil
	}
	missing := false
	for _, framework := range required {
		path := filepath.Join(r.Config.Commands.GeneratedDir, framework+".sh")
		if _, err := os.Stat(path); err != nil {
			missing = true
			break
		}
	}
	if !missing {
		return nil
	}

	defs, err := adapters.LoadDefinitions(r.Config.Commands.SourceDir)
	if err != nil {
		return err
	}
	filtered := make(map[string]adapters.Definition)
	for _, framework := range required {
		if def, ok := defs[framework]; ok {
			filtered[framework] = def
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("no adapter definitions available for frameworks: %s", strings.Join(required, ", "))
	}

	return adapters.GenerateScripts(filtered, r.Config.Commands.GeneratedDir)
}

func requiredFrameworks(enabled []string, defaultFramework string) []string {
	uniq := map[string]bool{}
	out := make([]string, 0, len(enabled)+1)
	for _, framework := range enabled {
		framework = strings.TrimSpace(framework)
		if framework == "" || uniq[framework] {
			continue
		}
		uniq[framework] = true
		out = append(out, framework)
	}
	defaultFramework = strings.TrimSpace(defaultFramework)
	if defaultFramework != "" && !uniq[defaultFramework] {
		out = append(out, defaultFramework)
	}
	return out
}
