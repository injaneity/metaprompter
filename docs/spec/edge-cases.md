# Metaprompter — Edge Cases

## Circular Dependencies

**Scenario**: Issue #3 depends on #5, and #5 depends on #3.

**Detection**: During topological sort (Kahn's algorithm), if the sorted output length is less than the total node count, a cycle exists.

**Handling**:
- `/execute` detects the cycle, logs a warning, and skips the cyclic group entirely.
- An error comment is posted to each issue in the cycle identifying the circular dependency.
- The remaining non-cyclic issues are still executed.

## Missing Dependencies

**Scenario**: Issue body says `depends on #99`, but issue #99 does not exist (was deleted or never created).

**Handling**:
- The missing dep is treated as resolved (not blocking). This prevents permanent deadlock from deleted issues.
- A warning comment is posted to the dependent issue noting the missing dep number.

## PR or Branch Already Exists

**Scenario**: Running `/execute` on an issue that already has a branch or open PR from a previous partial run.

**Branch exists**:
- Check out the existing branch instead of creating a new one.
- Continue from current state — do not reset or recreate.

**PR already open**:
- Skip `gh pr create` — post an `PR_OPENED` event pointing to the existing PR.
- Set `awaiting_review` label if not already set.

## Circuit Breaker Reset

**Scenario**: Circuit breaker tripped (`tripped: true` in `state.json`). User wants to resume.

**Manual reset procedure**:
1. Investigate the blocked issues to understand root cause.
2. Resolve the underlying problems (fix the codebase, adjust issue scope, etc.).
3. Edit `.metaprompter/state.json`: set `consecutive_failures` to 0 and `tripped` to `false`.
4. Re-run `/execute`.

There is no automatic reset. The circuit breaker requires deliberate human intervention.

## Heartbeat Timing

**Scenario**: `/execute` is working on a long-running task. How often to post heartbeats?

**Rule**: Post a `HEARTBEAT` comment if ~5 minutes have elapsed since the last event comment (START, HEARTBEAT, or RETRY) on the issue. This is approximate — prefer erring on the side of more frequent heartbeats.

**Why**: Prevents a stuck `in_progress` issue from being indistinguishable from an active one. A human checking `/status` can see when the last heartbeat was posted.

## awaiting_review Race

**Scenario**: Two agents run `/execute` simultaneously on the same issue.

**Mitigation**:
- Check current labels before setting `in_progress`. If already `in_progress` or `awaiting_review`, skip that issue.
- `gh issue edit --add-label` is not atomic, but the label check at the start of the loop provides a soft guard.
- `max_concurrent: 1` is the recommended configuration. Parallel execution is not officially supported.

## Issues Stuck in in_progress

**Scenario**: A previous run crashed mid-execution. The issue is labeled `in_progress` with no PR.

**Handling**: `/execute` will **not** automatically re-claim `in_progress` issues. This is intentional — a human should verify the branch state before re-running.

**Recovery**:
1. Inspect the branch and any partial changes.
2. Manually change the label back to `todo` (or `blocked` if investigation reveals a problem).
3. Re-run `/execute`.

## Large Dependency Graphs (>30 issues)

**Scenario**: Topological sort on a very large issue graph.

**Behavior**: The implementation works for ≤ ~30 issues. Beyond that, issues with zero detected dependencies are processed first, then `/execute` is re-run to pick up newly unblocked issues. This is a known limitation — there is no automatic multi-pass scheduling.

## Routing Label Conflict

**Scenario**: An issue has labels `frontend` and `bug` — both have routing entries.

**Behavior**: First match wins based on the order labels appear when returned by `gh issue view --json labels`. This is deterministic per issue but depends on label creation order. To make routing explicit: apply only one routing label per issue.
