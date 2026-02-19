# Metaprompter — Overview

## What It Is

Metaprompter is a specs-first metaprompting framework for agentic coding workflows. It is a collection of prompts and skills that run *inside* AI agents (Claude Code, OpenCode, Codex) rather than a binary that spawns them.

The AI is the orchestrator. It uses its own native tools (Bash, Read, Write, Edit) to drive the full workflow: interviewing users, writing specs, creating GitHub issues, implementing them, and opening pull requests. No binary. No adapter scripts.

## Two-Phase Model

### Phase 1: Interview (`/interview`)
Elicit requirements through structured dialogue. Produce a spec pack in `docs/spec/`. Create GitHub issues with dependency links. This is the **planning** phase.

### Phase 2: Execute (`/execute`)
Work through the GitHub issue backlog. For each issue: branch, implement, run checks, open a PR. Track progress via an append-only event log in issue comments. This is the **implementation** phase.

## Primary Interface (Claude Code)

Claude Code skills in `.claude/commands/` are the primary interface. They are invoked as slash commands within a Claude Code session:

| Command | Description |
|---------|-------------|
| `/interview` | Elicit requirements, write spec files, create GitHub issues |
| `/execute [--issue N] [--max N]` | Implement issues from the backlog |
| `/status` | Dashboard: issue states, circuit breaker, PRs, event log |

## Secondary Interfaces (OpenCode / Codex)

For environments without Claude Code, equivalent prompts in `prompts/opencode/` and `prompts/codex/` provide the same workflow via:

```bash
# OpenCode
opencode run "$(cat prompts/opencode/interview.md)"
opencode run "$(cat prompts/opencode/execute.md)"

# Codex
codex exec - < prompts/codex/interview.md
codex exec - < prompts/codex/execute.md
```

Env var overrides (no `$ARGUMENTS` support): `METAPROMPTER_ISSUE=N`, `METAPROMPTER_MAX=N`.

## Requirements

1. A git repository connected to GitHub (`origin` remote).
2. `gh` CLI authenticated — used for all GitHub operations across all three agent environments.
3. One of: Claude Code (`claude`), OpenCode (`opencode`), or Codex (`codex`).

## Key Properties

- **Specs-first**: No implementation until requirements are written to `docs/spec/`.
- **GitHub as state machine**: Issues are tasks; labels are states; comments are the event log.
- **Human touchpoint**: Only reviewing and merging PRs. Everything else is automated.
- **`awaiting_review` label**: Cleanly separates "agent done" from "human confirmed done (PR merged)".
- **Circuit breaker**: Consecutive failure tracking → auto-halt to prevent runaway spend.
