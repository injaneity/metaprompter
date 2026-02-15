package adapters

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Definition struct {
	Framework   string
	Description string
	Command     string
	Path        string
}

func BuiltinDefinitions() map[string]Definition {
	return map[string]Definition{
		"codex": {
			Framework:   "codex",
			Description: "Codex CLI adapter command",
			Command:     `codex exec - < "{{prompt_file}}"`,
			Path:        "builtin",
		},
		"opencode": {
			Framework:   "opencode",
			Description: "OpenCode CLI adapter command",
			Command:     `opencode run "$(cat "{{prompt_file}}")"`,
			Path:        "builtin",
		},
		"claude": {
			Framework:   "claude",
			Description: "Claude Code CLI adapter command",
			Command:     `claude -p "$(cat "{{prompt_file}}")"`,
			Path:        "builtin",
		},
	}
}

func LoadDefinitions(dir string) (map[string]Definition, error) {
	defs := BuiltinDefinitions()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defs, nil
		}
		return nil, fmt.Errorf("read commands directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		def, err := parseDefinition(path)
		if err != nil {
			return nil, err
		}
		defs[def.Framework] = def
	}
	return defs, nil
}

func parseDefinition(path string) (Definition, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read command definition %s: %w", path, err)
	}
	content := string(b)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Definition{}, fmt.Errorf("definition %s must include YAML-style front matter", path)
	}
	frontMatter := strings.TrimSpace(parts[1])

	def := Definition{Path: path}
	for _, line := range strings.Split(frontMatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = stripMatchingQuotes(value)
		switch key {
		case "framework":
			def.Framework = value
		case "description":
			def.Description = value
		case "command":
			def.Command = value
		}
	}

	if def.Framework == "" {
		return Definition{}, fmt.Errorf("definition %s missing framework", path)
	}
	if def.Command == "" {
		return Definition{}, fmt.Errorf("definition %s missing command", path)
	}
	return def, nil
}

func GenerateScripts(defs map[string]Definition, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create adapter output directory: %w", err)
	}

	frameworks := make([]string, 0, len(defs))
	for name := range defs {
		frameworks = append(frameworks, name)
	}
	sort.Strings(frameworks)

	for _, name := range frameworks {
		def := defs[name]
		script := generateScript(def)
		outPath := filepath.Join(outputDir, name+".sh")
		if err := os.WriteFile(outPath, []byte(script), 0o755); err != nil {
			return fmt.Errorf("write adapter script %s: %w", outPath, err)
		}
	}

	return nil
}

func generateScript(def Definition) string {
	cmd := shellSingleQuote(def.Command)
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${PROMPT_FILE:-}" ]]; then
  echo "PROMPT_FILE is required" >&2
  exit 1
fi

template='%s'

render() {
  local input="$1"
  input="${input//\{\{prompt_file\}\}/${PROMPT_FILE:-}}"
  input="${input//\{\{issue_number\}\}/${ISSUE_NUMBER:-}}"
  input="${input//\{\{issue_title\}\}/${ISSUE_TITLE:-}}"
  input="${input//\{\{issue_body\}\}/${ISSUE_BODY:-}}"
  input="${input//\{\{issue_url\}\}/${ISSUE_URL:-}}"
  input="${input//\{\{repo_path\}\}/${REPO_PATH:-}}"
  printf '%%s' "$input"
}

cmd="$(render "$template")"
exec bash -lc "$cmd"
`, cmd)
}

func shellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "'\"'\"'")
}

func stripMatchingQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	first := value[0]
	last := value[len(value)-1]
	if (first == '"' || first == '\'') && first == last {
		return value[1 : len(value)-1]
	}
	return value
}
