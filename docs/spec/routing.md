# Metaprompter — Topic Routing

## Purpose

Topic routing maps GitHub issue labels to specialized worker prompt templates. This allows different types of work (frontend, bugfix, etc.) to receive tailored instructions.

## Resolution Algorithm

1. Read `routing:` section from `.metaprompter/config.yaml`.
2. Fetch the issue's current labels from GitHub (`gh issue view N --json labels`).
3. Iterate over the issue's labels in the order they appear.
4. For each label, check if it exists as a key in `routing:`.
5. **First match wins** — return the `prompt:` path for that label.
6. If no label matches any routing entry, fall back to `prompts/execute/worker.md`.

## Template Variables

Worker prompt templates receive the following variables via substitution before the prompt is used. Templates use `{{variable_name}}` syntax.

| Variable | Description |
|----------|-------------|
| `{{issue_number}}` | GitHub issue number |
| `{{issue_title}}` | Issue title |
| `{{issue_url}}` | Full GitHub issue URL |
| `{{issue_body}}` | Full issue body text |
| `{{issue_labels}}` | Comma-separated list of label names |
| `{{branch}}` | Branch name for this issue (`issue-N-slug`) |
| `{{framework}}` | Active framework (`claude`, `opencode`, `codex`) |
| `{{repo}}` | `owner/repo` string |

## Config Example

```yaml
routing:
  frontend:
    prompt: prompts/execute/worker-frontend.md
  bug:
    prompt: prompts/execute/worker-bugfix.md
  docs:
    prompt: prompts/execute/worker-docs.md
```

## Topic Variant Descriptions

### Generic Worker (`prompts/execute/worker.md`)
Default for any issue without a matching routing label. Covers backend work, refactoring, configuration changes, and anything not requiring specialized context.

### Frontend Worker (`prompts/execute/worker-frontend.md`)
Extends the generic worker with:
- Framework/build tool/CSS detection
- Reuse-before-create for existing components
- No new dependencies without justification
- Accessibility (ARIA + keyboard navigation)
- Responsive design defaults
- Zero new build warnings rule

### Bugfix Worker (`prompts/execute/worker-bugfix.md`)
Bug-fix protocol:
1. Reproduce first — write a failing test before touching code
2. State root cause explicitly in a comment before fixing
3. Minimal fix only — no refactoring alongside bug fixes
4. Regression test required (or justify absence)
5. PR body must include regression test status

## Adding New Routes

1. Create a new prompt template at a path of your choosing.
2. Add a routing entry to `.metaprompter/config.yaml`:
   ```yaml
   routing:
     my-label:
       prompt: prompts/execute/worker-my-label.md
   ```
3. Apply the label to GitHub issues to activate routing.
