package runtimekit

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const AnalysisSchema = "kiro.analysis.v1"

type AnalysisNode struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Module   string `json:"module,omitempty"`
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	BodyPath string `json:"body_path,omitempty"`
	Text     string `json:"text,omitempty"`
	IsTest   bool   `json:"is_test,omitempty"`
}
type DependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}
type FunctionMetrics struct {
	Symbol     string `json:"symbol"`
	Statements int    `json:"statements"`
	Decisions  int    `json:"decisions"`
	Cyclomatic int    `json:"cyclomatic_complexity"`
	Nesting    int    `json:"max_nesting"`
	FanIn      int    `json:"fan_in"`
	FanOut     int    `json:"fan_out"`
	Unresolved int    `json:"unresolved_calls"`
	IsTest     bool   `json:"is_test"`
}
type ProgramAnalysis struct {
	Schema      string            `json:"schema"`
	Nodes       []AnalysisNode    `json:"nodes"`
	Edges       []DependencyEdge  `json:"edges"`
	Operations  []AnalysisNode    `json:"operations"`
	Dependences []DependencyEdge  `json:"dependences"`
	Metrics     []FunctionMetrics `json:"metrics"`
	Limitations []string          `json:"limitations"`
}

type abstractValue struct {
	typ     string
	targets []string
	unknown bool
	defs    []string
}
type analysisEnv map[string]abstractValue

func cloneAnalysisEnv(src analysisEnv) analysisEnv {
	out := analysisEnv{}
	for k, v := range src {
		out[k] = v
	}
	return out
}
func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	for _, s := range items {
		seen[s] = true
	}
	out := []string{}
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

type programAnalyzer struct {
	rt          *runtime
	out         ProgramAnalysis
	nodes       map[string]AnalysisNode
	edges       map[DependencyEdge]bool
	dependences map[DependencyEdge]bool
	fn          FuncSpec
	metric      FunctionMetrics
	env         analysisEnv
	controls    []string
	current     string
	reads       map[string]map[string]bool
}

