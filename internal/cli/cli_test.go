package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	if err := Run([]string{"help"}); err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
}

func TestRunUnknownCommandIncludesUsage(t *testing.T) {
	err := Run([]string{"wat"})
	if err == nil {
		t.Fatalf("Run(unknown) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "usage: kiro") {
		t.Fatalf("Run(unknown) error = %q, want usage text", err.Error())
	}
}

func TestUsageIncludesRuntimeCommands(t *testing.T) {
	err := Run([]string{"wat"})
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, cmd := range []string{"lsp", "build", "run", "test"} {
		if !strings.Contains(err.Error(), cmd) {
			t.Fatalf("usage missing %s command: %q", cmd, err.Error())
		}
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 {
  return 0
}
`)
	if err := Run([]string{"check", dir}); err != nil {
		t.Fatalf("Run(check) error = %v", err)
	}
}

func TestRunInspectGo(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

const Name = "kiro"
`)
	out := filepath.Join(dir, ".kiro-gen")
	if err := Run([]string{"inspect", "go", dir, "--out-dir", out}); err != nil {
		t.Fatalf("Run(inspect) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "src", "module.go")); err != nil {
		t.Fatalf("generated file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "runtime", "README.txt")); err != nil {
		t.Fatalf("runtime layout missing: %v", err)
	}
}

func TestRunCompat(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 {
  return 0
}
`)
	if err := Run([]string{"compat", dir, "--mode", "fmt,check"}); err != nil {
		t.Fatalf("Run(compat) error = %v", err)
	}
}

func TestRunNewHello(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	defer os.Chdir(prev)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := Run([]string{"new", "hello"}); err != nil {
		t.Fatalf("Run(new hello) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hello", "main.ki")); err != nil {
		t.Fatalf("hello template missing: %v", err)
	}
}

func TestRunNewService(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	defer os.Chdir(prev)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := Run([]string{"new", "service"}); err != nil {
		t.Fatalf("Run(new service) error = %v", err)
	}
	for _, p := range []string{
		"service/main.ki",
		"service/app/main.ki",
		"service/internal/config/main.ki",
		"service/test/health.ki",
		"service/README.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Fatalf("service template file missing %s: %v", p, err)
		}
	}
}

func TestRuntimeBuildAndRun(t *testing.T) {
	kiro := buildKiroBinary(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 !io {
  println("hello from build")
  return 0
}
`)
	out := filepath.Join(dir, binaryName("app"))
	cmd := exec.Command(kiro, "build", dir, "--out", out)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kiro build failed: %v\n%s", err, output)
	}
	run := exec.Command(out)
	run.Env = os.Environ()
	runOut, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("built binary failed: %v\n%s", err, runOut)
	}
	if !strings.Contains(string(runOut), "hello from build") {
		t.Fatalf("built binary output = %q, want greeting", string(runOut))
	}
}

func TestRuntimeRunPassesArgs(t *testing.T) {
	kiro := buildKiroBinary(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

import cli

fn main() -> i32 !io !proc {
  let args = cli.args()
  println("${args}")
  return 0
}
`)
	cmd := exec.Command(kiro, "run", dir, "--", "a", "b")
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kiro run failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "[a b]") {
		t.Fatalf("kiro run output = %q, want forwarded args", string(output))
	}
}

func TestRuntimeTestCommand(t *testing.T) {
	kiro := buildKiroBinary(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

import test

fn add(a:i32, b:i32) -> i32 {
  return a + b
}

fn test_add() -> nil {
  test.eq(add(2, 3), 5)
}
`)
	cmd := exec.Command(kiro, "test", dir)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kiro test failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS") {
		t.Fatalf("kiro test output = %q, want PASS", string(output))
	}
}

func buildKiroBinary(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), binaryName("kiro"))
	cmd := exec.Command("go", "build", "-o", out, filepath.Join("..", "..", "cmd", "kiro"))
	cmd.Env = os.Environ()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/kiro failed: %v\n%s", err, output)
	}
	return out
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
