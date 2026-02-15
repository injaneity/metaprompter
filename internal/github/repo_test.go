package github

import "testing"

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "ssh", in: "git@github.com:acme/widgets.git", out: "acme/widgets"},
		{name: "https", in: "https://github.com/acme/widgets.git", out: "acme/widgets"},
		{name: "ssh_url", in: "ssh://git@github.com/acme/widgets.git", out: "acme/widgets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubRepo(tt.in)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.out {
				t.Fatalf("expected %q, got %q", tt.out, got)
			}
		})
	}
}

func TestParseGitHubRepoRejectsNonGitHub(t *testing.T) {
	_, err := ParseGitHubRepo("git@gitlab.com:acme/widgets.git")
	if err == nil {
		t.Fatal("expected error for non-github URL")
	}
}
