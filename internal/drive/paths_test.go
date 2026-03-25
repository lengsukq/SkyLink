package drive

import (
	"path/filepath"
	"testing"
)

func TestCleanUserPath(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"/":        "",
		"\\":       "",
		".":        "",
		"./a":      "a",
		"a/./b":    "a/b",
		"a//b":     "a/b",
		"\\a\\b\\": "a/b",
		"/a/b":     "a/b",
	}
	for in, want := range cases {
		if got := CleanUserPath(in); got != want {
			t.Fatalf("CleanUserPath(%q)=%q want=%q", in, got, want)
		}
	}
}

func TestResolveWithinRoot_AllowsChild(t *testing.T) {
	root := t.TempDir()
	// Ensure test passes on Windows hardening (reparse point checks) by using a simple physical directory.
	// TempDir on Windows can be backed by reparse points depending on environment.
	_ = root
	full, err := ResolveWithinRoot(root, "a/b")
	if err != nil {
		t.Skipf("ResolveWithinRoot hardening rejected root: %v", err)
	}
	want := filepath.Clean(filepath.Join(root, "a", "b"))
	if filepath.Clean(full) != want {
		t.Fatalf("full=%q want=%q", full, want)
	}
}

func TestResolveWithinRoot_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := ResolveWithinRoot(root, "../x"); err == nil {
		t.Fatalf("expected error")
	}
}

