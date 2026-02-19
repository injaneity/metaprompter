# Metaprompter — Data Model

## config.yaml (`.metaprompter/config.yaml`)

```yaml
github:
  repo: owner/repo          # Optional — auto-detected from git remote origin
  labels:
    todo: todo
    in_progress: in_progress
    blocked: blocked
    done: done
    awaiting_review: awaiting_review

routing:                    # label → prompt template mapping
  <label-name>:
    prompt: <path-to-template.md>
  # Example:
  frontend:
    prompt: prompts/execute/worker-frontend.md
  bug:
    prompt: prompts/execute/worker-bugfix.md

frameworks:
  default: claude           # Which framework to use by default
  enabled:
    - claude
    - codex
    - opencode

execute:
  max_concurrent: 1         # Serial execution (>1 not yet supported)
  retry_attempts: 1         # Retries per issue before marking blocked
  allow_subissue_fanout: true

circuit_breaker:
  failure_threshold: 3      # Consecutive failures before tripping

checks:
  test_cmd:                 # Optional — run after implementation (e.g. "npm test")
  lint_cmd:                 # Optional — run after implementation (e.g. "npm run lint")
```

## state.json (`.metaprompter/state.json`)

Runtime state — tracks circuit breaker across sessions.

```json
{
  "consecutive_failures": 0,
  "tripped": false,
  "last_updated": "2024-01-15T10:30:00Z",
  "last_failure_issue": null
}
```

Fields:
| Field | Type | Description |
|-------|------|-------------|
| `consecutive_failures` | integer | Count of back-to-back exhausted issues |
| `tripped` | boolean | When true, /execute refuses to start |
| `last_updated` | ISO 8601 string | Timestamp of last state write |
| `last_failure_issue` | integer or null | Issue number of last failure |

## result-N.json

Written to repo root on the issue branch after successful implementation. See [output-contract.md](output-contract.md) for full schema.

## GitHub Label States

| Label | Meaning |
|-------|---------|
| `todo` | Ready to execute — all dependencies resolved |
| `in_progress` | Currently being worked on by an agent |
| `blocked` | Dependency unresolved, or exhausted retries |
| `awaiting_review` | Agent opened a PR; waiting for human merge |
| `done` | PR merged; issue closed |

Label transitions are mutually exclusive — an issue carries exactly one of these labels at a time.

## Issue Body Conventions

### Dependency Declaration
```
depends on #12
depends on #12, #15, #18
```
- Case-insensitive: `Depends on`, `depends on`, `DEPENDS ON` all work.
- Comma-separated for multiple dependencies.
- Parsed by `/execute` to build the dependency graph.

### Subtask Declaration (created by worker)
```
[Subtask of #N]
```
- Used in the title of sub-issues created when an issue is too large or blocked by a missing context.

## tasks.md

Rebuilt by `/execute` and `/interview` at the end of each run. Snapshot of current issue state.

```markdown
# Tasks
<!-- Generated: 2024-01-15T10:30:00Z -->

| # | Title | Labels | Updated |
|---|-------|--------|---------|
| 5 | Add dark mode toggle | in_progress | 2024-01-15 |
| 4 | Fix login redirect bug | todo | 2024-01-14 |
| 3 | Scaffold API endpoints | awaiting_review | 2024-01-13 |
```
