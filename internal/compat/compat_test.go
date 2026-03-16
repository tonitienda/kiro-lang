package compat

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunCompatibilityFixtures(t *testing.T) {
	if err := Run(RunOptions{Root: compatRoot()}); err != nil {
		t.Fatalf("compat run failed: %v", err)
	}
}

func compatRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "compat")
}
