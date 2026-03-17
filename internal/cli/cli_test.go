package cli

import (
	"os"
	"path/filepath"
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

func TestUsageIncludesLSP(t *testing.T) {
	err := Run([]string{"wat"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "lsp") {
		t.Fatalf("usage missing lsp command: %q", err.Error())
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 = 0
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

fn main() -> i32 = 0
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

func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
