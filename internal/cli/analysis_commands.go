package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	graphanalysis "github.com/kiro-lang/kiro/internal/analysis"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/project"
)

type analysisOptions struct {
	failOnSurvivor                                              bool
	totalTimeout                                                time.Duration
	operation, entry, symbol, value, direction, rules, coverage string
	interprocedural                                             bool
	maxMutants, maxSteps, maxNodes                              int
	timeout                                                     time.Duration
	json                                                        bool
}

const analysisUsage = "usage: kiro analyze <graph|impact|metrics|cycles|boundaries|rules|slice|coverage|mutate> <path> [--symbol ID] [--value name|return] [--direction backward|forward] [--interprocedural] [--rules file] [--coverage file] [--max-mutants N] [--max-steps N] [--max-nodes N] [--timeout 10s] [--total-timeout 60s] [--fail-on-survivor] [--json]"

func parseAnalysisOptions(args []string) (analysisOptions, error) {
	opts := analysisOptions{value: "return", direction: "backward", maxMutants: 10, maxSteps: 100000, maxNodes: 200, timeout: 10 * time.Second, totalTimeout: 60 * time.Second, json: jsonRequested(args)}
	if len(args) < 2 {
		return opts, errors.New(analysisUsage)
	}
	opts.operation = args[0]
	opts.entry = args[1]
	allowed := map[string]map[string]bool{
		"graph": {"--symbol": true}, "impact": {"--symbol": true, "--coverage": true}, "metrics": {"--symbol": true}, "cycles": {}, "boundaries": {"--max-nodes": true}, "rules": {"--rules": true}, "slice": {"--symbol": true, "--value": true, "--direction": true, "--interprocedural": true}, "coverage": {"--timeout": true, "--max-steps": true}, "mutate": {"--symbol": true, "--max-mutants": true, "--total-timeout": true, "--fail-on-survivor": true, "--timeout": true, "--max-steps": true},
	}
	flags, ok := allowed[opts.operation]
	if !ok || strings.HasPrefix(opts.entry, "-") {
		return opts, errors.New(analysisUsage)
	}
	seen := map[string]bool{}
	for i := 2; i < len(args); i++ {
		flag := args[i]
		if flag == "--json" {
			continue
		}
		if !flags[flag] || seen[flag] {
			return opts, fmt.Errorf("invalid or repeated %s option %s", opts.operation, flag)
		}
		seen[flag] = true
		if flag == "--fail-on-survivor" {
			opts.failOnSurvivor = true
			continue
		}
		if flag == "--interprocedural" {
			opts.interprocedural = true
			continue
		}
		if i+1 >= len(args) {
			return opts, fmt.Errorf("missing value for %s", flag)
		}
		i++
		value := args[i]
		switch flag {
		case "--symbol":
			opts.symbol = value
		case "--value":
			opts.value = value
		case "--direction":
			opts.direction = value
		case "--rules":
			opts.rules = value
		case "--coverage":
			opts.coverage = value
		case "--timeout", "--total-timeout":
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 || duration > 10*time.Minute {
				return opts, fmt.Errorf("timeout must be positive and no greater than 10m")
			}
			if flag == "--timeout" {
				opts.timeout = duration
			} else {
				opts.totalTimeout = duration
			}
		default:
			number, err := strconv.Atoi(value)
			if err != nil || number <= 0 {
				return opts, fmt.Errorf("%s must be a positive integer", flag)
			}
			switch flag {
			case "--max-mutants":
				if number > 100 {
					return opts, fmt.Errorf("mutation budget cannot exceed 100")
				}
				opts.maxMutants = number
			case "--max-steps":
				if number > 10000000 {
					return opts, fmt.Errorf("step budget cannot exceed 10000000")
				}
				opts.maxSteps = number
			case "--max-nodes":
				if number > 500 {
					return opts, fmt.Errorf("boundary node budget cannot exceed 500")
				}
				opts.maxNodes = number
			}
		}
	}
	if (opts.operation == "impact" || opts.operation == "slice" || opts.operation == "mutate") && opts.symbol == "" {
		return opts, fmt.Errorf("%s requires --symbol", opts.operation)
	}
	if opts.operation == "rules" && opts.rules == "" {
		return opts, fmt.Errorf("rules requires an explicit --rules file")
	}
	return opts, nil
}
func runAnalyze(args []string) error {
	opts, err := parseAnalysisOptions(args)
	if err != nil {
		return commandFailure("analyze", args, err, "KIRO_USAGE")
	}
	p, err := project.Load(opts.entry)
	if err != nil {
		return commandFailure("analyze", args, err, "KIRO_CHECK_FAILED")
	}
	data := map[string]any{"schema": runtimekit.AnalysisSchema, "operation": opts.operation, "revision": p.Revision(), "execution": "explicit_opt_in", "runs_in_check": false}
	spec := p.RuntimeSpec()
	var result any
	if opts.operation == "coverage" {
		execution := executeAnalysisTests(spec, opts.maxSteps, opts.timeout)
		result = execution
		data["result"] = result
		if execution.Status != "passed" {
			return analysisFailure(data, opts, "KIRO_TEST_FAILED", "coverage test execution: "+execution.Status)
		}
		return outputReport("analyze", data, nil, opts.json)
	}
	if opts.operation == "mutate" {
		report, err := runMutations(spec, opts)
		data["result"] = report
		if err != nil {
			d := runtimekit.DiagnosticOf(err, "KIRO_ANALYSIS")
			return analysisFailure(data, opts, d.Code, err.Error())
		}
		return outputReport("analyze", data, nil, opts.json)
	}
	graph, err := runtimekit.AnalyzeProgram(spec)
	if err != nil {
		return commandFailure("analyze", args, err, "KIRO_ANALYSIS")
	}
	switch opts.operation {
	case "graph":
		if opts.symbol != "" {
			found := false
			keep := map[string]bool{opts.symbol: true}
			edges := []runtimekit.DependencyEdge{}
			for _, node := range graph.Nodes {
				found = found || node.ID == opts.symbol
			}
			if !found {
				err = fmt.Errorf("unknown graph symbol %s", opts.symbol)
			}
			for _, edge := range graph.Edges {
				if edge.From == opts.symbol || edge.To == opts.symbol {
					edges = append(edges, edge)
					keep[edge.From] = true
					keep[edge.To] = true
				}
			}
			nodes := []runtimekit.AnalysisNode{}
			for _, node := range graph.Nodes {
				if keep[node.ID] {
					nodes = append(nodes, node)
				}
			}
			operations := []runtimekit.AnalysisNode{}
			for _, op := range graph.Operations {
				if op.Function == opts.symbol {
					operations = append(operations, op)
				}
			}
			graph.Nodes = nodes
			graph.Edges = edges
			graph.Operations = operations
			graph.Metrics = nil
			graph.Dependences = nil
		}
		result = graph
	case "metrics":
		metrics := []runtimekit.FunctionMetrics{}
		for _, metric := range graph.Metrics {
			if opts.symbol == "" || metric.Symbol == opts.symbol {
				metrics = append(metrics, metric)
			}
		}
		if len(metrics) == 0 {
			err = fmt.Errorf("unknown metric symbol %s", opts.symbol)
		}
		result = map[string]any{"functions": metrics, "convention": "1 + loop decisions + when alternatives (including possible no-match) + result unwrap decisions; structural Kiro CFG estimate, not semantic difficulty"}
	case "cycles":
		result = graphanalysis.Cycles(graph)
	case "impact":
		impact, e := graphanalysis.ImpactOf(graph, opts.symbol)
		err = e
		view := map[string]any{"static": impact}
		if err == nil && opts.coverage != "" {
			var saved struct {
				Data struct {
					Revision string        `json:"revision"`
					Result   testExecution `json:"result"`
				} `json:"data"`
			}
			raw, e := os.ReadFile(opts.coverage)
			if e == nil {
				e = json.Unmarshal(raw, &saved)
			}
			if e != nil {
				err = e
			} else if saved.Data.Revision != p.Revision() {
				err = fmt.Errorf("coverage source revision does not match this project")
			} else if saved.Data.Result.Report.Schema != runtimekit.AnalysisSchema || saved.Data.Result.Status == "" {
				err = fmt.Errorf("expected a saved coverage analysis report")
			} else {
				observed := []string{}
				for _, point := range saved.Data.Result.Report.Points {
					if point.Function == opts.symbol && point.Kind == "function" {
						observed = append(observed, point.Tests...)
					}
				}
				view["observed_tests"] = observed
				view["coverage_status"] = saved.Data.Result.Status
			}
		}
		result = view
	case "boundaries":
		result, err = graphanalysis.Boundaries(graph, opts.maxNodes)
	case "rules":
		var rules graphanalysis.ArchitectureRules
		raw, e := os.Open(opts.rules)
		if e != nil {
			err = e
			break
		}
		decoder := json.NewDecoder(raw)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&rules)
		if err == nil {
			var tail any
			if e := decoder.Decode(&tail); e != io.EOF {
				err = fmt.Errorf("rules file must contain exactly one JSON object")
			}
		}
		raw.Close()
		if err != nil {
			break
		}
		report, e := graphanalysis.CheckRules(graph, rules)
		err = e
		result = report
		if err == nil && !report.Passed {
			data["result"] = report
			return analysisFailure(data, opts, "KIRO_ARCHITECTURE", "explicit architecture rules failed; inspect violations and unresolved edges")
		}
	case "slice":
		result, err = graphanalysis.Slice(graph, opts.symbol, opts.value, opts.direction, opts.interprocedural)
	}
	data["result"] = result
	if err != nil {
		code := "KIRO_ANALYSIS"
		if opts.operation == "boundaries" {
			code = "KIRO_ANALYSIS_BUDGET"
		}
		return analysisFailure(data, opts, code, err.Error())
	}
	return outputReport("analyze", data, nil, opts.json)
}
func analysisFailure(data any, opts analysisOptions, code, message string) error {
	return outputReport("analyze", data, []runtimekit.Diagnostic{*runtimekit.NewDiagnostic(code, message)}, opts.json)
}

