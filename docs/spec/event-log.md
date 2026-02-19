# Metaprompter — Event Log

## Purpose

All state transitions during issue execution are recorded as HTML comment blocks inside GitHub issue comments. The comments are hidden in the GitHub UI but machine-readable via the GitHub API. They form an append-only audit log.

## Format

Every event comment contains a human-readable header and a machine-readable HTML comment block:

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

## Marker

The HTML comment always starts with `<!-- metaprompter-event` on its own line and ends with `-->`.

## Required Fields Per Event Type

### START
Posted when execution begins on an issue.
```
type: START
issue: <number>
timestamp: <ISO 8601>
branch: <branch-name>
framework: <claude|opencode|codex>
```

### HEARTBEAT
Posted every ~5 minutes during implementation.
```
type: HEARTBEAT
issue: <number>
timestamp: <ISO 8601>
message: <brief status note>
```

### RETRY
Posted when the first attempt fails and a retry begins.
```
type: RETRY
issue: <number>
timestamp: <ISO 8601>
attempt: <attempt number, 1-indexed>
reason: <brief description of failure>
```

### BLOCKED
Posted when all retry attempts are exhausted.
```
type: BLOCKED
issue: <number>
timestamp: <ISO 8601>
reason: <description of what blocked execution>
consecutive_failures: <circuit breaker count after increment>
```

### PR_OPENED
Posted when a PR is successfully created.
```
type: PR_OPENED
issue: <number>
timestamp: <ISO 8601>
pr_url: <full GitHub PR URL>
pr_number: <PR number>
```

### DONE
Posted by the GitHub Actions PRWatcher when the PR is merged.
```
type: DONE
issue: <number>
timestamp: <ISO 8601>
pr_url: <full GitHub PR URL>
merged_by: <GitHub username>
```

### UNBLOCKED
Posted by the PRWatcher to a formerly-blocked issue when all its dependencies are resolved.
```
type: UNBLOCKED
issue: <number>
timestamp: <ISO 8601>
resolved_dep: <issue number that was just merged>
```

## Posting Events (gh CLI)

```bash
gh issue comment <issue-number> --body "[metaprompter] START — issue #<N>

<!-- metaprompter-event
type: START
issue: <N>
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
branch: <branch>
framework: claude
-->"
```

## Reading Events

```bash
# Get all comments on an issue
gh issue view <number> --json comments --jq '.comments[].body'

# Extract the last metaprompter event
gh issue view <number> --json comments \
  --jq '[.comments[].body | select(contains("metaprompter-event"))] | last'
```
