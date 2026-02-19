# Metaprompter — Architecture

## Component Map

```
User
 │
 ▼
Claude Code / OpenCode / Codex
 │   (reads .claude/commands/ or prompts/opencode/ or prompts/codex/)
 │
 ├── /interview ──────────────────────────────────────────────────┐
 │    Reads: config.yaml, docs/spec/*.md, tasks.md, README.md     │
 │    Writes: docs/spec/*.md                                      │
 │    Calls: gh issue create                                      │
 │    Writes: tasks.md                                            │
 │                                                                │
 ├── /execute ────────────────────────────────────────────────────┤
 │    Reads: config.yaml, state.json                              │
 │    Calls: gh issue list → builds queue → topological sort      │
 │    For each issue:                                             │
 │      gh issue edit (labels)                                    │
 │      git checkout -b issue-N-slug                              │
 │      gh issue comment (event log)                              │
 │      Read/Write/Edit/Bash (implementation)                     │
 │      gh pr create                                              │
 │      writes result-N.json                                      │
 │    Writes: tasks.md, state.json                                │
 │                                                                │
 └── /status ─────────────────────────────────────────────────────┘
      Reads: state.json, result-N.json
      Calls: gh issue list, gh pr list


GitHub Actions (pr-merged.yml)
 │   Trigger: pull_request.types [closed] + merged == true
 │
 ├── Extract linked issue from PR body
 ├── gh issue edit: add done, remove awaiting_review + in_progress
 ├── gh issue close
 ├── gh issue comment (DONE event)
 └── For each open issue with "depends on #N":
       Check all its deps resolved → if yes: add todo, remove blocked, post UNBLOCKED event
```

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
                    │                            │
                    ▼                            ▼
                 [done]                      [blocked]
                                                 │
                                     circuit breaker threshold?
                                                 │
                                              tripped=true → HALT
```

## Dependency Tracking

- Issue bodies contain `depends on #N` (case-insensitive, comma-separated for multiple).
- `/execute` parses all open issues, builds a dependency graph, and applies Kahn's topological sort.
- Only issues whose all dependencies are closed (or labeled `done`) enter the execution queue.
- Issues with unresolved deps stay `blocked` until the PRWatcher unblocks them post-merge.

## Topic Routing

- Issue labels are checked against the `routing:` section of `config.yaml`.
- First matching label wins → its `prompt:` path is read as the worker template.
- Fallback: `prompts/execute/worker.md` (generic worker).
- See [routing.md](routing.md) for full algorithm.

## Circuit Breaker

- `state.json` tracks `consecutive_failures` count.
- On each successful issue completion, counter resets to 0.
- On each exhausted-retry failure, counter increments.
- When counter reaches `circuit_breaker.failure_threshold` (default: 3): set `tripped=true`, create a GitHub issue describing the halt, STOP execution.
- To resume: manually reset `state.json` after investigating failures.

## Heartbeat Comments

- Every ~5 minutes during implementation, `/execute` posts a `HEARTBEAT` event comment to the issue.
- Prevents silent hangs from appearing as stuck `in_progress` issues.

## Event Log

- All state transitions are recorded as HTML comment blocks in issue comments.
- Format: `<!-- metaprompter-event -->` — hidden in GitHub UI, machine-readable via API.
- See [event-log.md](event-log.md) for schema.

## result-N.json

- Written on the issue branch after successful implementation.
- Merges to `main` with the PR — provides a persistent audit trail.
- See [output-contract.md](output-contract.md) for schema.