type workerInput struct {
	Spec     runtimekit.ProgramSpec `json:"spec"`
	MaxSteps int                    `json:"max_steps"`
}
type testExecution struct {
	Status string                    `json:"status"`
	Report runtimekit.CoverageReport `json:"coverage"`
	Error  string                    `json:"error,omitempty"`
}

func runAnalysisWorker(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("analysis worker requires input and report files")
	}
	raw, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var input workerInput
	if err = json.Unmarshal(raw, &input); err != nil {
		return err
	}
	report, err := runtimekit.RunCoverage(input.Spec, input.MaxSteps)
	execution := testExecution{Status: "passed", Report: report}
	if err != nil {
		execution.Status = "error"
		execution.Error = err.Error()
	} else if !report.Complete {
		execution.Status = "incomplete"
	} else if !report.Passed {
		execution.Status = "failed"
		for _, test := range report.Tests {
			if test.Status == "budget_exhausted" {
				execution.Status = "budget_exhausted"
			}
		}
	}
	output, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	return os.WriteFile(args[1], output, 0600)
}
func executeAnalysisTests(spec runtimekit.ProgramSpec, maxSteps int, timeout time.Duration) testExecution {
	result := testExecution{Status: "worker_error"}
	dir, err := os.MkdirTemp("", "kiro-analysis-*")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer os.RemoveAll(dir)
	raw, err := json.Marshal(workerInput{spec, maxSteps})
	if err != nil {
		result.Error = err.Error()
		return result
	}
	input, output := filepath.Join(dir, "input.json"), filepath.Join(dir, "report.json")
	if err = os.WriteFile(input, raw, 0600); err != nil {
		result.Error = err.Error()
		return result
	}
	binary, err := os.Executable()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "__analysis-test", input, output)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			result.Status = "timeout"
			result.Error = ctx.Err().Error()
		} else {
			result.Error = err.Error()
		}
		return result
	}
	bytes, err := os.ReadFile(output)
	if err == nil {
		err = json.Unmarshal(bytes, &result)
	}
	if err != nil {
		result.Status = "worker_error"
		result.Error = err.Error()
	}
	return result
}

