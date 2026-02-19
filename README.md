# metaprompter

Specs-first metaprompting framework for agentic coding workflows.

The AI is the orchestrator. Metaprompter is a collection of prompts and skills that run *inside* AI agents — not a binary that spawns them.

## How It Works

**1. Interview** — Elicit requirements, write spec files, create GitHub issues.

**2. Execute** — Work through the issue backlog: branch, implement, run checks, open PRs.

**3. Automate** — GitHub Actions merges PRs, marks issues done, and unblocks dependents. The only human touchpoint is reviewing and merging PRs.

## Quick Start

```bash
# Prerequisites: git repo with GitHub origin, gh CLI authenticated, claude/opencode/codex installed

# Phase 1: Plan
/interview

# Phase 2: Implement
/execute

# Check progress
/status
```

## Primary Interface (Claude Code)

| Command | Description |
|---------|-------------|
| `/interview` | Elicit requirements, write `docs/spec/`, create GitHub issues |
| `/execute [--issue N] [--max N]` | Implement issues from the backlog |
| `/status` | Dashboard: issue states, circuit breaker, PRs, event log |

Skills live in `.claude/commands/`. They use Claude Code's native tools (Read, Write, Edit, Bash) — no adapter scripts needed.

## Secondary Interface (OpenCode / Codex)

```bash
# OpenCode
opencode run "$(cat prompts/opencode/interview.md)"
opencode run "$(cat prompts/opencode/execute.md)"

# Codex
codex exec - < prompts/codex/interview.md
codex exec - < prompts/codex/execute.md
```

Env var overrides (no `$ARGUMENTS` support): `METAPROMPTER_ISSUE=N`, `METAPROMPTER_MAX=N`.

## State Machine

```
[created] ──gh issue create──► [todo]
    │
    └──/execute picks up──► [in_progress]
                                  │
                    ┌─────────────┴──────────────┐
                    │                            │
                 success                      failure
                    │                            │
                    ▼                            ▼
            [awaiting_review]              retry? ──yes──► [in_progress]
                    │                            │
             PR merged                          no
      (Actions workflow)                        │
                    │                            ▼
                    ▼                        [blocked]
                 [done]                          │
                                     circuit breaker threshold?
                                                 │
                                              tripped=true → HALT
```

## Dependency Tracking

Add `depends on #N` to an issue body. `/execute` parses dependencies, applies topological sort (Kahn's algorithm), and only executes issues whose dependencies are resolved. When a PR merges, the Actions workflow unblocks dependent issues automatically.

```
# Issue body example
Implement the user profile page.

depends on #12, #15
```

## Topic Routing

Labels route issues to specialized worker prompts. Configure in `.metaprompter/config.yaml`:

```yaml
routing:
  frontend:
    prompt: prompts/execute/worker-frontend.md
  bug:
    prompt: prompts/execute/worker-bugfix.md
```

First matching label wins. No match → generic `prompts/execute/worker.md`.

## Circuit Breaker

Consecutive failure tracking prevents runaway spend. Configure threshold in `.metaprompter/config.yaml`:

```yaml
circuit_breaker:
  failure_threshold: 3
```

After N consecutive exhausted-retry failures: execution halts, a GitHub issue is created describing the halt. To resume: investigate blocked issues, fix root causes, reset `.metaprompter/state.json`.

## Event Log

All state transitions are recorded as HTML comment blocks in issue comments — hidden in the GitHub UI, machine-readable via API. Format:

```
[metaprompter] START — issue #42

<!-- metaprompter-event
type: START
issue: 42
timestamp: 2024-01-15T10:00:00Z
branch: issue-42-dark-mode-toggle
framework: claude
-->
```

Event types: `START`, `HEARTBEAT`, `RETRY`, `BLOCKED`, `PR_OPENED`, `DONE`, `UNBLOCKED`.

## Requirements

1. Git repository connected to GitHub (`origin` remote).
2. `gh` CLI authenticated — used for all GitHub operations across all agent environments.
3. One of: Claude Code (`claude`), OpenCode (`opencode`), or Codex (`codex`).

## Configuration

`.metaprompter/config.yaml` — full schema in [`docs/spec/data-model.md`](docs/spec/data-model.md).

```yaml
github:
  repo:                          # auto-detected from git remote origin
  labels:
    todo: todo
    in_progress: in_progress
    blocked: blocked
    done: done
    awaiting_review: awaiting_review

routing:
  frontend:
    prompt: prompts/execute/worker-frontend.md
  bug:
    prompt: prompts/execute/worker-bugfix.md

frameworks:
  default: claude
  enabled: [claude, codex, opencode]

execute:
  max_concurrent: 1
  retry_attempts: 1
  allow_subissue_fanout: true

circuit_breaker:
  failure_threshold: 3

checks:
  test_cmd:
  lint_cmd:
```

## Key Artifacts

| Path | Description |
|------|-------------|
| `docs/spec/*.md` | Canonical spec pack (written by `/interview`) |
| `docs/modules/*.md` | Per-module design docs for complex modules |
| `tasks.md` | Snapshot of GitHub issue state (rebuilt each run) |
| `.metaprompter/state.json` | Runtime state: circuit breaker counters |
| `result-N.json` | Per-issue audit trail (written on issue branch, merges with PR) |
| `.claude/commands/` | Claude Code skills |
| `prompts/execute/` | Worker prompt templates |
| `prompts/opencode/` | OpenCode equivalents |
| `prompts/codex/` | Codex equivalents |
| `.github/workflows/pr-merged.yml` | PRWatcher: auto-closes issues, unblocks deps on merge |
