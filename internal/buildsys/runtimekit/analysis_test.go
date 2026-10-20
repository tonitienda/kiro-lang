package runtimekit

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func observationSpec() ProgramSpec {
	return ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Funcs: []FuncSpec{
		{Module: "main", Name: "main", ReturnType: "i32", BlockBody: true, Body: "return 0"},
		{Module: "main", Name: "leaf", Params: []ParamSpec{{Name: "value", Type: "i32"}}, ReturnType: "R[i32,str]", BlockBody: true, Body: `when (value) {0 => {return Err(error:"zero")}
_ => {return Ok(value:value)}}`},
		{Module: "main", Name: "parent", Params: []ParamSpec{{Name: "value", Type: "i32"}}, ReturnType: "R[i32,str]", BlockBody: true, Body: "let number = leaf(value:value)?\nreturn Ok(value:number+1)"},
		{Module: "main", Name: "text", ReturnType: "R[str,str]", BlockBody: true, Body: `return Ok(value:"${leaf(value:0)?}")`},
		{Module: "main", Name: "test_success", ReturnType: "nil", BlockBody: true, Body: "test.eq(actual:parent(value:2),expected:Ok(value:3))\nreturn nil"},
		{Module: "main", Name: "test_error", ReturnType: "nil", BlockBody: true, Body: `test.eq(actual:parent(value:0),expected:Err(error:"zero"))
test.eq(actual:text(),expected:Err(error:"zero"))
return nil`},
	}}}}
}
func TestKiroCoverageAndPerTestAttribution(t *testing.T) {
	spec := observationSpec()
	report, err := RunCoverage(spec, 1000)
	if err != nil || !report.Passed || !report.Complete || len(report.Tests) != 2 {
		t.Fatalf("%+v %v", report, err)
	}
	ok, failed, interpolation := false, false, false
	for _, point := range report.Points {
		if point.Function == "main.parent" && point.Outcome == "unwrap_ok" {
			ok = point.Hits == 1 && strings.Join(point.Tests, ",") == "main.test_success"
		}
		if point.Function == "main.parent" && point.Outcome == "unwrap_err" {
			failed = point.Hits == 1 && strings.Join(point.Tests, ",") == "main.test_error"
		}
		if point.Function == "main.text" && point.Outcome == "unwrap_err" {
			interpolation = point.Hits == 1 && strings.Contains(point.ID, "interp0")
		}
	}
	if !ok || !failed || !interpolation {
		raw, _ := json.Marshal(report)
		t.Fatalf("coverage missing paths: %s", raw)
	}
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if rt.coverage != nil {
		t.Fatal("ordinary runtime enabled tracing")
	}
	if _, err := rt.run(); err != nil {
		t.Fatal(err)
	}
	if len(rt.parserMemo) != 1 {
		t.Fatal("ordinary run parsed extra analysis functions")
	}
}
func TestCoverageFailureBudgetAndZeroTests(t *testing.T) {
	spec := observationSpec()
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs, FuncSpec{Module: "main", Name: "test_a_failure", BlockBody: true, ReturnType: "nil", Body: "test.eq(actual:1,expected:2)\nreturn nil"})
	report, err := RunCoverage(spec, 1000)
	if err != nil || report.Passed || len(report.Tests) != 3 || report.Tests[0].Status != "failed" {
		t.Fatalf("%+v %v", report.Tests, err)
	}
	report, err = RunCoverage(observationSpec(), 1)
	if err != nil || report.Passed || report.Tests[0].Status != "budget_exhausted" {
		t.Fatalf("%+v %v", report.Tests, err)
	}
	empty := observationSpec()
	empty.Modules[0].Funcs = empty.Modules[0].Funcs[:1]
	if _, err = RunCoverage(empty, 100); err == nil {
		t.Fatal("no-test result accepted as coverage")
	}
}
func TestMutationOperatorsDoNotRewriteStringsOrCaseArrows(t *testing.T) {
	spec := observationSpec()
	mutants, err := GenerateMutations(spec, "main.leaf", 10)
	if err != nil || len(mutants) != 0 {
		t.Fatalf("mutated case arrows or strings: %+v %v", mutants, err)
	}
	mutants, err = GenerateMutations(spec, "main.parent", 10)
	if err != nil || len(mutants) != 1 || mutants[0].Before != "+" || mutants[0].After != "-" {
		t.Fatalf("mutations %+v %v", mutants, err)
	}
	original := spec.Modules[0].Funcs[2].Body
	changed := MutatedSpec(spec, mutants[0])
	if spec.Modules[0].Funcs[2].Body != original {
		t.Fatal("mutated source spec in place")
	}
	report, err := RunCoverage(changed, 1000)
	if err != nil || report.Passed {
		t.Fatal("known arithmetic mutant survived")
	}
	if _, err := GenerateMutations(spec, "main.test_success", 2); err == nil {
		t.Fatal("allowed test mutation")
	}
}

func TestCoverageCallDepthIsInconclusive(t *testing.T) {
	spec := ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Funcs: []FuncSpec{
		{Module: "main", Name: "recur", ReturnType: "nil", BlockBody: true, Body: "return recur()"},
		{Module: "main", Name: "test_depth", ReturnType: "nil", BlockBody: true, Body: "return recur()"},
	}}}}
	report, err := RunCoverage(spec, 100000)
	if err != nil || report.Passed || len(report.Tests) != 1 || report.Tests[0].Status != "budget_exhausted" {
		t.Fatalf("%+v %v", report.Tests, err)
	}
}
