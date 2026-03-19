package toolchain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocatePrefersEnvOverride(t *testing.T) {
	dir := t.TempDir()
	goBin := filepath.Join(dir, goBinaryName())
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go binary: %v", err)
	}
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

func TestGoBinaryName(t *testing.T) {
	if runtime.GOOS == "windows" && goBinaryName() != "go.exe" {
		t.Fatalf("goBinaryName() = %q, want go.exe", goBinaryName())
	}
	if runtime.GOOS != "windows" && goBinaryName() != "go" {
		t.Fatalf("goBinaryName() = %q, want go", goBinaryName())
	}
}
