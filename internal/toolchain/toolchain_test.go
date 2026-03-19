package toolchain

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocatePrefersEnvOverride(t *testing.T) {
	dir := t.TempDir()
	goBin := filepath.Join(dir, goBinaryName())
	writeFakeGo(t, goBin)
	t.Setenv("KIRO_GO_BIN", goBin)
	loc, err := Locate()
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if loc.GoBinary != goBin {
		t.Fatalf("Locate() go binary = %q, want %q", loc.GoBinary, goBin)
	}
	if loc.Source != "KIRO_GO_BIN" {
		t.Fatalf("Locate() source = %q, want KIRO_GO_BIN", loc.Source)
	}
}

func TestLocateSkipsBrokenBundledToolchainInFavorOfPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-format simulation uses a non-windows executable stub")
	}

	brokenRoot := t.TempDir()
	brokenGo := filepath.Join(brokenRoot, "go", "bin", goBinaryName())
	if err := os.MkdirAll(filepath.Dir(brokenGo), 0o755); err != nil {
		t.Fatalf("mkdir broken go dir: %v", err)
	}
	if err := os.WriteFile(brokenGo, []byte("not a real executable"), 0o755); err != nil {
		t.Fatalf("write broken go binary: %v", err)
	}

	fallbackDir := t.TempDir()
	fallbackGo := filepath.Join(fallbackDir, goBinaryName())
	writeFakeGo(t, fallbackGo)

	t.Setenv("KIRO_TOOLCHAIN_DIR", brokenRoot)
	t.Setenv("PATH", fallbackDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	loc, err := Locate()
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if loc.GoBinary != fallbackGo {
		t.Fatalf("Locate() go binary = %q, want PATH fallback %q", loc.GoBinary, fallbackGo)
	}
	if loc.Source != "PATH" {
		t.Fatalf("Locate() source = %q, want PATH", loc.Source)
	}
}

func TestLocateRejectsBrokenExplicitOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-format simulation uses a non-windows executable stub")
	}

	dir := t.TempDir()
	goBin := filepath.Join(dir, goBinaryName())
	if err := os.WriteFile(goBin, []byte("not a real executable"), 0o755); err != nil {
		t.Fatalf("write broken go binary: %v", err)
	}

	t.Setenv("KIRO_GO_BIN", goBin)
	_, err := Locate()
	if err == nil {
		t.Fatalf("Locate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "not a usable go binary") {
		t.Fatalf("Locate() error = %q, want unusable go binary message", err.Error())
	}
}

func TestGoBinaryName(t *testing.T) {
	if runtime.GOOS == "windows" && goBinaryName() != "go.exe" {
		t.Fatalf("goBinaryName() = %q, want go.exe", goBinaryName())
	}
	if runtime.GOOS != "windows" && goBinaryName() != "go" {
		t.Fatalf("goBinaryName() = %q, want go", goBinaryName())
	}
}

func writeFakeGo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fake go dir: %v", err)
	}
	var src []byte
	if runtime.GOOS == "windows" {
		src = []byte("@echo off\r\necho go version go-test\r\n")
	} else {
		src = []byte("#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo go version go-test\n  exit 0\nfi\nexit 0\n")
	}
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatalf("write fake go binary: %v", err)
	}
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath(path); err != nil {
			t.Fatalf("fake windows go binary not runnable: %v", err)
		}
	}
}
