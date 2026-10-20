// Package analysis implements opt-in graph algorithms; compiler checking never
// imports or invokes these algorithms.
package analysis

import (
	"fmt"
	"math"
	"path"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

type Impact struct {
	Target     string                      `json:"target"`
	Affected   []string                    `json:"affected"`
	Tests      []string                    `json:"candidate_tests"`
	Unresolved []runtimekit.DependencyEdge `json:"unresolved"`
	Complete   bool                        `json:"all_call_sites_resolved"`
}

func ImpactOf(g runtimekit.ProgramAnalysis, target string) (Impact, error) {
	known := false
	tests := map[string]bool{}
	for _, n := range g.Nodes {
		known = known || n.ID == target
		if n.IsTest {
			tests[n.ID] = true
		}
	}
	if !known {
		return Impact{}, fmt.Errorf("unknown analysis symbol %s", target)
	}
	out := Impact{Target: target, Affected: []string{}, Tests: []string{}, Unresolved: []runtimekit.DependencyEdge{}, Complete: true}
	seen := map[string]bool{target: true}
	queue := []string{target}
	reverse := map[string][]string{}
	for _, e := range g.Edges {
		if e.Kind == "unresolved_call" {
			out.Unresolved = append(out.Unresolved, e)
			out.Complete = false
		}
		reverse[e.To] = append(reverse[e.To], e.From)
	}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, caller := range reverse[next] {
			if !seen[caller] {
				seen[caller] = true
				queue = append(queue, caller)
			}
		}
	}
	for id := range seen {
		out.Affected = append(out.Affected, id)
		if tests[id] {
			out.Tests = append(out.Tests, id)
		}
	}
	sort.Strings(out.Affected)
	sort.Strings(out.Tests)
	return out, nil
}

type Cycle struct {
	Scope   string                      `json:"scope"`
	Members []string                    `json:"members"`
	Edges   []runtimekit.DependencyEdge `json:"edges"`
}

func Cycles(g runtimekit.ProgramAnalysis) []Cycle {
	nodes := map[string]runtimekit.AnalysisNode{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	out := []Cycle{}
	for _, scope := range []string{"function", "module"} {
		graph := map[string][]string{}
		edges := []runtimekit.DependencyEdge{}
		for _, e := range g.Edges {
			from, to := nodes[e.From], nodes[e.To]
			if scope == "function" {
				if from.Kind != "function" || to.Kind != "function" || (e.Kind != "call" && e.Kind != "callback") {
					continue
				}
			} else {
				if from.Module == "" || to.Module == "" || from.Module == to.Module {
					continue
				}
			}
			edge := e
			if scope == "module" {
				edge.From = from.Module
				edge.To = to.Module
			}
			graph[edge.From] = append(graph[edge.From], edge.To)
			if graph[edge.To] == nil {
				graph[edge.To] = []string{}
			}
			edges = append(edges, edge)
		}
		for _, component := range components(graph) {
			self := false
			set := map[string]bool{}
			for _, id := range component {
				set[id] = true
			}
			selected := []runtimekit.DependencyEdge{}
			for _, e := range edges {
				if set[e.From] && set[e.To] {
					selected = append(selected, e)
					self = self || e.From == e.To
				}
			}
			if len(component) > 1 || self {
				out = append(out, Cycle{scope, component, selected})
			}
		}
	}
	return out
}
func components(graph map[string][]string) [][]string {
	index := 0
	indices, low := map[string]int{}, map[string]int{}
	stack := []string{}
	onStack := map[string]bool{}
	out := [][]string{}
	var visit func(string)
	visit = func(v string) {
		index++
		indices[v] = index
		low[v] = index
		stack = append(stack, v)
		onStack[v] = true
		sort.Strings(graph[v])
		for _, w := range graph[v] {
			if indices[w] == 0 {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], indices[w])
			}
		}
		if low[v] == indices[v] {
			part := []string{}
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				part = append(part, w)
				if w == v {
					break
				}
			}
			sort.Strings(part)
			out = append(out, part)
		}
	}
	names := []string{}
	for id := range graph {
		names = append(names, id)
	}
	sort.Strings(names)
	for _, id := range names {
		if indices[id] == 0 {
			visit(id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Join(out[i], "\x00") < strings.Join(out[j], "\x00") })
	return out
}