func AnalyzeProgram(spec ProgramSpec) (ProgramAnalysis, error) {
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		return ProgramAnalysis{}, err
	}
	a := &programAnalyzer{rt: rt, nodes: map[string]AnalysisNode{}, edges: map[DependencyEdge]bool{}, dependences: map[DependencyEdge]bool{}}
	a.out = ProgramAnalysis{Schema: AnalysisSchema, Nodes: []AnalysisNode{}, Edges: []DependencyEdge{}, Operations: []AnalysisNode{}, Dependences: []DependencyEdge{}, Metrics: []FunctionMetrics{}, Limitations: []string{
		"known targets and explicit callback dependencies; unresolved dynamic calls stay visible",
		"no interprocedural points-to or heap/alias analysis; no inference of reflection or external behavior",
		"slicing is conservative local def-use/control analysis with call boundaries, not a whole-program semantic equivalence proof",
		"body paths identify executable AST nodes, not exact source spans; use symbols for declaration locations",
	}}
	for _, m := range spec.Modules {
		for _, t := range m.Types {
			id := m.Name + "." + t.Name
			kind := "record"
			if len(t.Variants) > 0 {
				kind = "variant_type"
			}
			a.node(AnalysisNode{ID: id, Kind: kind, Module: m.Name})
			for _, field := range t.Fields {
				a.typeEdges(id, field.Type, m.Name)
			}
			for _, v := range t.Variants {
				tag := id + "." + v.Name
				a.node(AnalysisNode{ID: tag, Kind: "constructor", Module: m.Name})
				a.edge(tag, id, "returns")
				for _, f := range v.Fields {
					a.typeEdges(tag, f.Type, m.Name)
				}
			}
		}
		for _, c := range m.Consts {
			a.node(AnalysisNode{ID: m.Name + "." + c.Name, Kind: "constant", Module: m.Name, Text: "const " + c.Name + " = " + c.Value})
		}
		for _, fn := range m.Funcs {
			if _, exists := a.nodes[FunctionID(fn)]; exists {
				return ProgramAnalysis{}, fmt.Errorf("ambiguous analysis symbol %s; rename duplicate or colliding declarations", FunctionID(fn))
			}
			a.node(AnalysisNode{ID: FunctionID(fn), Kind: "function", Module: m.Name, File: fn.File, IsTest: strings.HasPrefix(fn.Name, "test_")})
		}
	}
	for _, m := range spec.Modules {
		for _, fn := range m.Funcs {
			parsed, err := rt.parseFunc(fn)
			if err != nil {
				return ProgramAnalysis{}, FunctionDiagnostic(err, fn, "KIRO_SYNTAX")
			}
			a.fn = fn
			a.current = ""
			a.metric = FunctionMetrics{Symbol: FunctionID(fn), Cyclomatic: 1, IsTest: strings.HasPrefix(fn.Name, "test_")}
			a.env = analysisEnv{}
			a.controls = nil
			a.reads = map[string]map[string]bool{}
			for _, p := range fn.Params {
				typ := rt.qualifyType(p.Type, fn.Module)
				id := FunctionID(fn) + "/param/" + p.Name
				a.op(id, "parameter", p.Name)
				a.env[p.Name] = abstractValue{typ: typ, unknown: strings.HasPrefix(typ, "Fn["), defs: []string{id}}
				a.typeEdges(FunctionID(fn), typ, fn.Module)
			}
			a.typeEdges(FunctionID(fn), fn.ReturnType, fn.Module)
			for _, effect := range fn.Effects {
				id := "effect:" + effect
				a.node(AnalysisNode{ID: id, Kind: "effect"})
				a.edge(FunctionID(fn), id, "effect")
			}
			a.op(FunctionID(fn)+"/return", "result", "return")
			a.statements(parsed.stmts, "body", 0)
			a.metric.Cyclomatic += a.metric.Decisions
			a.out.Metrics = append(a.out.Metrics, a.metric)
		}
	}
	for _, n := range a.nodes {
		a.out.Nodes = append(a.out.Nodes, n)
	}
	for e := range a.edges {
		a.out.Edges = append(a.out.Edges, e)
	}
	for e := range a.dependences {
		a.out.Dependences = append(a.out.Dependences, e)
	}
	sort.Slice(a.out.Nodes, func(i, j int) bool { return a.out.Nodes[i].ID < a.out.Nodes[j].ID })
	sort.Slice(a.out.Operations, func(i, j int) bool { return a.out.Operations[i].ID < a.out.Operations[j].ID })
	sortEdges(a.out.Edges)
	sortEdges(a.out.Dependences)
	for i := range a.out.Metrics {
		m := &a.out.Metrics[i]
		incoming, outgoing := map[string]bool{}, map[string]bool{}
		for _, e := range a.out.Edges {
			if !callEdge(e.Kind) {
				continue
			}
			if e.From == m.Symbol {
				outgoing[e.To] = true
			}
			if e.To == m.Symbol {
				incoming[e.From] = true
			}
		}
		m.FanIn = len(incoming)
		m.FanOut = len(outgoing)
	}
	sort.Slice(a.out.Metrics, func(i, j int) bool { return a.out.Metrics[i].Symbol < a.out.Metrics[j].Symbol })
	return a.out, nil
}
func sortEdges(edges []DependencyEdge) {
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Kind < b.Kind
	})
}
func callEdge(kind string) bool { return kind == "call" || kind == "callback" || kind == "capture" }
func (a *programAnalyzer) node(n AnalysisNode) {
	if _, exists := a.nodes[n.ID]; !exists {
		a.nodes[n.ID] = n
	}
}
func (a *programAnalyzer) edge(from, to, kind string) { a.edges[DependencyEdge{from, to, kind}] = true }
func (a *programAnalyzer) dep(from, to, kind string) {
	if from != "" && to != "" && from != to {
		a.dependences[DependencyEdge{from, to, kind}] = true
	}
}
func (a *programAnalyzer) op(id, kind, text string) {
	a.out.Operations = append(a.out.Operations, AnalysisNode{ID: id, Kind: kind, Text: text, Module: a.fn.Module, Function: FunctionID(a.fn), File: a.fn.File, BodyPath: strings.TrimPrefix(id, FunctionID(a.fn)+"/"), IsTest: strings.HasPrefix(a.fn.Name, "test_")})
}
func (a *programAnalyzer) typeEdges(from, typ, module string) {
	typ = strings.TrimPrefix(typ, "?")
	if strings.HasPrefix(typ, "Fn[") {
		if fn, err := ParseFunctionType(typ); err == nil {
			for _, p := range fn.Params {
				a.typeEdges(from, p.Type, module)
			}
			a.typeEdges(from, fn.ReturnType, module)
		}
		return
	}
	for _, prefix := range []string{"List", "R", "Transform"} {
		if parts, ok := genericParts(typ, prefix); ok {
			for _, part := range parts {
				a.typeEdges(from, part, module)
			}
			return
		}
	}
	if schema, err := a.rt.recordSchema(typ, module); err == nil {
		id := schema.Module + "." + schema.Type.Name
		a.node(AnalysisNode{ID: id, Kind: "record", Module: schema.Module})
		a.edge(from, id, "type")
		if from == FunctionID(a.fn) {
			a.dep(a.current, id, "type")
		}
	}
}

