# /interview

Run the specs-first interview phase. Elicit requirements, write `docs/spec/` files, and create GitHub issues.

---

## Step 1: Load Context

Before asking anything, read all available context:

1. Read `.metaprompter/config.yaml` — understand repo config and any existing routing.
2. Read all existing `docs/spec/*.md` files (if any) — understand prior decisions.
3. Read `tasks.md` (if present) — understand the current issue backlog.
4. Read `README.md` — understand the project.
5. Run `gh issue list --state open --json number,title,labels,body --limit 50` — see current GitHub issue state.
6. If the user's request references specific code, read the relevant files before asking questions.

Summarize what you found: existing spec coverage, open issues count, apparent project stage.

---

## Step 2: Classify Intent

Classify the user's request into one of:

- **trivial** — single-line fix, obvious change, no design needed
- **simple** — well-understood feature, minimal unknowns
- **refactor** — restructuring existing code without new behavior
- **new-build** — new module, service, or significant feature
- **architecture** — system-level design decision
- **research** — investigation/analysis before design

Adjust interview depth accordingly:
- trivial/simple → minimal or no questions; write a brief spec and proceed to issues
- refactor → focus on scope boundaries and what must NOT change
- new-build/architecture → full interview protocol
- research → clarify output format and success criteria

---

## Step 3: Interview Protocol

Ask **one focused question at a time**. Do not ask several questions at once unless they are tightly coupled.

Guidelines:
- Inspect the codebase before asking — never ask what you can read.
- Probe vague terms into concrete criteria: "simple" → what exactly? "fast" → what latency? "production-ready" → what does that mean here?
- If alternatives exist, present 2–3 concise options with tradeoffs.
- Keep a running draft of decisions visible to the user; update it after each answer.
- Ask non-obvious, high-signal questions that materially change the design.

Anti-patterns (forbidden):
- Do not jump to implementation tasks before clarification is complete.
- Do not ask generic boilerplate questions ("What is the purpose of this feature?") that any LLM would produce without reading the codebase.
- Do not generate code during interview mode.
- Do not ask more than one new question at a time.

---

## Step 4: Write Spec Files

When ambiguity is sufficiently resolved, write the spec pack to `docs/spec/`. Standard set:

| File | Content |
|------|---------|
| `docs/spec/overview.md` | What it is, two-phase model, interfaces, requirements |
| `docs/spec/architecture.md` | Component map, state machine, key subsystems |
| `docs/spec/data-model.md` | Schemas: config, state, result files, label states, conventions |
| `docs/spec/output-contract.md` | result-N.json schema and semantics |
| `docs/spec/event-log.md` | Event comment format and required fields |
| `docs/spec/routing.md` | Label→prompt algorithm and template variables |
| `docs/spec/edge-cases.md` | Circular deps, missing deps, circuit breaker reset, race conditions |
| `docs/spec/test-plan.md` | Verification steps for each subsystem |

Also write `docs/modules/<name>.md` for any non-trivial module with its own internal design.

Omit files that do not apply to this project's scope. Write concisely — specs are for agents to execute, not for humans to read on holiday.

---

## Step 5: Create GitHub Issues

After specs are finalized:

1. For each unit of work, create a GitHub issue:
   ```bash
   gh issue create \
     --title "<imperative title>" \
     --body "<description>\n\ndepends on #N" \
     --label "todo"
   ```
   - Use `depends on #N` (comma-separated for multiple) in the body for dependencies.
   - Apply additional labels for routing: `frontend`, `bug`, or any label with a routing entry in config.
   - Keep issues atomic — one branch, one PR per issue.

2. Rebuild `tasks.md`:
   ```bash
   gh issue list --state open --json number,title,labels,updatedAt \
     --jq '"# Tasks\n<!-- Generated: " + now | strftime("%Y-%m-%dT%H:%M:%SZ") + " -->\n\n| # | Title | Labels | Updated |\n|---|-------|--------|---------|" + (.[] | "\n| " + (.number|tostring) + " | " + .title + " | " + (.labels | map(.name) | join(", ")) + " | " + .updatedAt[:10] + " |")' \
     > tasks.md
   ```

---

## Step 6: Confirm Completion

Print a summary:
- List of spec files written (with one-line description each).
- List of GitHub issues created (number + title).
- Current `tasks.md` state (paste or describe).
- Next step: "Run `/execute` to begin implementation."
