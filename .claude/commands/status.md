# /status

Display a dashboard of the current metaprompter state: issues by label, circuit breaker, open PRs, recent events, and result summaries.

---

## Step 1: Issue State by Label

Fetch all open issues grouped by label:

```bash
gh issue list --state open --json number,title,labels,updatedAt --limit 100
```

Print grouped sections:

**In Progress**
Issues currently labeled `in_progress`.

**Awaiting Review**
Issues labeled `awaiting_review` (agent done, PR open, waiting for merge).

**Blocked**
Issues labeled `blocked` (exhausted retries or unresolved deps).

**Todo**
Issues labeled `todo` (queued, all deps resolved).

**Recently Closed (last 10)**
```bash
gh issue list --state closed --json number,title,closedAt --limit 10
```

---

## Step 2: Circuit Breaker State

Read `.metaprompter/state.json` (if missing: assume healthy state).

Display:
```
Circuit Breaker: OK (1/3 failures)
```
or:
```
Circuit Breaker: ⚠ TRIPPED — execution halted
  Reset: set consecutive_failures=0, tripped=false in .metaprompter/state.json
```

---

## Step 3: Open PRs Cross-Referenced

```bash
gh pr list --state open --json number,title,url,headRefName
```

For each open PR, show which issue it resolves (parse `Resolves #N` from PR body):
```bash
gh pr view <number> --json body --jq '.body'
```

Display:
```
Open PRs:
  #87 — Add dark mode toggle  →  Issue #42  [awaiting_review]
  #88 — Fix login redirect    →  Issue #39  [awaiting_review]
```

---

## Step 4: Recent Event Log Per Active Issue

For each issue currently `in_progress` or `awaiting_review`:

```bash
gh issue view <number> --json comments \
  --jq '[.comments[].body | select(contains("metaprompter-event"))] | last'
```

Display the last event type and timestamp:
```
Issue #42 — last event: HEARTBEAT at 2024-01-15T11:35:00Z
Issue #39 — last event: PR_OPENED at 2024-01-15T10:20:00Z
```

---

## Step 5: Result File Summaries

Find all `result-*.json` files in the repo root:
```bash
ls result-*.json 2>/dev/null
```

For each file, read and display:
```
result-42.json — Issue #42 — success — confidence: 0.9
  "Added dark mode toggle to Settings page using CSS variables"

result-39.json — Issue #39 — success — confidence: 0.85
  "Fixed login redirect to use returnUrl param from query string"
```

---

## Step 6: Print Dashboard

Assemble all sections into a formatted dashboard. Example output:

```
═══════════════════════════════════════════
  METAPROMPTER STATUS  2024-01-15 11:45 UTC
═══════════════════════════════════════════

IN PROGRESS (1)
  #42  Add dark mode toggle  [last event: HEARTBEAT 11:35]

AWAITING REVIEW (1)
  #39  Fix login redirect  →  PR #88

BLOCKED (1)
  #6   Scaffold auth module

TODO (3)
  #44  Add user preferences page  (depends on #42)
  #45  Add email notifications
  #46  Write API docs

RECENTLY CLOSED
  #38  Initial project scaffold  (closed 2024-01-14)

CIRCUIT BREAKER: OK (0/3)

OPEN PRS
  #87 — dark mode toggle  →  Issue #42 (in_progress)
  #88 — fix login redirect  →  Issue #39 (awaiting_review)

RESULT FILES
  result-39.json  success  0.85  "Fixed login redirect..."
═══════════════════════════════════════════
```
