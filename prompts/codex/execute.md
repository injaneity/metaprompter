# Metaprompter Execute — Codex

Invocation: `codex exec - < prompts/codex/execute.md`

Env var overrides:
- `METAPROMPTER_ISSUE=N` — execute only issue #N
- `METAPROMPTER_MAX=N` — execute at most N issues

Use the file tool to read/write files and the shell tool to run commands.

---

## Step 1: Config + Circuit Breaker

1. Use the file tool to read `.metaprompter/config.yaml`.
2. Use the file tool to read `.metaprompter/state.json` (create if missing: `{"consecutive_failures":0,"tripped":false}`).
3. Check env: `echo ${METAPROMPTER_ISSUE:-}` and `echo ${METAPROMPTER_MAX:-}`.

If `tripped == true` in state.json:
```
Circuit breaker tripped. Execution halted.
Reset: .metaprompter/state.json → consecutive_failures: 0, tripped: false
```
**STOP.**

---

## Step 2: Build Queue

```bash
gh issue list --state open --json number,title,labels,body,updatedAt --limit 100
gh issue list --state closed --json number --limit 200
```

Parse `depends on #N` from each issue body. Build dependency map. Apply Kahn's topological sort on issues labeled `todo`. Filter: only issues with all deps resolved.

If `METAPROMPTER_ISSUE` is set: only that issue.
If `METAPROMPTER_MAX` is set: take first N.

---

## Step 3: Per-Issue Loop

For each issue in the queue:

**3a. Set in_progress:**
```bash
gh issue edit <number> --add-label "in_progress" --remove-label "todo"
```

**3b. Branch:**
Compute `issue-<number>-<slug>` (lowercase, alphanumeric+dash, max 50 chars).
```bash
git ls-remote --heads origin issue-<number>-<slug>
# If exists: git checkout <branch> && git pull
# If not: git checkout -b <branch>
```

**3c. Post START event:**
```bash
gh issue comment <number> --body "[metaprompter] START — issue #<number>

<!-- metaprompter-event
type: START
issue: <number>
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
branch: <branch>
framework: codex
-->"
```

**3d. Route prompt:**
Check issue labels against `routing:` in config.yaml. First match → read that template. No match → read `prompts/execute/worker.md`.

**3e. Implement:**
Use file and shell tools to implement the changes. Post HEARTBEAT every ~5 min:
```bash
gh issue comment <number> --body "[metaprompter] HEARTBEAT — issue #<number>

<!-- metaprompter-event
type: HEARTBEAT
issue: <number>
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
message: <brief status>
-->"
```

**3f. Run checks:**
Run `test_cmd` and `lint_cmd` from config if set.

**3g. On success:**
```bash
git add -A
git commit -m "<summary>\n\nResolves #<number>"
git push -u origin <branch>
```
Write `result-<number>.json`. Commit and push it.
```bash
gh pr create --title "<title>" --body "Resolves #<number>\n\n<description>" --base main
gh issue edit <number> --add-label "awaiting_review" --remove-label "in_progress"
```
Post PR_OPENED event. Set `consecutive_failures: 0` in state.json.

**3h. On failure:**
If retries remain: post RETRY event, retry from 3e.
On exhaustion:
```bash
gh issue edit <number> --add-label "blocked" --remove-label "in_progress"
```
Post BLOCKED event. Increment `consecutive_failures`. If threshold hit: set `tripped: true`, create halt issue, **STOP**.

---

## Step 4: Rebuild tasks.md

```bash
gh issue list --state open --json number,title,labels,updatedAt --limit 100 \
  --jq '"# Tasks\n<!-- Generated: " + (now | strftime("%Y-%m-%dT%H:%M:%SZ")) + " -->\n\n| # | Title | Labels | Updated |\n|---|-------|--------|---------|" + (.[] | "\n| " + (.number|tostring) + " | " + .title + " | " + (.labels | map(.name) | join(", ")) + " | " + .updatedAt[:10] + " |")' \
  > tasks.md
```

---

## Step 5: Report Summary

Print a table of completed/blocked/skipped issues, PR URLs, and circuit breaker state.
