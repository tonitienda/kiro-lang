package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadResolvesImportsByModuleName(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

import util

fn main() -> i32 = 0
`)
	mustWrite(t, filepath.Join(dir, "util.ki"), `mod util

const Version = "1"
`)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(p.Files) != 2 {
		t.Fatalf("file count = %d", len(p.Files))
	}
}

func TestLoadIncludesSnippetInParserDiagnostics(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), "mod\n")
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "^") || !strings.Contains(msg, "main.ki") {
		t.Fatalf("diagnostic = %q", msg)
	}
}

func mustWrite(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
