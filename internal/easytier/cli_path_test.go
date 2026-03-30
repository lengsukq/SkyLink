package easytier

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveCLIPathAdjacentToCore(t *testing.T) {
	dir := t.TempDir()
	coreName := "easytier-core"
	cliName := "easytier-cli"
	if runtime.GOOS == "windows" {
		coreName += ".exe"
		cliName += ".exe"
	}
	core := filepath.Join(dir, "windows-amd64", coreName)
	cli := filepath.Join(dir, "windows-amd64", cliName)
	if err := os.MkdirAll(filepath.Dir(core), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(core, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveCLIPathAdjacentToCore(core)
	if got == "" {
		t.Fatal("expected sibling cli path")
	}
	if !strings.HasPrefix(filepath.Base(got), "easytier-cli") {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCLIPathAdjacentToCore_BareName(t *testing.T) {
	if got := ResolveCLIPathAdjacentToCore("easytier-core.exe"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
