package buildsys

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildCreatesRunnableBinary(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 !io {
  println("hello from buildsys")
  return 0
}
`)
	out := filepath.Join(dir, binaryName("app"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("built binary failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "hello from buildsys") {
		t.Fatalf("built binary output = %q, want greeting", string(output))
	}
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