type ArchitectureRules struct {
	Schema     string             `json:"schema"`
	Rules      []ArchitectureRule `json:"rules"`
	Unresolved string             `json:"unresolved"`
}
type ArchitectureRule struct {
	Name  string   `json:"name"`
	From  string   `json:"from"`
	Deny  []string `json:"deny"`
	Kinds []string `json:"kinds"`
}
type RuleViolation struct {
	Rule string                    `json:"rule"`
	Edge runtimekit.DependencyEdge `json:"edge"`
}
type RuleReport struct {
	Violations []RuleViolation             `json:"violations"`
	Unresolved []runtimekit.DependencyEdge `json:"unresolved"`
	Complete   bool                        `json:"all_call_sites_resolved"`
	Passed     bool                        `json:"passed"`
}

func CheckRules(g runtimekit.ProgramAnalysis, rules ArchitectureRules) (RuleReport, error) {
	out := RuleReport{Violations: []RuleViolation{}, Unresolved: []runtimekit.DependencyEdge{}, Complete: true, Passed: true}
	if rules.Schema != "kiro.architecture.v1" {
		return out, fmt.Errorf("rules schema must be kiro.architecture.v1")
	}
	if rules.Unresolved != "" && rules.Unresolved != "report" && rules.Unresolved != "error" {
		return out, fmt.Errorf("unresolved must be report or error")
	}
	names := map[string]bool{}
	for _, rule := range rules.Rules {
		if rule.Name == "" || rule.From == "" || len(rule.Deny) == 0 || names[rule.Name] {
			return out, fmt.Errorf("each rule needs a unique name, from pattern and nonempty deny patterns")
		}
		names[rule.Name] = true
		for _, pattern := range append([]string{rule.From}, rule.Deny...) {
			if _, err := path.Match(pattern, ""); err != nil {
				return out, err
			}
		}
		for _, kind := range rule.Kinds {
			if kind != "call" && kind != "callback" && kind != "capture" && kind != "type" && kind != "read" && kind != "effect" && kind != "reference" {
				return out, fmt.Errorf("unknown rule edge kind %s", kind)
			}
		}
		for _, edge := range g.Edges {
			match, _ := path.Match(rule.From, edge.From)
			if !match || edge.Kind == "unresolved_call" {
				continue
			}
			if len(rule.Kinds) > 0 {
				allowed := false
				for _, k := range rule.Kinds {
					allowed = allowed || k == edge.Kind
				}
				if !allowed {
					continue
				}
			}
			for _, deny := range rule.Deny {
				match, _ = path.Match(deny, edge.To)
				if match {
					out.Violations = append(out.Violations, RuleViolation{rule.Name, edge})
					break
				}
			}
		}
	}
	for _, edge := range g.Edges {
		if edge.Kind == "unresolved_call" {
			out.Unresolved = append(out.Unresolved, edge)
			out.Complete = false
		}
	}
	out.Passed = len(out.Violations) == 0 && (rules.Unresolved != "error" || out.Complete)
	return out, nil
}

type BoundaryCandidate struct {
	Symbols        []string                    `json:"symbols"`
	Modules        []string                    `json:"current_modules"`
	EntryPoints    []string                    `json:"entry_points"`
	Boundary       []runtimekit.DependencyEdge `json:"boundary_edges"`
	InternalWeight float64                     `json:"internal_weight"`
	Explanation    string                      `json:"explanation"`
}
type BoundaryReport struct {
	Algorithm  string              `json:"algorithm"`
	Objective  string              `json:"objective"`
	Candidates []BoundaryCandidate `json:"candidates"`
	Unresolved int                 `json:"unresolved_calls"`
	Nodes      int                 `json:"nodes"`
	Budget     int                 `json:"node_budget"`
	Status     string              `json:"status"`
}

