package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

func WritePromptScaffold(root string) error {
	files := map[string]string{
		filepath.Join(root, "prompts/interview/system.md"): interviewPrompt,
		filepath.Join(root, "prompts/execute/worker.md"):   executeWorkerPrompt,
		filepath.Join(root, "commands/codex.md"):           codexCommand,
		filepath.Join(root, "commands/opencode.md"):        opencodeCommand,
		filepath.Join(root, "commands/claude.md"):          claudeCommand,
	}
	for path, content := range files {
		if err := writeIfMissing(path, content); err != nil {
			return err
		}
	}
	return nil
}

func WriteDefaultConfig(path string) error {
	return writeIfMissing(path, defaultConfig)
}

func writeIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

const defaultConfig = `github:
  # Optional: auto-detected from current git remote origin if empty.
  repo:
  labels:
    todo: todo
    in_progress: in_progress
    blocked: blocked
    done: done
frameworks:
  default: codex
  enabled:
    - codex
    - opencode
    - claude
commands:
  source_dir: commands
  generated_dir: .metaprompter/bin
execute:
  retry_attempts: 1
  allow_subissue_fanout: true
checks:
  test_cmd:
  lint_cmd:
`

const interviewPrompt = `# Interview Mode System Prompt

You are running the specs-first /interview phase for this repository.

Reference guiding principles:
- https://github.com/code-yeongyu/oh-my-opencode/blob/dev/src/agents/prometheus/interview-mode.ts

Operating protocol:
1. Intent classification
- Classify request intent first: trivial, simple enhancement, refactor, new build, architecture design, research/problem-solving, or collaborative discussion.
- Adjust depth accordingly; do not over-interview trivial/simple asks.
2. Context-first analysis
- For refactor/new build/architecture/research intents, inspect existing code/docs/config first.
- Identify unknowns, constraints, and risk areas before asking questions.
3. Interview strategy
- Be a thinking partner, not a checklist bot.
- Ask non-obvious, high-signal questions that materially change design.
- Follow the user's thread and ask one focused question at a time when possible.
- Probe vague terms into concrete criteria.
4. Test and quality framing
- Clarify test strategy, validation approach, and quality bars before finalization.
5. Draft progression
- Keep a running draft of decisions/assumptions and update specs incrementally.

Anti-patterns (forbidden):
- Do not jump directly to implementation plans/tasks before clarification is complete.
- Do not ask generic boilerplate questions detached from context.
- Do not generate code during interview mode.

Finalize behavior:
- When ambiguity is resolved, produce finalized spec files:
- docs/spec/overview.md
- docs/spec/requirements.md
- docs/spec/architecture.md
- docs/spec/data-model.md
- docs/spec/api-conventions.md
- docs/spec/edge-cases.md
- docs/spec/test-plan.md
- docs/spec/issue-map.md
- docs/modules/<module>.md for non-trivial modules
- Only after spec finalization, create/refresh GitHub issues if requested by workflow.
`

const executeWorkerPrompt = `# Execute Worker Prompt

You are the execution worker for issue #{{issue_number}}.

Issue title: {{issue_title}}
Issue URL: {{issue_url}}
Issue body:
{{issue_body}}

Execution protocol:
1. Implement only this issue scope unless explicitly resolving another issue.
2. If blocked by dependency or context-rot, propose/create a subissue.
3. Update progress through issue comments and local log conventions.
4. Keep changes atomic for one issue branch and one pull request.
5. Run tests/lint commands when available.

Framework hint: {{framework}}
`

const codexCommand = `---
framework: codex
description: Codex CLI adapter command (non-interactive)
command: codex exec - < "{{prompt_file}}"
---
# Codex Command

Uses stdin prompt mode for codex exec.
`

const opencodeCommand = `---
framework: opencode
description: OpenCode CLI adapter command (non-interactive)
command: opencode run "$(cat "{{prompt_file}}")"
---
# OpenCode Command

Uses opencode run with prompt content from file.
`

const claudeCommand = `---
framework: claude
description: Claude Code CLI adapter command (non-interactive)
command: claude -p "$(cat "{{prompt_file}}")"
---
# Claude Command

Uses Claude print mode for non-interactive execution.
`
