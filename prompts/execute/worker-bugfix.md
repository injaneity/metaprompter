# Execute Worker Prompt — Bugfix

You are implementing issue #{{issue_number}}: {{issue_title}}

Issue URL: {{issue_url}}

Issue body:
{{issue_body}}

---

## Before You Start

1. Re-read the issue body carefully. Identify the exact reported behavior vs. expected behavior.
2. If the issue says `depends on #N`, verify the dependency work is present before proceeding.
3. Read the relevant code paths before touching anything.

---

## Bug-Fix Protocol

### Step 1: Reproduce First
Before changing any production code, write a failing test that demonstrates the bug:
- Unit test, integration test, or a minimal reproduction script — whatever fits the project.
- The test must fail on the current code and pass after the fix.
- If you cannot write an automated test (e.g., visual bug, environment-specific), document the manual reproduction steps in a comment.

### Step 2: State Root Cause
In a code comment above your fix (or in the PR body), explicitly state:
```
Root cause: <what was wrong and why>
```
Do not proceed to step 3 without completing step 2.

### Step 3: Minimal Fix
Fix the root cause. Do not:
- Refactor unrelated code alongside the fix
- "Improve" adjacent logic that isn't broken
- Add features while fixing bugs

If you notice other bugs, create separate issues for them rather than fixing them here.

### Step 4: Regression Test
Ensure the test from Step 1 is committed alongside the fix. If no automated test is possible, document explicitly why in the PR body:
```
Regression test: not applicable because <reason>
```

---

## PR Body Requirements

Your implementation must be committed. The orchestrator will create the PR, but ensure your commit message includes:

```
Fix: <what was broken>

Root cause: <explanation>
Regression test: [added | not applicable because X]

Resolves #{{issue_number}}
```

---

## Heartbeat

If the investigation takes more than ~5 minutes, post a heartbeat:

```bash
gh issue comment {{issue_number}} --body "[metaprompter] HEARTBEAT — issue #{{issue_number}}

<!-- metaprompter-event
type: HEARTBEAT
issue: {{issue_number}}
timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
message: <brief status note>
-->"
```

---

## Quality Bar

- Fix is targeted — diff should be small and easy to review.
- Existing tests must still pass.
- No debug logging, no commented-out code.

---

## When Done

Stop after completing the fix and test. The orchestrator handles PR creation, label transitions, and result file writing.

Framework: {{framework}}