func Boundaries(g runtimekit.ProgramAnalysis, budget int) (BoundaryReport, error) {
	out := BoundaryReport{Algorithm: "deterministic_greedy_modularity", Objective: "weighted undirected modularity with degree-normalized affinities; heuristic, not an architectural proof", Candidates: []BoundaryCandidate{}, Budget: budget, Status: "complete"}
	nodes := map[string]runtimekit.AnalysisNode{}
	active := map[string]bool{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
		if n.Kind == "function" && !n.IsTest {
			active[n.ID] = true
		}
	}
	for _, e := range g.Edges {
		if active[e.From] && (nodes[e.To].Kind == "record" || nodes[e.To].Kind == "variant_type" || nodes[e.To].Kind == "constructor") && nodes[e.To].Module != "http" {
			active[e.To] = true
		}
		if e.Kind == "unresolved_call" {
			out.Unresolved++
		}
	}
	out.Nodes = len(active)
	if len(active) > budget {
		out.Status = "budget_exceeded"
		return out, fmt.Errorf("boundary analysis has %d nodes, exceeding explicit budget %d", len(active), budget)
	}
	ids := []string{}
	for id := range active {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	weights := map[[2]string]float64{}
	degree := map[string]float64{}
	for _, e := range g.Edges {
		if !active[e.From] || !active[e.To] || e.From == e.To {
			continue
		}
		weight := 1.0
		if e.Kind == "call" {
			weight = 3
		} else if e.Kind == "callback" || e.Kind == "capture" {
			weight = 2
		}
		x, y := e.From, e.To
		if x > y {
			x, y = y, x
		}
		weights[[2]string{x, y}] += weight
		degree[x] += weight
		degree[y] += weight
	}
	normalized := map[[2]string]float64{}
	d := map[string]float64{}
	total := 0.0
	pairs := [][2]string{}
	for pair := range weights {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}
		return pairs[i][1] < pairs[j][1]
	})
	for _, pair := range pairs {
		w := weights[pair]
		w /= math.Sqrt(degree[pair[0]] * degree[pair[1]])
		normalized[pair] = w
		d[pair[0]] += w
		d[pair[1]] += w
		total += w
	}
	if total == 0 {
		return out, nil
	}
	clusters := [][]string{}
	for _, id := range ids {
		clusters = append(clusters, []string{id})
	}
	for {
		bestI, bestJ := -1, -1
		bestGain := 1e-12
		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				between, di, dj := 0.0, 0.0, 0.0
				for _, x := range clusters[i] {
					di += d[x]
					for _, y := range clusters[j] {
						p := [2]string{x, y}
						if x > y {
							p = [2]string{y, x}
						}
						between += normalized[p]
					}
				}
				for _, y := range clusters[j] {
					dj += d[y]
				}
				gain := between/total - di*dj/(2*total*total)
				if gain > bestGain {
					bestI, bestJ, bestGain = i, j, gain
				}
			}
		}
		if bestI < 0 {
			break
		}
		clusters[bestI] = append(clusters[bestI], clusters[bestJ]...)
		sort.Strings(clusters[bestI])
		clusters = append(clusters[:bestJ], clusters[bestJ+1:]...)
	}
	for _, cluster := range clusters {
		if len(cluster) < 3 {
			continue
		}
		set := map[string]bool{}
		mods := map[string]bool{}
		entries := map[string]bool{}
		for _, id := range cluster {
			set[id] = true
			mods[nodes[id].Module] = true
		}
		candidate := BoundaryCandidate{Symbols: cluster, Modules: []string{}, EntryPoints: []string{}, Boundary: []runtimekit.DependencyEdge{}, Explanation: "candidate group with relatively strong structural affinity; inspect policy/data ownership and unresolved calls before extracting a module"}
		for _, pair := range pairs {
			w := normalized[pair]
			if set[pair[0]] && set[pair[1]] {
				candidate.InternalWeight += w
			}
		}
		for _, e := range g.Edges {
			if set[e.From] != set[e.To] {
				candidate.Boundary = append(candidate.Boundary, e)
				if set[e.To] && nodes[e.From].Kind == "function" && !nodes[e.From].IsTest {
					entries[e.To] = true
				}
			}
		}
		for module := range mods {
			candidate.Modules = append(candidate.Modules, module)
		}
		for id := range entries {
			candidate.EntryPoints = append(candidate.EntryPoints, id)
		}
		sort.Strings(candidate.Modules)
		sort.Strings(candidate.EntryPoints)
		out.Candidates = append(out.Candidates, candidate)
	}
	sort.Slice(out.Candidates, func(i, j int) bool {
		return strings.Join(out.Candidates[i].Symbols, "\x00") < strings.Join(out.Candidates[j].Symbols, "\x00")
	})
	return out, nil
}

