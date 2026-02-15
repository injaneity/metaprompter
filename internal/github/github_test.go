package github

import "testing"

func TestExtractIssueNumber(t *testing.T) {
	n := extractIssueNumber("https://github.com/acme/project/issues/123")
	if n != 123 {
		t.Fatalf("expected 123, got %d", n)
	}
}
