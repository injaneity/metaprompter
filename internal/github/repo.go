package github

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

func DetectRepoFromGitRemote(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	repo, err := ParseGitHubRepo(strings.TrimSpace(stdout.String()))
	if err != nil {
		return "", err
	}
	return repo, nil
}

func ParseGitHubRepo(remoteURL string) (string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", fmt.Errorf("empty remote URL")
	}

	if strings.HasPrefix(remoteURL, "git@github.com:") {
		path := strings.TrimPrefix(remoteURL, "git@github.com:")
		return normalizeRepoPath(path)
	}
	if strings.HasPrefix(remoteURL, "ssh://git@github.com/") {
		path := strings.TrimPrefix(remoteURL, "ssh://git@github.com/")
		return normalizeRepoPath(path)
	}
	if strings.HasPrefix(remoteURL, "https://github.com/") || strings.HasPrefix(remoteURL, "http://github.com/") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", fmt.Errorf("parse remote URL: %w", err)
		}
		if !strings.EqualFold(u.Hostname(), "github.com") {
			return "", fmt.Errorf("unsupported host %q", u.Hostname())
		}
		return normalizeRepoPath(strings.TrimPrefix(u.Path, "/"))
	}

	// Fallback for uncommon URL styles that still contain github.com/owner/repo
	idx := strings.Index(remoteURL, "github.com/")
	if idx >= 0 {
		path := remoteURL[idx+len("github.com/"):]
		return normalizeRepoPath(path)
	}

	return "", fmt.Errorf("remote URL does not point to github.com")
}

func normalizeRepoPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub repo path %q", path)
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", fmt.Errorf("invalid owner/repo in %q", path)
	}
	return owner + "/" + repo, nil
}
