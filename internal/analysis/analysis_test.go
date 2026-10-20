package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/project"
)

const analysisSource = `mod main
import test
fn add(value:i32,amount:i32)->i32 {return value+amount}
fn apply(value:i32,transform:Fn[(value:i32)->i32])->i32 {return transform(value:value)}
fn use()->i32 {
 let alias = add
 let increment = partial alias(amount:2)
 return apply(value:increment(value:3),transform:increment)
}
fn sliced(seed:i32,unused:i32)->i32 {
 let unrelated = unused*99
 mut total = seed
 for value in [1,2] {total = total + value}
 when (total>0) {true => {total = total + 1}
 _ => {total = 0}}
 return total
}
fn test_use()->nil {test.eq(actual:use(),expected:7)
return nil}
fn main()->i32 {return use()}
`

func parsedGraph(t *testing.T) runtimekit.ProgramAnalysis {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.ki"), []byte(analysisSource), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := runtimekit.AnalyzeProgram(p.RuntimeSpec())
	if err != nil {
		t.Fatal(err)
	}
	return graph
}
func TestKnownCallbacksAndUnknownCalls(t *testing.T) {
	graph := parsedGraph(t)
	expected := map[runtimekit.DependencyEdge]bool{
		{From: "main.use", To: "main.add", Kind: "capture"}:  false,
		{From: "main.use", To: "main.add", Kind: "call"}:     false,
		{From: "main.use", To: "main.add", Kind: "callback"}: false,
		{From: "main.use", To: "main.apply", Kind: "call"}:   false,
	}
	unknown := false
	for _, edge := range graph.Edges {
		if _, ok := expected[edge]; ok {
			expected[edge] = true
		}
		if edge.From == "main.apply" && edge.Kind == "unresolved_call" {
			unknown = true
		}
	}
	for edge, seen := range expected {
		if !seen {
			t.Fatalf("missing edge %+v", edge)
		}
	}
	if !unknown {
		t.Fatal("dynamic call silently omitted")
	}
	impact, err := ImpactOf(graph, "main.add")
	if err != nil {
		t.Fatal(err)
	}
	if impact.Complete || !strings.Contains(strings.Join(impact.Tests, ","), "main.test_use") {
		t.Fatalf("impact %+v", impact)
	}
}
func TestBackwardSliceSelectsDependencies(t *testing.T) {
	graph := parsedGraph(t)
	slice, err := Slice(graph, "main.sliced", "return", "backward", false)
	if err != nil {
		t.Fatal(err)
	}
	texts := []string{}
	loopCarried := false
	for _, node := range slice.Nodes {
		texts = append(texts, node.Text)
		if node.Text == "unused" || strings.HasPrefix(node.Text, "unrelated =") {
			t.Fatalf("unrelated statement in slice: %s", node.Text)
		}
	}
	for _, edge := range slice.Edges {
		loopCarried = loopCarried || edge.Kind == "loop_carried"
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "seed") || !strings.Contains(joined, "total =") || !loopCarried {
		t.Fatalf("slice incomplete: %s %+v", joined, slice.Edges)
	}
	forward, err := Slice(graph, "main.sliced", "seed", "forward", false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, node := range forward.Nodes {
		found = found || node.ID == "main.sliced/return"
	}
	if !found {
		t.Fatal("forward slice missed result")
	}
}
func TestCyclesAndExplicitArchitectureRules(t *testing.T) {
	graph := runtimekit.ProgramAnalysis{Nodes: []runtimekit.AnalysisNode{{ID: "a.f", Kind: "function", Module: "a"}, {ID: "b.g", Kind: "function", Module: "b"}, {ID: "unresolved:x", Kind: "unresolved"}}, Edges: []runtimekit.DependencyEdge{{From: "a.f", To: "b.g", Kind: "call"}, {From: "b.g", To: "a.f", Kind: "call"}, {From: "a.f", To: "unresolved:x", Kind: "unresolved_call"}}}
	cycles := Cycles(graph)
	if len(cycles) != 2 {
		t.Fatalf("cycles %+v", cycles)
	}
	rules := ArchitectureRules{Schema: "kiro.architecture.v1", Rules: []ArchitectureRule{{Name: "no-b", From: "a.*", Deny: []string{"b.*"}, Kinds: []string{"call"}}}}
	report, err := CheckRules(graph, rules)
	if err != nil || report.Passed || report.Complete || len(report.Violations) != 1 {
		t.Fatalf("rules %+v %v", report, err)
	}
	rules.Rules = nil
	rules.Unresolved = "error"
	report, err = CheckRules(graph, rules)
	if err != nil || report.Passed {
		t.Fatal("strict unknown policy ignored")
	}
	rules.Unresolved = "report"
	report, err = CheckRules(graph, rules)
	if err != nil || !report.Passed || report.Complete {
		t.Fatal("unknown must remain visible")
	}
	rules.Rules = []ArchitectureRule{{Name: "bad", From: "[", Deny: []string{"*"}}}
	if _, err = CheckRules(graph, rules); err == nil {
		t.Fatal("invalid pattern accepted")
	}
}
func TestDeterministicBoundaryHeuristicAndBudget(t *testing.T) {
	graph := runtimekit.ProgramAnalysis{Nodes: []runtimekit.AnalysisNode{}, Edges: []runtimekit.DependencyEdge{}}
	for _, id := range []string{"a.f", "a.g", "a.h", "b.f", "b.g", "b.h"} {
		graph.Nodes = append(graph.Nodes, runtimekit.AnalysisNode{ID: id, Module: id[:1], Kind: "function"})
	}
	for _, prefix := range []string{"a.", "b."} {
		for _, pair := range [][2]string{{"f", "g"}, {"g", "h"}, {"h", "f"}} {
			graph.Edges = append(graph.Edges, runtimekit.DependencyEdge{From: prefix + pair[0], To: prefix + pair[1], Kind: "call"})
		}
	}
	report, err := Boundaries(graph, 20)
	if err != nil || len(report.Candidates) != 2 {
		t.Fatalf("boundaries %+v %v", report, err)
	}
	baseline, _ := json.Marshal(report)
	for i := 0; i < 30; i++ {
		again, err := Boundaries(graph, 20)
		if err != nil {
			t.Fatal(err)
		}
		bytes, _ := json.Marshal(again)
		if string(bytes) != string(baseline) {
			t.Fatal("nondeterministic analysis report")
		}
	}
	exceeded, err := Boundaries(graph, 2)
	if err == nil || exceeded.Status != "budget_exceeded" {
		t.Fatal("budget not enforced")
	}
}
