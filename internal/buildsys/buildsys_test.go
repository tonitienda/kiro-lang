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
  println(value : "hello from buildsys")
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

func TestBuildRunReportsInvalidResultUnwrap(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn value() -> i32 {
  return 1
}

fn main() -> i32 {
  let n = value()?
  return n
}
`)
	out := filepath.Join(dir, binaryName("bad-unwrap"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("expected runtime failure, got success: %s", output)
	}
	if !strings.Contains(string(output), "invalid ? on non-result value") {
		t.Fatalf("runtime output = %q", string(output))
	}
}

func TestBuildRunReportsAwaitHint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn value() -> i32 {
  return 1
}

fn main() -> i32 {
  let n = await value()
  return n
}
`)
	out := filepath.Join(dir, binaryName("bad-await"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("expected runtime failure, got success: %s", output)
	}
	if !strings.Contains(string(output), "await requires a spawned task") || !strings.Contains(string(output), "call spawn first") {
		t.Fatalf("runtime output = %q", string(output))
	}
}

func TestQuoteServiceExample(t *testing.T) {
	entry := filepath.Join("..", "..", "examples", "quote_service")
	out := filepath.Join(t.TempDir(), binaryName("quote-tests"))
	result, err := Build(Options{Entry: entry, Out: out, Mode: ModeTest})
	if err != nil {
		t.Fatalf("build quote service tests: %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	output, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("quote service tests failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "ok\t14 test(s)") {
		t.Fatalf("expected the complete quote service suite to execute:\n%s", output)
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

func TestFunctionalExampleNativeBinary(t *testing.T) {
	for _, tc := range []struct {
		mode BuildMode
		want string
	}{{ModeBuild, "3700\n"}, {ModeTest, "ok\t3 test(s)"}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			out := filepath.Join(t.TempDir(), binaryName("functional"))
			result, err := Build(Options{Entry: filepath.Join("..", "..", "examples", "functional"), Out: out, Mode: tc.mode})
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(result.WorkDir)
			output, err := exec.Command(out).CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("want %q, got %s", tc.want, output)
			}
		})
	}
}
