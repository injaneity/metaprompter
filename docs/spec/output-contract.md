# Metaprompter — Output Contract (result-N.json)

## Purpose

`result-N.json` is written to the repo root on the issue branch after successful implementation. It provides a structured audit trail that merges to `main` with the PR. The `/status` skill reads these files to display per-issue summaries.

## File Path

`result-<issue_number>.json` in the repository root.

Example: `result-42.json`

## Schema

```json
{
  "issue": 42,
  "status": "success",
  "summary": "Added dark mode toggle to Settings page using CSS variables",
  "confidence": 0.9,
  "pr_url": "https://github.com/owner/repo/pull/87",
  "branch": "issue-42-dark-mode-toggle",
  "files_changed": [
    "src/components/Settings.tsx",
    "src/styles/tokens.css"
  ],
  "checks": {
    "test_cmd": "passed",
    "lint_cmd": "passed"
  },
  "completed_at": "2024-01-15T11:42:00Z",
  "worker_notes": "Used existing --theme CSS variable pattern. No new dependencies added."
}
```

## Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `issue` | integer | yes | GitHub issue number |
| `status` | string | yes | `"success"` or `"blocked"` |
| `summary` | string | yes | One-sentence description of what was implemented |
| `confidence` | float | yes | 0.0–1.0: agent's confidence in correctness |
| `pr_url` | string | when success | Full GitHub PR URL |
| `branch` | string | yes | Branch name used |
| `files_changed` | array of strings | yes | Relative paths of modified files |
| `checks.test_cmd` | string | when configured | `"passed"`, `"failed"`, or `"skipped"` |
| `checks.lint_cmd` | string | when configured | `"passed"`, `"failed"`, or `"skipped"` |
| `completed_at` | ISO 8601 string | yes | Timestamp of result write |
| `worker_notes` | string | no | Anything notable about the implementation approach |

## Status Semantics

- `"success"` — Implementation complete, checks passed, PR opened, `awaiting_review` label set.
- `"blocked"` — Retries exhausted. Written on the issue branch if one was created. Issue labeled `blocked`.

## How It Is Consumed

- `/status` reads all `result-*.json` files in the repo root and displays confidence + summary per issue.
- Human reviewers can inspect it alongside the PR diff.
- The file merges to `main` with the PR, forming a permanent record.
