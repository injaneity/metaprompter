package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client interface {
	ListIssues(ctx context.Context) ([]Issue, error)
	CreateIssue(ctx context.Context, req CreateIssueRequest) (IssueRef, error)
	SetWorkflowState(ctx context.Context, issueNumber int, stateLabel string, workflowLabels []string, doneLabel string) error
	AddComment(ctx context.Context, issueNumber int, body string) error
	CreatePR(ctx context.Context, req CreatePRRequest) (string, error)
}

type CLI struct {
	Repo string
}

type Issue struct {
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	State     string       `json:"state"`
	CreatedAt time.Time    `json:"createdAt"`
	Labels    []IssueLabel `json:"labels"`
	URL       string       `json:"url"`
	Body      string       `json:"body"`
}

type IssueLabel struct {
	Name string `json:"name"`
}

type CreateIssueRequest struct {
	Title  string
	Body   string
	Labels []string
}

type IssueRef struct {
	Number int
	URL    string
}

type CreatePRRequest struct {
	Title string
	Body  string
	Head  string
}

func NewCLI(repo string) *CLI {
	return &CLI{Repo: repo}
}

func (c *CLI) ListIssues(ctx context.Context) ([]Issue, error) {
	out, err := c.run(ctx, "issue", "list", "--state", "all", "--limit", "1000", "--json", "number,title,state,createdAt,labels,url,body")
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse issue list: %w", err)
	}
	return issues, nil
}

func (c *CLI) CreateIssue(ctx context.Context, req CreateIssueRequest) (IssueRef, error) {
	args := []string{"issue", "create", "--title", req.Title, "--body", req.Body}
	for _, l := range req.Labels {
		if strings.TrimSpace(l) == "" {
			continue
		}
		args = append(args, "--label", l)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return IssueRef{}, err
	}
	url := strings.TrimSpace(string(out))
	number := extractIssueNumber(url)
	if number == 0 {
		return IssueRef{}, fmt.Errorf("create issue returned unexpected output: %q", url)
	}
	return IssueRef{Number: number, URL: url}, nil
}

func (c *CLI) SetWorkflowState(ctx context.Context, issueNumber int, stateLabel string, workflowLabels []string, doneLabel string) error {
	args := []string{"issue", "edit", strconv.Itoa(issueNumber)}
	for _, label := range workflowLabels {
		if strings.TrimSpace(label) != "" {
			args = append(args, "--remove-label", label)
		}
	}
	if strings.TrimSpace(stateLabel) != "" {
		args = append(args, "--add-label", stateLabel)
	}
	if _, err := c.run(ctx, args...); err != nil {
		return err
	}

	if stateLabel == doneLabel {
		_, err := c.run(ctx, "issue", "close", strconv.Itoa(issueNumber))
		return err
	}

	// Ignore reopen errors so the label transition still works.
	_, _ = c.run(ctx, "issue", "reopen", strconv.Itoa(issueNumber))
	return nil
}

func (c *CLI) AddComment(ctx context.Context, issueNumber int, body string) error {
	_, err := c.run(ctx, "issue", "comment", strconv.Itoa(issueNumber), "--body", body)
	return err
}

func (c *CLI) CreatePR(ctx context.Context, req CreatePRRequest) (string, error) {
	args := []string{"pr", "create", "--title", req.Title, "--body", req.Body, "--head", req.Head}
	out, err := c.run(ctx, args...)
	if err != nil {
		return "", err
	}
	url := strings.TrimSpace(string(out))
	if !strings.HasPrefix(url, "http") {
		return "", fmt.Errorf("create PR returned unexpected output: %q", url)
	}
	return url, nil
}

func (c *CLI) run(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := make([]string, 0, len(args)+2)
	if strings.TrimSpace(c.Repo) != "" {
		fullArgs = append(fullArgs, "-R", c.Repo)
	}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "gh", fullArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

var issueNumberPattern = regexp.MustCompile(`/issues/(\d+)$`)

func extractIssueNumber(url string) int {
	m := issueNumberPattern.FindStringSubmatch(strings.TrimSpace(url))
	if len(m) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
