package interview

import "testing"

func TestSelectInterviewFramework(t *testing.T) {
	got := selectInterviewFramework("claude", "codex", []string{"codex", "claude"})
	if got != "claude" {
		t.Fatalf("expected override framework, got %s", got)
	}

	got = selectInterviewFramework("unknown", "codex", []string{"codex", "claude"})
	if got != "codex" {
		t.Fatalf("expected default framework, got %s", got)
	}
}

func TestTruncate(t *testing.T) {
	in := "abcdefghijklmnopqrstuvwxyz"
	out := truncate(in, 10)
	if len(out) != 10 {
		t.Fatalf("expected length 10, got %d", len(out))
	}
	if out[7:] != "..." {
		t.Fatalf("expected ellipsis suffix, got %q", out)
	}
}

func TestInterviewLaunchCommand(t *testing.T) {
	cmd, ok := interviewLaunchCommand("codex")
	if !ok || cmd == "" {
		t.Fatal("expected codex interview command")
	}
	cmd, ok = interviewLaunchCommand("claude")
	if !ok || cmd == "" {
		t.Fatal("expected claude interview command")
	}
	cmd, ok = interviewLaunchCommand("opencode")
	if !ok || cmd == "" {
		t.Fatal("expected opencode interview command")
	}
	_, ok = interviewLaunchCommand("unknown")
	if ok {
		t.Fatal("expected unknown framework to be unsupported")
	}
}
