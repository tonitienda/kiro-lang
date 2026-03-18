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

func TestLoadRejectsMissingEffects(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

import env

fn load() -> str !env {
  return env.get_or("PORT", ":8080")
}

fn main() -> i32 {
  let port = load()
  println(port)
  return 0
}
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, `function "main" calls "load" which requires effect "!env"`) {
		t.Fatalf("diagnostic = %q", msg)
	}
}

func TestLoadRejectsUnknownEffects(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 !database {
  return 0
}
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `unknown effect "!database"`) {
		t.Fatalf("diagnostic = %q", err)
	}
}

func TestLoadRejectsDuplicateEffects(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 !env !env {
  return 0
}
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `duplicate effect "!env"`) {
		t.Fatalf("diagnostic = %q", err)
	}
}

func TestLoadAllowsPureJSONCalls(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

import json

type Msg {
  text:str
}

fn main() -> i32 {
  let body = json.encode(Msg{text:"ok"})?
  println(body)
  return 0
}
`)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), `requires effect "!io"`) {
		t.Fatalf("expected io-only error, got %v", err)
	}
}

func TestLoadPropagatesSpawnedEffects(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main

fn work() -> nil !io {
  println("hi")
}

fn main() -> i32 {
  group {
    let t = spawn work()
    await t
  }
  return 0
}
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `requires effect "!io"`) {
		t.Fatalf("diagnostic = %q", err)
	}
}

func mustWrite(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
