# /execute [--issue N] [--max N]

Implement issues from the GitHub backlog. For each issue: branch, implement, run checks, open a PR.

Arguments:
- `--issue N` — execute only issue #N (skip queue logic)
- `--max N` — execute at most N issues this run (default: unlimited)

---

## Step 1: Config + Circuit Breaker

1. Read `.metaprompter/config.yaml`. Extract: labels, routing, circuit_breaker.failure_threshold, execute.retry_attempts, checks.test_cmd, checks.lint_cmd.

2. Read `.metaprompter/state.json` (create if missing with `{"consecutive_failures":0,"tripped":false}`).

3. If `tripped == true`:
   ```
   Circuit breaker tripped. Consecutive failures reached threshold.
   Investigate blocked issues, fix root causes, then reset:
     .metaprompter/state.json → consecutive_failures: 0, tripped: false
   ```
   **STOP. Do not execute any issues.**

---

## Step 2: Build Queue

1. Fetch all open issues:
   ```bash
   gh issue list --state open --json number,title,labels,body,updatedAt --limit 100
   ```

2. Parse `depends on #N` from each issue body (case-insensitive, comma-separated). Build a dependency map: `{ issue_number: [dep_numbers] }`.

3. Fetch closed/done issues to know which deps are resolved:
   ```bash
   gh issue list --state closed --json number --limit 200
   ```
   Also treat any open issue with label `done` as resolved.

4. Apply Kahn's topological sort:
   - Nodes: all open issues with label `todo`
   - Edges: dependency relationships
   - Remove nodes whose all deps are resolved
   - Process in topological order
   - If a cycle is detected (sorted count < node count): log warning, skip cyclic group, continue with rest

5. Filter the sorted list:
   - Keep only issues where all deps are resolved
   - If `--issue N` specified: keep only issue #N (skip queue entirely)
   - If `--max N` specified: take first N from the sorted list

---

## Step 3: Per-Issue Loop

For each issue in the queue:

### 3a. Set in_progress label
```bash
gh issue edit <number> --add-label "in_progress" --remove-label "todo"
```

### 3b. Compute branch name and checkout
Branch naming: `issue-<number>-<slug>` where slug is:
- Issue title lowercased
- Non-alphanumeric chars replaced with `-`
- Consecutive dashes collapsed
- Truncated to 50 chars total (including `issue-N-` prefix)
- Trailing dashes removed

Check if branch already exists:
```bash
git ls-remote --heads origin issue-<number>-<slug>
```
- If exists: `git checkout issue-<number>-<slug> && git pull`
- If not: `git checkout -b issue-<number>-<slug>`

### 3c. Post START event comment
```bash
gh issue comment <number> --body "[metaprompter] START — issue #<number>

<!-- metaprompter-event
type: START
issue: <number>
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
branch: <branch>
framework: claude
-->"
```

### 3d. Route prompt
1. Read `routing:` from config.yaml.
2. Fetch issue labels: `gh issue view <number> --json labels --jq '.labels[].name'`
3. For each label (in order): check if it matches a routing key.
4. First match → read that template file.
5. No match → read `prompts/execute/worker.md`.
6. Substitute template variables: `{{issue_number}}`, `{{issue_title}}`, `{{issue_url}}`, `{{issue_body}}`, `{{issue_labels}}`, `{{branch}}`, `{{framework}}`, `{{repo}}`.

### 3e. Implement
Execute the implementation described in the issue. Use Read, Write, Edit, and Bash tools.

Post a HEARTBEAT comment every ~5 minutes if still working:
```bash
gh issue comment <number> --body "[metaprompter] HEARTBEAT — issue #<number>

<!-- metaprompter-event
type: HEARTBEAT
issue: <number>
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
message: <brief status note>
-->"
```

### 3f. Run checks
If `checks.test_cmd` is set: run it. Record pass/fail.
If `checks.lint_cmd` is set: run it. Record pass/fail.
Failure → proceed to retry/blocked logic (treat as implementation failure).

### 3g. On success
1. Stage and commit:
   ```bash
   git add -A
   git commit -m "<imperative summary>\n\nResolves #<number>"
   git push -u origin <branch>
   ```

2. Write `result-<number>.json` to repo root:
   ```json
   {
     "issue": <number>,
     "status": "success",
     "summary": "<one sentence>",
     "confidence": <0.0-1.0>,
     "pr_url": "",
     "branch": "<branch>",
     "files_changed": ["<path>", ...],
     "checks": {"test_cmd": "<passed|failed|skipped>", "lint_cmd": "<passed|failed|skipped>"},
     "completed_at": "<ISO 8601>",
     "worker_notes": "<optional>"
   }
   ```
   Commit and push the result file.

3. Create PR:
   ```bash
   gh pr create \
     --title "<issue title>" \
     --body "Resolves #<number>\n\n<brief description of changes>" \
     --base main
   ```

4. Update labels:
   ```bash
   gh issue edit <number> --add-label "awaiting_review" --remove-label "in_progress"
   ```

5. Post PR_OPENED event:
   ```bash
   gh issue comment <number> --body "[metaprompter] PR opened — issue #<number>

   <!-- metaprompter-event
   type: PR_OPENED
   issue: <number>
   timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
   pr_url: <pr_url>
   pr_number: <pr_number>
   -->"
   ```

6. Reset circuit breaker: set `consecutive_failures: 0` in `state.json`.

### 3h. On failure
1. If retry attempts remain (check `execute.retry_attempts`):
   ```bash
   gh issue comment <number> --body "[metaprompter] RETRY — issue #<number>

   <!-- metaprompter-event
   type: RETRY
   issue: <number>
   timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
   attempt: <attempt_number>
   reason: <brief description>
   -->"
   ```
   Go back to 3e and retry.

2. On exhausted retries:
   ```bash
   gh issue edit <number> --add-label "blocked" --remove-label "in_progress"
   ```
   Post BLOCKED event. Increment `consecutive_failures` in `state.json`.

   If `consecutive_failures >= failure_threshold`:
   - Set `tripped: true` in `state.json`
   - Create a GitHub issue:
     ```bash
     gh issue create \
       --title "[metaprompter] Circuit breaker tripped" \
       --body "Execution halted after <N> consecutive failures. Last failure: issue #<number>.\n\nInvestigate blocked issues, fix root causes, then reset state.json." \
       --label "blocked"
     ```
   - **STOP. Do not process more issues.**

---

## Step 4: Rebuild tasks.md

After the loop (or after a STOP):
```bash
gh issue list --state open --json number,title,labels,updatedAt --limit 100 \
  --jq '"# Tasks\n<!-- Generated: " + (now | strftime("%Y-%m-%dT%H:%M:%SZ")) + " -->\n\n| # | Title | Labels | Updated |\n|---|-------|--------|---------|" + (.[] | "\n| " + (.number|tostring) + " | " + .title + " | " + (.labels | map(.name) | join(", ")) + " | " + .updatedAt[:10] + " |")' \
  > tasks.md
```

---

## Step 5: Report Summary

Print a table:

```
## Execution Summary

| Issue | Title | Result | PR |
|-------|-------|--------|----|
| #5    | Add dark mode | ✓ awaiting_review | #87 |
| #6    | Fix login bug | ✗ blocked | — |

Circuit breaker: 1/3 consecutive failures
```

List any PRs opened. State circuit breaker status. Note if tripped.