type mutantResult struct {
	Mutation    runtimekit.Mutation    `json:"mutation"`
	Status      string                 `json:"status"`
	FailedTests []string               `json:"failed_tests"`
	Diagnostic  *runtimekit.Diagnostic `json:"diagnostic,omitempty"`
}
type mutationReport struct {
	Status       string         `json:"status"`
	Unexecuted   int            `json:"unexecuted"`
	Symbol       string         `json:"symbol"`
	Baseline     testExecution  `json:"baseline"`
	Mutants      []mutantResult `json:"mutants"`
	Killed       int            `json:"killed"`
	Survived     int            `json:"survived"`
	Invalid      int            `json:"invalid"`
	Inconclusive int            `json:"inconclusive"`
	Limit        int            `json:"max_mutants"`
	Selection    string         `json:"selection"`
}

func runMutations(spec runtimekit.ProgramSpec, opts analysisOptions) (mutationReport, error) {
	report := mutationReport{Status: "complete", Symbol: opts.symbol, Mutants: []mutantResult{}, Limit: opts.maxMutants, Selection: "bounded source-order operator mutations inside the explicitly selected covered function; equivalent mutants are not automatically identified"}
	mutants, err := runtimekit.GenerateMutations(spec, opts.symbol, opts.maxMutants)
	if err != nil {
		return report, err
	}
	if len(mutants) == 0 {
		return report, fmt.Errorf("no supported operator mutations in target; choose an exercised function containing arithmetic, comparisons or booleans")
	}
	deadline := time.Now().Add(opts.totalTimeout)
	report.Baseline = executeAnalysisTests(spec, opts.maxSteps, min(opts.timeout, opts.totalTimeout))
	if report.Baseline.Status != "passed" {
		return report, fmt.Errorf("mutation baseline must pass; got %s", report.Baseline.Status)
	}
	covered := false
	for _, point := range report.Baseline.Report.Points {
		if point.ID == opts.symbol && point.Kind == "function" && point.Hits > 0 {
			covered = true
		}
	}
	if !covered {
		return report, fmt.Errorf("mutation target is not exercised by the baseline tests")
	}
	for index, mutation := range mutants {
		if time.Until(deadline) <= 0 {
			report.Status = "budget_exhausted"
			report.Unexecuted = len(mutants) - index
			return report, runtimekit.NewDiagnostic("KIRO_ANALYSIS_BUDGET", "mutation total-time budget exhausted")
		}
		result := mutantResult{Mutation: mutation, FailedTests: []string{}}
		changed := runtimekit.MutatedSpec(spec, mutation)
		if err := runtimekit.ValidateFunctionalContracts(changed); err != nil {
			d := runtimekit.DiagnosticOf(err, "KIRO_CHECK_FAILED")
			result.Status = "invalid"
			result.Diagnostic = &d
			report.Invalid++
		} else {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				report.Status = "budget_exhausted"
				report.Unexecuted = len(mutants) - index
				return report, runtimekit.NewDiagnostic("KIRO_ANALYSIS_BUDGET", "mutation total-time budget exhausted")
			}
			execution := executeAnalysisTests(changed, opts.maxSteps, min(opts.timeout, remaining))
			switch execution.Status {
			case "passed":
				result.Status = "survived"
				report.Survived++
			case "failed":
				result.Status = "killed"
				report.Killed++
				for _, test := range execution.Report.Tests {
					if !test.Passed {
						result.FailedTests = append(result.FailedTests, test.Symbol)
					}
				}
			default:
				result.Status = execution.Status
				report.Inconclusive++
			}
		}
		report.Mutants = append(report.Mutants, result)
	}
	if opts.failOnSurvivor && report.Survived > 0 {
		return report, runtimekit.NewDiagnostic("KIRO_MUTATION_SURVIVOR", "targeted mutations survived; inspect equivalents and assertions before deciding on repairs")
	}
	return report, nil
}
