# Metaprompter — Test Plan

All tests are manual verification steps. Metaprompter has no unit test suite (it is a prompt/skill framework). Tests verify end-to-end behavior against a real GitHub repository.

## Test 1: Dry-Run Smoke Test

**Purpose**: Verify `/execute` reads config, builds queue, and reports without making changes.

**Setup**: A test GitHub repo with 3 issues labeled `todo`.

**Steps**:
1. Run `/execute --max 0` (zero issues).
2. Verify: no branches created, no labels changed, no comments posted.
3. Verify: summary table shown with correct queue count.

**Pass criteria**: Queue built, zero side effects.

---

## Test 2: GitHub Actions PRWatcher Validation

**Purpose**: Verify the Actions workflow triggers correctly and updates labels/comments.

**Steps**:
1. Manually open a PR with body `Resolves #5`.
2. Merge the PR.
3. Verify: issue #5 label changes from `awaiting_review` → `done`, issue closed.
4. Verify: `DONE` event comment posted to issue #5.
5. Verify: If any open issues had `depends on #5`, verify they transition from `blocked` → `todo` and receive `UNBLOCKED` comments.

**Pass criteria**: All label transitions correct, all event comments present.

---

## Test 3: Dependency Ordering Test

**Purpose**: Verify Kahn's topological sort produces correct execution order.

**Setup**:
- Issue #1: no dependencies, `todo`
- Issue #2: `depends on #1`, `todo`
- Issue #3: `depends on #1`, `todo`
- Issue #4: `depends on #2, #3`, `todo`

**Steps**:
1. Run `/execute --max 1`.
2. Verify: only #1 is executed (others blocked by unresolved deps).
3. After simulating #1 done (close #1): re-run `/execute --max 2`.
4. Verify: #2 and #3 are candidates, one executes.
5. After both done: verify #4 becomes executable.

**Pass criteria**: Order respects dependency graph at each step.

---

## Test 4: Circuit Breaker Test

**Purpose**: Verify the circuit breaker halts execution after threshold consecutive failures.

**Setup**: `failure_threshold: 2` in config. Two issues that will fail (e.g., impossible tasks).

**Steps**:
1. Run `/execute`. First issue fails and retries, then is marked blocked. `consecutive_failures: 1`.
2. Second issue also fails. `consecutive_failures: 2` → `tripped: true`.
3. Verify: GitHub issue created describing the halt.
4. Verify: Execution stops.
5. Manually reset `state.json`. Re-run `/execute`. Verify it proceeds.

**Pass criteria**: Halts at threshold, resumes after manual reset.

---

## Test 5: Routing Test

**Purpose**: Verify label-based routing selects the correct prompt template.

**Setup**:
- Issue A labeled `bug`
- Issue B labeled `frontend`
- Issue C labeled `enhancement` (no routing entry)

**Steps**:
1. Inspect which prompt template `/execute` would use for each issue (check via HEARTBEAT or status output).
2. Issue A → `prompts/execute/worker-bugfix.md`
3. Issue B → `prompts/execute/worker-frontend.md`
4. Issue C → `prompts/execute/worker.md` (fallback)

**Pass criteria**: Each issue routes to the correct template.

---

## Test 6: Event Log Format Test

**Purpose**: Verify event comments are machine-readable.

**Steps**:
1. Run `/execute` on a single issue through to PR creation.
2. Fetch all comments: `gh issue view N --json comments --jq '.comments[].body'`
3. Verify presence of: START, at least one HEARTBEAT, PR_OPENED events.
4. Verify each has required fields per the [event-log spec](event-log.md).
5. Parse the HTML comment blocks — verify valid field:value format.

**Pass criteria**: All events present, all required fields present, parseable format.

---

## Test 7: OpenCode Parity Test

**Purpose**: Verify OpenCode prompts produce equivalent behavior to Claude Code skills.

**Steps**:
1. Set up a test repo with one `todo` issue.
2. Run `opencode run "$(cat prompts/opencode/execute.md)"`.
3. Verify: branch created, implementation attempted, PR opened (or blocked), event comments posted, result file written.
4. Compare behavior against the same issue run via `/execute` in Claude Code.

**Pass criteria**: Equivalent state transitions and artifacts, modulo implementation quality differences.

---

## Test 8: Codex Parity Test

Same as Test 7 but with: `codex exec - < prompts/codex/execute.md`

---

## Test 9: METAPROMPTER_ISSUE Env Var Override

**Purpose**: Verify OpenCode/Codex can target a specific issue via env var.

**Steps**:
1. Set `METAPROMPTER_ISSUE=7`.
2. Run `opencode run "$(cat prompts/opencode/execute.md)"`.
3. Verify: only issue #7 is executed, regardless of queue order.

**Pass criteria**: Single targeted issue executed.
