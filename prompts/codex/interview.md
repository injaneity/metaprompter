# Metaprompter Interview — Codex

Invocation: `codex exec - < prompts/codex/interview.md`

You are running the specs-first interview phase for this repository using Codex.

Use the file tool to read files and the shell tool to run commands.

---

## Step 1: Load Context

Before asking anything, read all available context:

1. Use the file tool to read `.metaprompter/config.yaml`.
2. Use the file tool to read all existing `docs/spec/*.md` files (if any).
3. Use the file tool to read `tasks.md` (if present).
4. Use the file tool to read `README.md`.
5. Use the shell tool to run: `gh issue list --state open --json number,title,labels,body --limit 50`
6. If the user's request references specific code, read those files before asking questions.

Summarize what you found.

---

## Step 2: Classify Intent

Classify the request into one of:
- **trivial** — single-line fix, obvious change
- **simple** — well-understood enhancement
- **refactor** — restructuring without new behavior
- **new-build** — new module or significant feature
- **architecture** — system-level design
- **research** — investigation before design

Adjust interview depth: trivial/simple → minimal; refactor/new-build/architecture → full protocol.

---

## Step 3: Interview Protocol

Ask one focused question at a time. Read the codebase before asking.

Guidelines:
- Probe vague terms into concrete criteria.
- If alternatives exist, present 2–3 options with tradeoffs.
- Keep a running draft of decisions; update after each answer.
- Never ask what you can discover by reading files.

Anti-patterns: no boilerplate questions, no code generation, no jumping to tasks early.

---

## Step 4: Write Spec Files

When ambiguity is resolved, write spec files using the file tool:

- `docs/spec/overview.md`
- `docs/spec/architecture.md`
- `docs/spec/data-model.md`
- `docs/spec/output-contract.md`
- `docs/spec/event-log.md`
- `docs/spec/routing.md`
- `docs/spec/edge-cases.md`
- `docs/spec/test-plan.md`
- `docs/modules/<name>.md` for non-trivial modules

Omit files that don't apply.

---

## Step 5: Create GitHub Issues

For each unit of work, use the shell tool:

```bash
gh issue create \
  --title "<imperative title>" \
  --body "<description>\n\ndepends on #N" \
  --label "todo"
```

Add routing labels (`frontend`, `bug`) where appropriate.

Rebuild `tasks.md`:
```bash
gh issue list --state open --json number,title,labels,updatedAt \
  --jq '"# Tasks\n<!-- Generated: " + (now | strftime("%Y-%m-%dT%H:%M:%SZ")) + " -->\n\n| # | Title | Labels | Updated |\n|---|-------|--------|---------|" + (.[] | "\n| " + (.number|tostring) + " | " + .title + " | " + (.labels | map(.name) | join(", ")) + " | " + .updatedAt[:10] + " |")' \
  > tasks.md
```

---

## Step 6: Confirm Completion

List spec files written, issues created, and current tasks.md state. Suggest: "Run the execute prompt to begin implementation."