func (a *programAnalyzer) inferred(e expr) string {
	env := map[string]inferredValue{}
	mutable := map[string]bool{}
	for name, v := range a.env {
		entry := inferredValue{Type: v.typ}
		if strings.HasPrefix(v.typ, "Fn[") {
			entry.Function, _ = ParseFunctionType(v.typ)
		}
		env[name] = entry
		mutable[name] = true
	}
	c := &functionalChecker{rt: a.rt, module: a.fn.Module, env: env, mutable: mutable, effects: a.fn.Effects, returns: a.rt.qualifyType(a.fn.ReturnType, a.fn.Module)}
	if v, err := c.infer(e); err == nil {
		return v.Type
	}
	return "Any"
}
func (a *programAnalyzer) statements(stmts []stmt, path string, depth int) {
	if depth > a.metric.Nesting {
		a.metric.Nesting = depth
	}
	for i, statement := range stmts {
		id := FunctionID(a.fn) + "/" + path + "/s" + strconv.Itoa(i)
		a.current = id
		a.metric.Statements++
		a.op(id, "statement", statementText(statement))
		for _, parent := range a.controls {
			a.dep(id, parent, "control")
		}
		switch n := statement.(type) {
		case *letStmt:
			v := a.expression(n.Value, id+"/value", depth)
			v.defs = []string{id}
			a.env[n.Name] = v
		case *assignStmt:
			v := a.expression(n.Value, id+"/value", depth)
			v.defs = []string{id}
			a.env[n.Name] = v
		case *exprStmt:
			a.expression(n.Value, id+"/value", depth)
		case *returnStmt:
			a.expression(n.Value, id+"/value", depth)
			a.dep(FunctionID(a.fn)+"/return", id, "data")
		case *groupStmt:
			a.statements(n.Body, path+"/s"+strconv.Itoa(i)+"/group", depth+1)
		case *whenStmt:
			a.when(n.Value, n.Cases, id, depth)
		case *forStmt:
			v := a.expression(n.Items, id+"/items", depth)
			base := cloneAnalysisEnv(a.env)
			a.env = cloneAnalysisEnv(base)
			element := "Any"
			if parts, ok := genericParts(v.typ, "List"); ok && len(parts) == 1 {
				element = parts[0]
			}
			bind := id + "/binding"
			a.op(bind, "binding", n.Name)
			a.dep(bind, id, "data")
			a.env[n.Name] = abstractValue{typ: element, defs: []string{bind}}
			old := a.controls
			a.controls = append(append([]string{}, old...), id)
			a.metric.Decisions++
			start := len(a.out.Operations)
			a.statements(n.Body, path+"/s"+strconv.Itoa(i)+"/loop", depth+1)
			for _, op := range a.out.Operations[start:] {
				for name := range a.reads[op.ID] {
					for _, def := range a.env[name].defs {
						a.dep(op.ID, def, "loop_carried")
					}
				}
			}
			branch := a.env
			delete(branch, n.Name)
			a.env = mergeEnv(base, []analysisEnv{branch}, true)
			a.controls = old
		}
	}
}
func mergeEnv(base analysisEnv, branches []analysisEnv, includeBase bool) analysisEnv {
	out := cloneAnalysisEnv(base)
	for name, original := range base {
		values := []abstractValue{}
		if includeBase {
			values = append(values, original)
		}
		for _, branch := range branches {
			if v, ok := branch[name]; ok {
				values = append(values, v)
			} else {
				values = append(values, original)
			}
		}
		merged := original
		merged.defs = nil
		merged.targets = nil
		merged.unknown = false
		for _, v := range values {
			merged.defs = append(merged.defs, v.defs...)
			merged.targets = append(merged.targets, v.targets...)
			merged.unknown = merged.unknown || v.unknown
		}
		merged.defs = uniqueStrings(merged.defs)
		merged.targets = uniqueStrings(merged.targets)
		out[name] = merged
	}
	return out
}
func (a *programAnalyzer) when(subject expr, cases []whenCase, path string, depth int) abstractValue {
	value := a.expression(subject, path+"/subject", depth)
	base := cloneAnalysisEnv(a.env)
	oldCurrent := a.current
	oldControls := a.controls
	branches := []analysisEnv{}
	a.metric.Decisions += max(0, len(cases)-1)
	if !whenCovers(cases) {
		a.metric.Decisions++
	}
	result := abstractValue{typ: "Any"}
	for i, arm := range cases {
		a.env = cloneAnalysisEnv(base)
		id := path + "/arm" + strconv.Itoa(i)
		a.op(id, "branch", patternText(arm))
		a.dep(id, oldCurrent, "control")
		a.dep(oldCurrent, id, "value")
		a.controls = append(append([]string{}, oldControls...), id)
		a.current = id
		a.bindPattern(arm.Pattern, value.typ, id, value.defs)
		if arm.Expr != nil {
			v := a.expression(arm.Expr, id+"/value", depth+1)
			result.typ = v.typ
			result.targets = append(result.targets, v.targets...)
			result.unknown = result.unknown || v.unknown
			result.defs = append(result.defs, v.defs...)
		} else {
			a.statements(arm.Body, strings.TrimPrefix(id, FunctionID(a.fn)+"/"), depth+1)
		}
		branches = append(branches, a.env)
	}
	a.env = mergeEnv(base, branches, true)
	a.controls = oldControls
	a.current = oldCurrent
	result.targets = uniqueStrings(result.targets)
	result.defs = uniqueStrings(result.defs)
	return result
}
func (a *programAnalyzer) bindPattern(p expr, typ, id string, defs []string) {
	switch n := p.(type) {
	case *bindingExpr:
		a.op(id+"/bind/"+n.Name, "binding", n.Name)
		for _, def := range defs {
			a.dep(id+"/bind/"+n.Name, def, "data")
		}
		a.dep(id+"/bind/"+n.Name, id, "control")
		a.env[n.Name] = abstractValue{typ: typ, defs: []string{id + "/bind/" + n.Name}, unknown: strings.HasPrefix(typ, "Fn[")}
	case *callExpr:
		types := map[string]string{}
		name := expressionName(n.Callee)
		if parts, ok := genericParts(typ, "R"); ok && len(parts) == 2 {
			types["value"] = parts[0]
			types["error"] = parts[1]
		}
		if ctor, ok := a.rt.variantConstructor(name, a.fn.Module); ok {
			for _, f := range ctor.Variant.Fields {
				types[f.Name] = a.rt.qualifyType(f.Type, ctor.Schema.Module)
			}
		}
		for _, arg := range n.Args {
			t := types[arg.Name]
			if t == "" {
				t = "Any"
			}
			a.bindPattern(arg.Value, t, id, defs)
		}
	}
}
func (a *programAnalyzer) expression(e expr, path string, depth int) abstractValue {
	if depth > a.metric.Nesting {
		a.metric.Nesting = depth
	}
	result := abstractValue{typ: a.inferred(e)}
	switch n := e.(type) {
	case *identExpr:
		if v, ok := a.env[n.Name]; ok {
			if a.reads[a.current] == nil {
				a.reads[a.current] = map[string]bool{}
			}
			a.reads[a.current][n.Name] = true
			for _, def := range v.defs {
				a.dep(a.current, def, "data")
			}
			return v
		}
		if fn, ok := a.rt.modules[a.fn.Module].funcs[n.Name]; ok {
			result.targets = []string{FunctionID(fn)}
			a.reference(result.targets[0])
			return result
		}
		if _, ok := builtinFunctionType(n.Name); ok {
			result.targets = []string{"builtin:" + n.Name}
			a.node(AnalysisNode{ID: result.targets[0], Kind: "builtin"})
			a.reference(result.targets[0])
			return result
		}
		if _, ok := a.rt.modules[a.fn.Module].consts[n.Name]; ok {
			a.edge(FunctionID(a.fn), a.fn.Module+"."+n.Name, "read")
			a.dep(a.current, a.fn.Module+"."+n.Name, "read")
		}
	case *selectorExpr:
		name := expressionName(n)
		root := strings.Split(name, ".")[0]
		if _, local := a.env[root]; !local {
			if ctor, ok := a.rt.variantConstructor(name, a.fn.Module); ok {
				result.targets = []string{ctor.Schema.Module + "." + ctor.Schema.Type.Name + "." + ctor.Variant.Name}
				a.reference(result.targets[0])
				return result
			}
			if _, ok := builtinFunctionType(name); ok {
				result.targets = []string{"builtin:" + name}
				a.node(AnalysisNode{ID: result.targets[0], Kind: "builtin"})
				a.reference(result.targets[0])
				return result
			}
			if left, ok := n.Left.(*identExpr); ok {
				if mod := a.rt.modules[left.Name]; mod != nil {
					if fn, ok := mod.funcs[n.Name]; ok {
						result.targets = []string{FunctionID(fn)}
						a.reference(result.targets[0])
						return result
					}
					if _, ok := mod.consts[n.Name]; ok {
						a.edge(FunctionID(a.fn), name, "read")
						a.dep(a.current, name, "read")
						return result
					}
				}
			}
		}
		left := a.expression(n.Left, path+"/left", depth)
		if method, ok := a.rt.lookupMethod(left.typ, n.Name); ok {
			result.targets = []string{FunctionID(method)}
			a.reference(result.targets[0])
		} else if strings.HasPrefix(result.typ, "Fn[") {
			result.unknown = true
		}
		a.typeEdges(FunctionID(a.fn), left.typ, a.fn.Module)
	case *partialExpr:
		return a.call(n.Call, path, depth, true)
	case *callExpr:
		return a.call(n, path, depth, false)
	case *binaryExpr:
		a.expression(n.Left, path+"/left", depth)
		a.expression(n.Right, path+"/right", depth)
	case *structExpr:
		a.typeEdges(FunctionID(a.fn), n.TypeName, a.fn.Module)
		for i, f := range n.Fields {
			a.expression(f.Value, fmt.Sprintf("%s/f%d", path, i), depth)
		}
	case *unwrapExpr:
		a.metric.Decisions++
		a.expression(n.Value, path+"/unwrap", depth)
	case *whenExpr:
		return a.when(n.Value, n.Cases, path, depth)
	case *sequenceExpr:
		for i, item := range n.Items {
			result = a.expression(item, fmt.Sprintf("%s/e%d", path, i), depth)
		}
	case *listExpr:
		for i, item := range n.Items {
			a.expression(item, fmt.Sprintf("%s/i%d", path, i), depth)
		}
	case *spawnExpr:
		a.expression(n.Value, path+"/spawn", depth)
	case *awaitExpr:
		a.expression(n.Value, path+"/await", depth)
	case *stringExpr:
		index := 0
		_, _ = interpolateText(n.Value, func(raw string) (string, error) {
			expression, err := interpolationExpr(raw)
			if err == nil {
				a.expression(expression, fmt.Sprintf("%s/interp%d", path, index), depth)
				index++
			}
			return "", nil
		})
	}
	return result
}
func (a *programAnalyzer) call(call *callExpr, path string, depth int, partial bool) abstractValue {
	callee := a.expression(call.Callee, path+"/callee", depth)
	kind := "call"
	if partial {
		kind = "capture"
	}
	for _, target := range callee.targets {
		a.edge(FunctionID(a.fn), target, kind)
		a.dep(a.current, target, kind)
	}
	if len(callee.targets) == 0 || callee.unknown {
		target := "unresolved:" + path
		a.node(AnalysisNode{ID: target, Kind: "unresolved", Function: FunctionID(a.fn), Text: expressionName(call.Callee)})
		a.edge(FunctionID(a.fn), target, "unresolved_call")
		a.dep(a.current, target, "unresolved_call")
		a.metric.Unresolved++
	}
	for i, arg := range call.Args {
		v := a.expression(arg.Value, fmt.Sprintf("%s/arg%d", path, i), depth)
		for _, target := range v.targets {
			a.edge(FunctionID(a.fn), target, "callback")
			a.dep(a.current, target, "callback")
		}
		if v.unknown || (strings.HasPrefix(v.typ, "Transform[") && len(v.targets) == 0) {
			target := fmt.Sprintf("unresolved:%s/arg%d", path, i)
			a.node(AnalysisNode{ID: target, Kind: "unresolved", Function: FunctionID(a.fn), Text: "callback or transform argument " + arg.Name})
			a.edge(FunctionID(a.fn), target, "unresolved_call")
			a.dep(a.current, target, "unresolved_call")
			a.metric.Unresolved++
		}
	}
	for _, typ := range call.TypeArgs {
		a.typeEdges(FunctionID(a.fn), typ, a.fn.Module)
	}
	if partial {
		callee.typ = a.inferred(&partialExpr{Call: call})
		return callee
	}
	typ := a.inferred(call)
	return abstractValue{typ: typ, unknown: strings.HasPrefix(typ, "Fn[")}
}
func statementText(s stmt) string {
	switch n := s.(type) {
	case *letStmt:
		return n.Name + " = " + exprText(n.Value)
	case *assignStmt:
		return n.Name + " = " + exprText(n.Value)
	case *returnStmt:
		return "return " + exprText(n.Value)
	case *exprStmt:
		return exprText(n.Value)
	case *whenStmt:
		return "when (" + exprText(n.Value) + ")"
	case *forStmt:
		return "for " + n.Name + " in " + exprText(n.Items)
	case *groupStmt:
		return "group"
	}
	return ""
}
func patternText(arm whenCase) string {
	if arm.Wildcard {
		return "_"
	}
	return exprText(arm.Pattern)
}
func exprText(e expr) string {
	switch n := e.(type) {
	case *intExpr:
		return strconv.Itoa(n.Value)
	case *stringExpr:
		return strconv.Quote(n.Value)
	case *identExpr:
		return n.Name
	case *bindingExpr:
		return "let " + n.Name
	case *nilExpr:
		return "nil"
	case *defaultExpr:
		return "default"
	case *selectorExpr:
		return exprText(n.Left) + "." + n.Name
	case *callExpr:
		args := []string{}
		for _, arg := range n.Args {
			args = append(args, arg.Name+":"+exprText(arg.Value))
		}
		return exprText(n.Callee) + "(" + strings.Join(args, ",") + ")"
	case *partialExpr:
		return "partial " + exprText(n.Call)
	case *binaryExpr:
		return "(" + exprText(n.Left) + n.Op + exprText(n.Right) + ")"
	case *unwrapExpr:
		return exprText(n.Value) + "?"
	case *whenExpr:
		return "when (" + exprText(n.Value) + ") {...}"
	case *listExpr:
		items := []string{}
		for _, i := range n.Items {
			items = append(items, exprText(i))
		}
		return "[" + strings.Join(items, ",") + "]"
	case *structExpr:
		fields := []string{}
		for _, f := range n.Fields {
			fields = append(fields, f.Name+":"+exprText(f.Value))
		}
		return n.TypeName + "{" + strings.Join(fields, ",") + "}"
	case *spawnExpr:
		return "spawn " + exprText(n.Value)
	case *awaitExpr:
		return "await " + exprText(n.Value)
	case *sequenceExpr:
		items := []string{}
		for _, i := range n.Items {
			items = append(items, exprText(i))
		}
		return strings.Join(items, " ")
	}
	return "..."
}

func (a *programAnalyzer) reference(target string) {
	a.edge(FunctionID(a.fn), target, "reference")
	a.dep(a.current, target, "reference")
}
