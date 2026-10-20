package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const analyzedProject = `mod main
import test
fn add(value:i32,amount:i32)->i32 {return value+amount}
fn test_add()->nil {test.eq(actual:add(value:2,amount:3),expected:5)
return nil}
fn main()->i32 {return add(value:2,amount:3)}
`

func nativeAnalysis(t *testing.T, binary, dir, operation string, extra ...string) (toolReport, int) {
	t.Helper()
	args := []string{"analyze", operation, dir}
	args = append(args, extra...)
	args = append(args, "--json")
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("machine stderr: %s", stderr.String())
	}
	var report toolReport
	if err := json.Unmarshal([]byte(stdout.String()), &report); err != nil {
		t.Fatalf("protocol %s: %v", stdout.String(), err)
	}
	return report, code
}

func TestNativeOptInAnalysisWorkflow(t *testing.T) {
	binary := buildKiroBinary(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "main.ki")
	write(t, file, analyzedProject)
	for _, operation := range []string{"graph", "metrics", "cycles", "boundaries"} {
		report, code := nativeAnalysis(t, binary, dir, operation)
		if code != 0 || !report.OK {
			t.Fatalf("%s %+v", operation, report)
		}
	}
	coverage, code := nativeAnalysis(t, binary, dir, "coverage", "--max-steps", "1000")
	if code != 0 || !coverage.OK {
		t.Fatalf("coverage %+v", coverage)
	}
	result := coverage.Data.(map[string]any)["result"].(map[string]any)
	if result["status"] != "passed" {
		t.Fatal(result)
	}
	saved := filepath.Join(t.TempDir(), "coverage.json")
	bytes, _ := json.Marshal(coverage)
	if err := os.WriteFile(saved, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	impact, code := nativeAnalysis(t, binary, dir, "impact", "--symbol", "main.add", "--coverage", saved)
	if code != 0 || !impact.OK {
		t.Fatalf("impact %+v", impact)
	}
	observed := impact.Data.(map[string]any)["result"].(map[string]any)["observed_tests"].([]any)
	if len(observed) != 1 || observed[0] != "main.test_add" {
		t.Fatal(observed)
	}
	slice, code := nativeAnalysis(t, binary, dir, "slice", "--symbol", "main.add", "--value", "return")
	if code != 0 || !slice.OK {
		t.Fatal(slice)
	}
	mutants, code := nativeAnalysis(t, binary, dir, "mutate", "--symbol", "main.add", "--max-mutants", "1")
	if code != 0 || !mutants.OK {
		t.Fatalf("mutate %+v", mutants)
	}
	mr := mutants.Data.(map[string]any)["result"].(map[string]any)
	if mr["killed"] != float64(1) || mr["invalid"] != float64(0) {
		t.Fatal(mr)
	}
	rules := filepath.Join(t.TempDir(), "rules.json")
	write(t, rules, `{"schema":"kiro.architecture.v1","rules":[{"name":"no-add","from":"main.main","deny":["main.add"]}]}`)
	violated, code := nativeAnalysis(t, binary, dir, "rules", "--rules", rules)
	if code != 1 || violated.Diagnostics[0].Code != "KIRO_ARCHITECTURE" {
		t.Fatal(violated)
	}
	// No rules are discovered or executed by ordinary checking.
	if err := Run([]string{"check", dir}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(file)
	if string(raw) != analyzedProject {
		t.Fatal("analysis modified candidate source")
	}
	write(t, file, analyzedProject+"\n")
	stale, code := nativeAnalysis(t, binary, dir, "impact", "--symbol", "main.add", "--coverage", saved)
	if code != 1 || stale.OK {
		t.Fatal("stale coverage accepted")
	}
	budget, code := nativeAnalysis(t, binary, dir, "coverage", "--max-steps", "1")
	if code != 1 || budget.OK {
		t.Fatal("exhausted test budget accepted")
	}
	timeout, code := nativeAnalysis(t, binary, dir, "coverage", "--timeout", "1ns")
	if code != 1 || timeout.Data.(map[string]any)["result"].(map[string]any)["status"] != "timeout" {
		t.Fatal(timeout)
	}
	write(t, file, strings.Replace(analyzedProject, "add(value:2,amount:3),expected:5", "add(value:2,amount:0),expected:2", 1))
	survivor, code := nativeAnalysis(t, binary, dir, "mutate", "--symbol", "main.add", "--max-mutants", "1", "--fail-on-survivor")
	if code != 1 || survivor.Diagnostics[0].Code != "KIRO_MUTATION_SURVIVOR" {
		t.Fatalf("mutation hook: %+v", survivor)
	}

}
func TestAnalysisFlagsAndBudgets(t *testing.T) {
	for _, args := range [][]string{{"graph", ".", "--rules", "x"}, {"coverage", ".", "--max-steps", "0"}, {"mutate", ".", "--symbol", "x", "--max-mutants", "101"}, {"slice", "."}, {"rules", "."}, {"boundaries", ".", "--max-nodes", "501"}, {"graph", ".", "--unknown"}} {
		if _, err := parseAnalysisOptions(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
