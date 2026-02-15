package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	tpl := "Issue {{issue_number}}: {{title}}"
	out, err := RenderTemplate(tpl, map[string]string{
		"issue_number": "42",
		"title":        "Add execute mode",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "Issue 42: Add execute mode" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRenderTemplateUnresolved(t *testing.T) {
	_, err := RenderTemplate("{{missing}}", map[string]string{})
	if err == nil {
		t.Fatal("expected unresolved placeholder error")
	}
}

func TestLoadTemplateOrFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(path, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := LoadTemplateOrFallback(path, "fallback"); got != "abc" {
		t.Fatalf("expected file content, got %q", got)
	}
	if got := LoadTemplateOrFallback(filepath.Join(dir, "missing.md"), "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}
