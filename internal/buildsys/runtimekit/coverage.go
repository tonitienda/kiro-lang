package runtimekit

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type CoveragePoint struct {
	ID       string   `json:"id"`
	Function string   `json:"function"`
	Kind     string   `json:"kind"`
	Outcome  string   `json:"outcome,omitempty"`
	BodyPath string   `json:"body_path,omitempty"`
	IsTest   bool     `json:"is_test"`
	Hits     int      `json:"hits"`
	Tests    []string `json:"tests"`
}
type TestObservation struct {
	Symbol string `json:"symbol"`
	Passed bool   `json:"passed"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Steps  int    `json:"steps"`
}
type CoverageTotals struct {
	Covered int `json:"covered"`
	Total   int `json:"total"`
}
type CoverageReport struct {
	Complete    bool              `json:"complete"`
	Schema      string            `json:"schema"`
	Passed      bool              `json:"passed"`
	Points      []CoveragePoint   `json:"points"`
	Tests       []TestObservation `json:"tests"`
	Functions   CoverageTotals    `json:"production_functions"`
	Statements  CoverageTotals    `json:"production_statements"`
	Branches    CoverageTotals    `json:"production_branches"`
	MaxSteps    int               `json:"max_steps_per_test"`
	Limitations []string          `json:"limitations"`
}
type executionBudget struct{}

func (*executionBudget) Error() string { return "analysis execution budget exhausted" }

type coverageRecorder struct {
	mu        sync.Mutex
	points    map[string]CoveragePoint
	nodes     map[any]map[string]string
	strings   map[*stringExpr]string
	hits      map[string]int
	steps     int
	maxSteps  int
	active    int
	pending   int
	exhausted bool
}

func newCoverageRecorder(maxSteps int) *coverageRecorder {
	return &coverageRecorder{points: map[string]CoveragePoint{}, nodes: map[any]map[string]string{}, strings: map[*stringExpr]string{}, hits: map[string]int{}, maxSteps: maxSteps}
}
func (r *coverageRecorder) point(fn FuncSpec, id, kind, outcome string, node any, label string) {
	r.points[id] = CoveragePoint{ID: id, Function: FunctionID(fn), Kind: kind, Outcome: outcome, BodyPath: strings.TrimPrefix(id, FunctionID(fn)+"/"), IsTest: strings.HasPrefix(fn.Name, "test_"), Tests: []string{}}
	if node != nil {
		if r.nodes[node] == nil {
			r.nodes[node] = map[string]string{}
		}
		r.nodes[node][label] = id
	}
}
func (r *coverageRecorder) statements(fn FuncSpec, statements []stmt, path string) {
	for i, statement := range statements {
		id := path + "/s" + strconv.Itoa(i)
		r.point(fn, id, "statement", "", statement, "statement")
		switch n := statement.(type) {
		case *letStmt:
			r.expression(fn, n.Value, id+"/value")
		case *assignStmt:
			r.expression(fn, n.Value, id+"/value")
		case *returnStmt:
			r.expression(fn, n.Value, id+"/value")
		case *exprStmt:
			r.expression(fn, n.Value, id+"/value")
		case *groupStmt:
			r.statements(fn, n.Body, id+"/group")
		case *whenStmt:
			r.when(fn, n.Value, n.Cases, id)
		case *forStmt:
			r.expression(fn, n.Items, id+"/items")
			r.point(fn, id+"/body", "branch", "loop_body", n, "body")
			r.point(fn, id+"/exit", "branch", "loop_exit", n, "exit")
			r.statements(fn, n.Body, id+"/loop")
		}
	}
}
func (r *coverageRecorder) when(fn FuncSpec, subject expr, cases []whenCase, path string) {
	r.expression(fn, subject, path+"/subject")
	for i := range cases {
		arm := &cases[i]
		id := path + "/arm" + strconv.Itoa(i)
		r.point(fn, id, "branch", "when_arm", arm, "arm")
		if arm.Expr != nil {
			r.expression(fn, arm.Expr, id+"/value")
		} else {
			r.statements(fn, arm.Body, id)
		}
	}
	if !whenCovers(cases) {
		r.point(fn, path+"/no_match", "branch", "when_no_match", subject, "no_match")
	}
}
func (r *coverageRecorder) expression(fn FuncSpec, e expr, path string) {
	switch n := e.(type) {
	case *callExpr:
		r.expression(fn, n.Callee, path+"/callee")
		for i, arg := range n.Args {
			r.expression(fn, arg.Value, fmt.Sprintf("%s/arg%d", path, i))
		}
	case *partialExpr:
		r.expression(fn, n.Call, path)
	case *selectorExpr:
		r.expression(fn, n.Left, path+"/left")
	case *binaryExpr:
		r.expression(fn, n.Left, path+"/left")
		r.expression(fn, n.Right, path+"/right")
	case *unwrapExpr:
		r.expression(fn, n.Value, path+"/unwrap")
		r.point(fn, path+"/ok", "branch", "unwrap_ok", n, "ok")
		r.point(fn, path+"/err", "branch", "unwrap_err", n, "err")
	case *whenExpr:
		r.when(fn, n.Value, n.Cases, path)
	case *listExpr:
		for i, item := range n.Items {
			r.expression(fn, item, fmt.Sprintf("%s/i%d", path, i))
		}
	case *structExpr:
		for i, f := range n.Fields {
			r.expression(fn, f.Value, fmt.Sprintf("%s/f%d", path, i))
		}
	case *sequenceExpr:
		for i, item := range n.Items {
			r.expression(fn, item, fmt.Sprintf("%s/e%d", path, i))
		}
	case *spawnExpr:
		r.expression(fn, n.Value, path+"/spawn")
	case *awaitExpr:
		r.expression(fn, n.Value, path+"/await")
	case *stringExpr:
		r.strings[n] = path
		index := 0
		_, _ = interpolateText(n.Value, func(raw string) (string, error) {
			if e, err := interpolationExpr(raw); err == nil {
				r.expression(fn, e, fmt.Sprintf("%s/interp%d", path, index))
			}
			index++
			return "", nil
		})
	}
}
func (r *coverageRecorder) step() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps++
	if r.steps > r.maxSteps {
		r.exhausted = true
		return &executionBudget{}
	}
	return nil
}
func (r *coverageRecorder) enter(fn FuncSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps++
	if r.steps > r.maxSteps || r.active >= 256 {
		r.exhausted = true
		return &executionBudget{}
	}
	r.active++
	r.hits[FunctionID(fn)]++
	return nil
}
func (r *coverageRecorder) leave() { r.mu.Lock(); r.active--; r.mu.Unlock() }
func (r *coverageRecorder) hit(node any, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id := r.nodes[node][label]; id != "" {
		r.hits[id]++
	}
}
func (ctx *execContext) trackedString(node *stringExpr) (value, error) {
	r := ctx.rt.coverage
	r.mu.Lock()
	path := r.strings[node]
	r.mu.Unlock()
	index := 0
	return interpolateText(node.Value, func(raw string) (string, error) {
		e, err := interpolationExpr(raw)
		if err != nil {
			return "", err
		}
		// Parsing interpolation creates fresh expression pointers. Register them under
		// the same structural IDs while keeping shared tracing maps race-free.
		r.mu.Lock()
		r.registerInterpolation(ctx.traceFunction, path, index, e)
		r.mu.Unlock()
		index++
		v, err := ctx.eval(e)
		if err != nil {
			return "", err
		}
		return fmt.Sprint(toPlain(v)), nil
	})
}
func (r *coverageRecorder) registerInterpolation(function, path string, index int, e expr) {
	// expression only needs a function's ID/test marker, never runtime contracts.
	module, name := "", function
	if dot := strings.Index(function, "."); dot >= 0 {
		module = function[:dot]
		name = function[dot+1:]
	}
	r.expression(FuncSpec{Module: module, Name: name}, e, fmt.Sprintf("%s/interp%d", path, index))
}

// RunCoverage executes each test in a fresh interpreter and continues after test
// failures. The CLI runs it in an isolated subprocess with an external timeout.
func RunCoverage(spec ProgramSpec, maxSteps int) (CoverageReport, error) {
	if maxSteps <= 0 {
		return CoverageReport{}, fmt.Errorf("max steps must be positive")
	}
	out := CoverageReport{Complete: true, Schema: AnalysisSchema, Passed: true, Points: []CoveragePoint{}, Tests: []TestObservation{}, MaxSteps: maxSteps, Limitations: []string{"Kiro function/statement/branch coverage, not Go interpreter line coverage", "tests are isolated; ordinary kiro test retains its existing shared-runtime semantics", "coverage records attempted statements and selected branches, not proof of assertions or correctness", "tests execute their declared effects; subprocess isolation is not a filesystem/network sandbox", "unjoined spawned tasks may outlive a test's observation window; no concurrency completeness claim"}}
	all := map[string]CoveragePoint{}
	tests := []FuncSpec{}
	for _, mod := range spec.Modules {
		for _, fn := range mod.Funcs {
			if strings.HasPrefix(fn.Name, "test_") {
				tests = append(tests, fn)
			}
		}
	}
	sort.Slice(tests, func(i, j int) bool { return FunctionID(tests[i]) < FunctionID(tests[j]) })
	if len(tests) == 0 {
		return out, fmt.Errorf("no test_* functions discovered")
	}
	for _, test := range tests {
		rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
		if err != nil {
			return out, err
		}
		recorder := newCoverageRecorder(maxSteps)
		for _, mod := range spec.Modules {
			for _, fn := range mod.Funcs {
				parsed, err := rt.parseFunc(fn)
				if err != nil {
					return out, err
				}
				recorder.point(fn, FunctionID(fn), "function", "", nil, "")
				recorder.statements(fn, parsed.stmts, FunctionID(fn)+"/body")
			}
		}
		rt.coverage = recorder
		v, err := rt.callUser(test, test.Module, nil, nil)
		if err == nil {
			_, err = terminalValue(v)
		}
		observation := TestObservation{Symbol: FunctionID(test), Passed: err == nil, Status: "passed"}
		if err != nil {
			observation.Status = "failed"
			observation.Error = err.Error()
			out.Passed = false
		}
		recorder.mu.Lock()
		// Budget failures remain inconclusive even if a task discards its error.
		if recorder.exhausted {
			observation.Passed = false
			observation.Status = "budget_exhausted"
			observation.Error = (&executionBudget{}).Error()
			out.Passed = false
		}
		observation.Steps = recorder.steps
		if recorder.active > 0 || recorder.pending > 0 {
			out.Complete = false
		}
		for id, point := range recorder.points {
			aggregate, exists := all[id]
			if !exists {
				aggregate = point
			}
			if hits := recorder.hits[id]; hits > 0 {
				aggregate.Hits += hits
				aggregate.Tests = append(aggregate.Tests, observation.Symbol)
			}
			all[id] = aggregate
		}
		recorder.mu.Unlock()
		out.Tests = append(out.Tests, observation)
	}
	for _, point := range all {
		out.Points = append(out.Points, point)
		if point.IsTest {
			continue
		}
		var total *CoverageTotals
		switch point.Kind {
		case "function":
			total = &out.Functions
		case "statement":
			total = &out.Statements
		case "branch":
			total = &out.Branches
		}
		if total != nil {
			total.Total++
			if point.Hits > 0 {
				total.Covered++
			}
		}
	}
	sort.Slice(out.Points, func(i, j int) bool { return out.Points[i].ID < out.Points[j].ID })
	return out, nil
}

func (r *coverageRecorder) spawn()        { r.mu.Lock(); r.pending++; r.mu.Unlock() }
func (r *coverageRecorder) completeTask() { r.mu.Lock(); r.pending--; r.mu.Unlock() }
