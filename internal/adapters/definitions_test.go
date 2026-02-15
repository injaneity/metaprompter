package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefinitions(t *testing.T) {
	dir := t.TempDir()
	md := `---
framework: codex
description: Codex adapter
command: codex exec - < "{{prompt_file}}"
---
# Codex
`
	if err := os.WriteFile(filepath.Join(dir, "codex.md"), []byte(md), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}

	defs, err := LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("load definitions: %v", err)
	}
	def, ok := defs["codex"]
	if !ok {
		t.Fatal("expected codex definition")
	}
	if def.Command == "" {
		t.Fatal("expected command")
	}
	if def.Command != `codex exec - < "{{prompt_file}}"` {
		t.Fatalf("unexpected command parse: %q", def.Command)
	}
}

func TestLoadDefinitionsMissingDirUsesBuiltins(t *testing.T) {
	defs, err := LoadDefinitions(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("load builtins: %v", err)
	}
	if len(defs) < 3 {
		t.Fatalf("expected builtin definitions, got %d", len(defs))
	}
	if _, ok := defs["codex"]; !ok {
		t.Fatal("expected codex builtin")
	}
}

func TestGenerateScripts(t *testing.T) {
	dir := t.TempDir()
	defs := map[string]Definition{
		"claude": {
			Framework: "claude",
			Command:   "claude code --prompt-file \"{{prompt_file}}\"",
		},
	}

	if err := GenerateScripts(defs, dir); err != nil {
		t.Fatalf("generate scripts: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "claude.sh"))
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("script should not be empty")
	}
}
