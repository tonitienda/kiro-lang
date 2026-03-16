package cli

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if _, err := os.Stat(filepath.Join(out, "module.go")); err != nil {
		t.Fatalf("generated file missing: %v", err)
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

func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
