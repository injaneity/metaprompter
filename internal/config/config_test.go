package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `github:
  repo: acme/project
  labels:
    todo: todo
    in_progress: in_progress
    blocked: blocked
    done: done
frameworks:
  default: codex
  enabled:
    - codex
    - opencode
commands:
  source_dir: commands
  generated_dir: .metaprompter/bin
execute:
  retry_attempts: 1
  allow_subissue_fanout: true
checks:
  test_cmd: go test ./...
  lint_cmd: go vet ./...
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.GitHub.Repo != "acme/project" {
		t.Fatalf("unexpected repo: %s", cfg.GitHub.Repo)
	}
	if cfg.Frameworks.Default != "codex" {
		t.Fatalf("unexpected default framework: %s", cfg.Frameworks.Default)
	}
	if len(cfg.Frameworks.Enabled) != 2 {
		t.Fatalf("expected two enabled frameworks, got %d", len(cfg.Frameworks.Enabled))
	}
	if cfg.Execute.RetryAttempts != 1 {
		t.Fatalf("expected retry_attempts=1, got %d", cfg.Execute.RetryAttempts)
	}
	if got := filepath.Base(cfg.Commands.SourceDir); got != "commands" {
		t.Fatalf("unexpected source dir resolution: %s", cfg.Commands.SourceDir)
	}
}

func TestLoadAllowsMissingRepo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `github:
  labels:
    todo: todo
frameworks:
  default: codex
  enabled:
    - codex
commands:
  source_dir: commands
  generated_dir: .metaprompter/bin
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load without repo, got %v", err)
	}
	if cfg.GitHub.Repo != "" {
		t.Fatalf("expected empty repo, got %q", cfg.GitHub.Repo)
	}
}

func TestLoadOrDefaultMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".metaprompter", "config.yaml")
	cfg, found, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("load default: %v", err)
	}
	if found {
		t.Fatal("expected config file to be missing")
	}
	if cfg.Frameworks.Default != "codex" {
		t.Fatalf("unexpected default framework: %s", cfg.Frameworks.Default)
	}
	if len(cfg.Frameworks.Enabled) < 3 {
		t.Fatalf("expected built-in frameworks, got %v", cfg.Frameworks.Enabled)
	}
}
