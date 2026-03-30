package easytier

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveRuntimePaths_SiblingCLI(t *testing.T) {
	dir := t.TempDir()
	coreName := "easytier-core"
	cliName := "easytier-cli"
	if runtime.GOOS == "windows" {
		coreName += ".exe"
		cliName += ".exe"
	}
	core := filepath.Join(dir, "v1", "windows-amd64", coreName)
	if err := os.MkdirAll(filepath.Dir(core), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(core, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(filepath.Dir(core), cliName)
	if err := os.WriteFile(cli, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := ResolveRuntimePaths(core)
	if !strings.HasSuffix(filepath.Clean(paths.CorePath), coreName) {
		t.Fatalf("CorePath: %q", paths.CorePath)
	}
	if paths.CLIPath == "" || paths.CLIPath == CLICommand {
		t.Fatalf("expected concrete CLI path, got %q", paths.CLIPath)
	}
	if paths.WorkDir == "" {
		t.Fatal("expected WorkDir")
	}
	client := NewRPCClient(paths, "127.0.0.1:15888")
	if client.workDir != paths.WorkDir {
		t.Fatalf("client workDir %q != paths %q", client.workDir, paths.WorkDir)
	}
}
