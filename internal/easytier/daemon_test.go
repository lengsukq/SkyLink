package easytier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDaemonWorkDir_Absolute(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "v2.0.0", "windows-amd64", "easytier-core.exe")
	want := filepath.Join(dir, "v2.0.0", "windows-amd64")
	if got := ResolveDaemonWorkDir(bin); got != want {
		t.Fatalf("ResolveDaemonWorkDir(%q) = %q, want %q", bin, got, want)
	}
}

func TestResolveDaemonWorkDir_Empty(t *testing.T) {
	if got := ResolveDaemonWorkDir(""); got != "" {
		t.Fatalf("got %q want empty", got)
	}
	if got := ResolveDaemonWorkDir("   "); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestNewDaemonConfig_SetsWorkDir(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "easytier-core.exe")
	cfg := NewDaemonConfig(bin, "/tmp/env")
	if cfg.BinaryPath != bin {
		t.Fatalf("BinaryPath: %q", cfg.BinaryPath)
	}
	if cfg.EnvFile != "/tmp/env" {
		t.Fatalf("EnvFile: %q", cfg.EnvFile)
	}
	if cfg.WorkDir != dir {
		t.Fatalf("WorkDir: got %q want %q", cfg.WorkDir, dir)
	}
}

func TestNormalizeDaemonBinaryPath_RelativeUsesAbs(t *testing.T) {
	base := t.TempDir()
	rel := filepath.Join("windows-amd64", "easytier-core.exe")
	full := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	got := normalizeDaemonBinaryPath(rel)
	want, err := filepath.Abs(full)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeDaemonBinaryPath_BareNameUnchanged(t *testing.T) {
	if got := normalizeDaemonBinaryPath("easytier-core.exe"); got != "easytier-core.exe" {
		t.Fatalf("got %q", got)
	}
}