type SliceReport struct {
	Symbol          string                      `json:"symbol"`
	Value           string                      `json:"value"`
	Direction       string                      `json:"direction"`
	Nodes           []runtimekit.AnalysisNode   `json:"nodes"`
	Edges           []runtimekit.DependencyEdge `json:"edges"`
	Boundaries      []string                    `json:"call_boundaries"`
	Interprocedural bool                        `json:"interprocedural"`
	Limitations     []string                    `json:"limitations"`
}

func Slice(g runtimekit.ProgramAnalysis, symbol, value, direction string, interprocedural bool) (SliceReport, error) {
	out := SliceReport{Symbol: symbol, Value: value, Direction: direction, Interprocedural: interprocedural, Nodes: []runtimekit.AnalysisNode{}, Edges: []runtimekit.DependencyEdge{}, Boundaries: []string{}, Limitations: g.Limitations}
	if direction != "backward" && direction != "forward" {
		return out, fmt.Errorf("slice direction must be backward or forward")
	}
	nodes := map[string]runtimekit.AnalysisNode{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	for _, n := range g.Operations {
		nodes[n.ID] = n
	}
	roots := []string{}
	if value == "return" {
		id := symbol + "/return"
		if _, ok := nodes[id]; ok {
			roots = append(roots, id)
		}
	} else {
		for _, n := range g.Operations {
			if n.Function == symbol && (n.Text == value || strings.HasPrefix(n.Text, value+" = ")) {
				roots = append(roots, n.ID)
			}
		}
	}
	if len(roots) == 0 {
		return out, fmt.Errorf("no slice criterion %s in %s", value, symbol)
	}
	edges := append([]runtimekit.DependencyEdge{}, g.Dependences...)
	if interprocedural {
		for _, n := range g.Nodes {
			if n.Kind == "function" {
				edges = append(edges, runtimekit.DependencyEdge{From: n.ID, To: n.ID + "/return", Kind: "callee_result"})
			}
		}
	}
	adjacency := map[string][]string{}
	for _, e := range edges {
		if direction == "backward" {
			adjacency[e.From] = append(adjacency[e.From], e.To)
		} else {
			adjacency[e.To] = append(adjacency[e.To], e.From)
		}
	}
	seen := map[string]bool{}
	queue := roots
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		queue = append(queue, adjacency[id]...)
	}
	for id := range seen {
		if n, ok := nodes[id]; ok {
			out.Nodes = append(out.Nodes, n)
			if n.Kind == "function" || n.Kind == "builtin" || n.Kind == "unresolved" {
				out.Boundaries = append(out.Boundaries, id)
			}
		}
	}
	for _, e := range edges {
		if seen[e.From] && seen[e.To] {
			out.Edges = append(out.Edges, e)
		}
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	sort.Strings(out.Boundaries)
	return out, nil
}
