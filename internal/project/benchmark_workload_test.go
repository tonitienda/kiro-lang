package project

import (
	"path/filepath"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

func TestMaintenanceBenchmarkBaseline(t *testing.T) {
	p, err := Load(filepath.Join("..", "..", "benchmarks", "maintenance", "workload"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := runtimekit.RunCoverage(p.RuntimeSpec(), 10000)
	if err != nil || !report.Passed || !report.Complete || len(report.Tests) != 6 {
		t.Fatalf("benchmark public baseline tests: %+v %v", report.Tests, err)
	}
}
