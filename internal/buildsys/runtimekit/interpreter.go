package runtimekit

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type runtime struct {
	spec       ProgramSpec
	modules    map[string]*moduleRuntime
	entry      *moduleRuntime
	stdout     io.Writer
	stderr     io.Writer
	args       []string
	callMu     sync.Mutex
	parserMemo map[string]parsedFunction
}

type moduleRuntime struct {
	spec   ModuleSpec
	consts map[string]value
	types  map[string]TypeSpec
	funcs  map[string]FuncSpec
}

type parsedFunction struct {
	stmts []stmt
	expr  expr
}

type value interface{}

type structValue struct {
	Type   string         `json:"type"`
	Module string         `json:"module,omitempty"`
	Fields map[string]any `json:"fields"`
}

type resultValue struct {
	Kind  resultKind
	Value value
	Err   value
}

type functionValue struct {
	Module string
	Name   string
}

type methodValue struct {
	Receiver value
	Method   FuncSpec
}

type futureValue struct {
	ch chan futureResult
}

type futureResult struct {
	val value
	err error
}

type execContext struct {
	rt         *runtime
	module     string
	locals     map[string]value
	receiver   value
	returnType string
	effects    []string
	mutable    map[string]bool
	shadowed   map[string]bool
}

type returnSignal struct {
	value value
}

func (*returnSignal) Error() string { return "function return" }

func newRuntime(spec ProgramSpec, args []string, stdout, stderr io.Writer) (*runtime, error) {
	if err := ValidateDeclarations(spec); err != nil {
		return nil, err
	}
	rt := &runtime{spec: spec, stdout: stdout, stderr: stderr, parserMemo: map[string]parsedFunction{}}
	rt.modules = map[string]*moduleRuntime{}
	for _, mod := range spec.Modules {
		mr := &moduleRuntime{spec: mod, consts: map[string]value{}, types: map[string]TypeSpec{}, funcs: map[string]FuncSpec{}}
		for _, c := range mod.Consts {
			switch c.ValueKind {
			case "STRING":
				v, err := strconv.Unquote("\"" + c.Value + "\"")
				if err != nil {
					return nil, err
				}
				mr.consts[c.Name] = v
			case "INT":
				n, err := strconv.Atoi(c.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid const %s.%s: %w", mod.Name, c.Name, err)
				}
				mr.consts[c.Name] = n
			default:
				mr.consts[c.Name] = c.Value
			}
		}
		for _, typ := range mod.Types {
			mr.types[typ.Name] = typ
		}
		for _, fn := range mod.Funcs {
			mr.funcs[fn.Name] = fn
		}
		rt.modules[mod.Name] = mr
	}
	entry := rt.modules[spec.EntryModule]
	if entry == nil {
		return nil, fmt.Errorf("entry module %q not found", spec.EntryModule)
	}
	rt.entry = entry
	rt.args = append([]string(nil), args...)
	return rt, nil
}

func (rt *runtime) run() (int, error) {
	switch rt.spec.Mode {
	case "test":
		return rt.runTests()
	default:
		fn, ok := rt.entry.funcs["main"]
		if !ok {
			return 1, fmt.Errorf("entry module %q does not define fn main", rt.entry.spec.Name)
		}
		val, err := rt.callUser(fn, rt.entry.spec.Name, nil, nil)
		if err != nil {
			return 1, err
		}
		val, err = terminalValue(val)
		if err != nil {
			return 1, err
		}
		if code, ok := asInt(val); ok {
			return code, nil
		}
		return 0, nil
	}
}

func (rt *runtime) runTests() (int, error) {
	total := 0
	for _, mod := range rt.spec.Modules {
		for _, fn := range mod.Funcs {
			if !strings.HasPrefix(fn.Name, "test_") {
				continue
			}
			total++
			fmt.Fprintf(rt.stdout, "=== RUN   %s.%s\n", mod.Name, fn.Name)
			testValue, err := rt.callUser(fn, mod.Name, nil, nil)
			if err == nil {
				_, err = terminalValue(testValue)
			}
			if err != nil {
				fmt.Fprintf(rt.stdout, "--- FAIL: %s.%s (%v)\n", mod.Name, fn.Name, err)
				return 1, nil
			}
			fmt.Fprintf(rt.stdout, "--- PASS: %s.%s\n", mod.Name, fn.Name)
		}
	}
	if total == 0 {
		fmt.Fprintln(rt.stdout, "ok\t(no tests found)")
		return 0, nil
	}
	fmt.Fprintf(rt.stdout, "ok\t%d test(s)\n", total)
	return 0, nil
}

func (rt *runtime) callUser(fn FuncSpec, module string, receiver value, args []value) (value, error) {
	if receiver != nil && fn.ReceiverType != "" {
		args = append([]value{receiver}, args...)
	}
	if len(args) != len(fn.Params) {
		return nil, fmt.Errorf("call %s.%s requires %d arguments, got %d", module, fn.Name, len(fn.Params), len(args))
	}
	for i, param := range fn.Params {
		if err := rt.validateValueIn(args[i], param.Type, module); err != nil {
			return nil, fmt.Errorf("argument %s: %w", param.Name, err)
		}
	}
	parsed, err := rt.parseFunc(fn)
	if err != nil {
		return nil, err
	}
	ctx := &execContext{rt: rt, module: module, locals: map[string]value{}, receiver: receiver, effects: fn.Effects, returnType: fn.ReturnType, mutable: map[string]bool{}}
	for i, param := range fn.Params {
		ctx.locals[param.Name] = ascribeValue(args[i], rt.qualifyType(param.Type, module), module)
	}
	if receiver != nil && fn.ReceiverType != "" && len(fn.Params) > 0 {
		ctx.locals[fn.Params[0].Name] = receiver
	}
	if fn.BlockBody {
		val, returned, err := ctx.execStatements(parsed.stmts)
		if err != nil {
			var propagation *propagationSignal
			var explicit *returnSignal
			switch {
			case errors.As(err, &explicit):
				val = explicit.value
			case errors.As(err, &propagation):
				if parts, ok := genericParts(fn.ReturnType, "R"); !ok || len(parts) != 2 {
					return nil, fmt.Errorf("? requires an R[T,E] return type in %s.%s", module, fn.Name)
				}
				val = propagation.Result
			default:
				return nil, err
			}
			returned = true
		}
		if returned {
			if err := rt.validateValueIn(val, fn.ReturnType, module); err != nil {
				return nil, fmt.Errorf("%s.%s return: %w", module, fn.Name, err)
			}
			return ascribeValue(val, rt.qualifyType(fn.ReturnType, module), module), nil
		}
		if err := rt.validateValue(nil, fn.ReturnType); err != nil {
			return nil, fmt.Errorf("%s.%s falls through without a valid return: %w", module, fn.Name, err)
		}
		return nil, nil
	}
	if parsed.expr != nil {
		return ctx.eval(parsed.expr)
	}
	val, _, err := ctx.execStatements(parsed.stmts)
	return val, err
}

func (rt *runtime) parseFunc(fn FuncSpec) (parsedFunction, error) {
	rt.callMu.Lock()
	defer rt.callMu.Unlock()
	key := fn.Module + "." + fn.Name + ":" + fn.Body
	if cached, ok := rt.parserMemo[key]; ok {
		return cached, nil
	}
	p, err := newBodyParser(fn.Body)
	if err != nil {
		return parsedFunction{}, err
	}
	var parsed parsedFunction
	if fn.BlockBody {
		parsed.stmts, err = p.parseStatementsUntil("")
	} else {
		parsed.expr, err = p.parseSequenceExprUntil(TokenEOF)
	}
	if err != nil {
		return parsedFunction{}, err
	}
	rt.parserMemo[key] = parsed
	return parsed, nil
}

func (ctx *execContext) execStatements(stmts []stmt) (value, bool, error) {
	var last value
	for _, st := range stmts {
		val, returned, err := ctx.execStatement(st)
		if err != nil {
			return nil, false, err
		}
		if returned {
			return val, true, nil
		}
		last = val
	}
	return last, false, nil
}

func (ctx *execContext) execStatement(st stmt) (value, bool, error) {
	switch node := st.(type) {
	case *letStmt:
		val, err := ctx.eval(node.Value)
		if err != nil {
			return nil, false, err
		}
		ctx.locals[node.Name] = val
		if ctx.shadowed != nil {
			ctx.shadowed[node.Name] = true
		}
		if ctx.mutable == nil {
			ctx.mutable = map[string]bool{}
		}
		ctx.mutable[node.Name] = node.Mutable
		return val, false, nil
	case *assignStmt:
		if _, ok := ctx.locals[node.Name]; !ok {
			return nil, false, fmt.Errorf("unknown variable %q", node.Name)
		}
		if !ctx.mutable[node.Name] {
			return nil, false, fmt.Errorf("cannot assign to immutable binding %s", node.Name)
		}
		val, err := ctx.eval(node.Value)
		if err != nil {
			return nil, false, err
		}
		ctx.locals[node.Name] = val
		return val, false, nil

	case *returnStmt:
		val, err := ctx.eval(node.Value)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil
	case *exprStmt:
		val, err := ctx.eval(node.Value)
		return val, false, err
	case *whenStmt:
		val, err := ctx.evalWhen(node.Value, node.Cases)
		if err != nil {
			return nil, false, err
		}
		return val, false, nil
	case *forStmt:
		return ctx.execFor(node)
	case *groupStmt:
		val, returned, err := ctx.execStatements(node.Body)
		return val, returned, err
	default:
		return nil, false, fmt.Errorf("unsupported statement %T", st)
	}
}

func (ctx *execContext) evalWhen(target expr, cases []whenCase) (value, error) {
	matchValue, err := ctx.eval(target)
	if err != nil {
		return nil, err
	}
	for _, c := range cases {
		bindings := map[string]value{}
		matched := c.Wildcard
		if !matched {
			matched, err = ctx.matchPattern(c.Pattern, matchValue, bindings)
			if err != nil {
				return nil, err
			}
		}
		if !matched {
			continue
		}
		child := *ctx
		child.locals = copyLocals(ctx.locals)
		child.mutable = copyMutable(ctx.mutable)
		child.shadowed = map[string]bool{}
		for k, v := range bindings {
			child.locals[k] = v
			child.mutable[k] = false
			child.shadowed[k] = true
		}
		var val value
		returned := false
		if c.Expr != nil {
			val, err = child.eval(c.Expr)
		} else {
			val, returned, err = child.execStatements(c.Body)
		}
		for k := range ctx.locals {
			if ctx.mutable[k] {
				if !child.shadowed[k] {
					ctx.locals[k] = child.locals[k]
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if returned {
			if ctx.returnType == "" {
				return nil, fmt.Errorf("return requires an enclosing function")
			}
			return nil, &returnSignal{value: val}
		}
		return val, nil
	}
	return nil, nil
}

func (ctx *execContext) eval(e expr) (value, error) {
	switch node := e.(type) {
	case *partialExpr:
		return ctx.evaluateCall(node.Call, true)
	case *defaultExpr:
		return nil, fmt.Errorf("default is only valid as an explicit call argument")
	case *intExpr:
		return node.Value, nil
	case *stringExpr:
		return ctx.interpolate(node.Value)
	case *nilExpr:
		return nil, nil
	case *identExpr:
		return ctx.resolveIdent(node.Name)
	case *selectorExpr:
		return ctx.resolveSelector(node)
	case *callExpr:
		return ctx.evalCall(node)
	case *binaryExpr:
		return ctx.evalBinary(node)
	case *structExpr:
		return ctx.evalStruct(node)
	case *unwrapExpr:
		return ctx.evalUnwrap(node)
	case *whenExpr:
		return ctx.evalWhen(node.Value, node.Cases)
	case *sequenceExpr:
		var last value
		for _, item := range node.Items {
			v, err := ctx.eval(item)
			if err != nil {
				return nil, err
			}
			last = v
		}
		return last, nil
	case *spawnExpr:
		return ctx.evalSpawn(node)
	case *awaitExpr:
		return ctx.evalAwait(node)
	case *listExpr:
		out := make([]value, 0, len(node.Items))
		for _, item := range node.Items {
			v, err := ctx.eval(item)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported expression %T", e)
	}
}

func interpolateText(src string, expand func(string) (string, error)) (string, error) {
	var out strings.Builder
	for {
		start := strings.Index(src, "${")
		if start < 0 {
			out.WriteString(src)
			return out.String(), nil
		}
		out.WriteString(src[:start])
		src = src[start+2:]
		end := strings.Index(src, "}")
		if end < 0 {
			return "", fmt.Errorf("unterminated interpolation")
		}
		expanded, err := expand(src[:end])
		if err != nil {
			return "", err
		}
		out.WriteString(expanded)
		src = src[end+1:]
	}
}
func interpolationExpr(raw string) (expr, error) {
	parser, err := newBodyParser(raw)
	if err != nil {
		return nil, err
	}
	expression, err := parser.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if parser.peek().Kind != TokenEOF {
		return nil, fmt.Errorf("unexpected trailing interpolation tokens")
	}
	return expression, nil
}
func (ctx *execContext) interpolate(src string) (string, error) {
	return interpolateText(src, func(raw string) (string, error) {
		expression, err := interpolationExpr(raw)
		if err != nil {
			return "", err
		}
		v, err := ctx.eval(expression)
		if err != nil {
			return "", err
		}
		return fmt.Sprint(toPlain(v)), nil
	})
}

func (ctx *execContext) resolveIdent(name string) (value, error) {
	if name == "true" {
		return true, nil
	}
	if name == "false" {
		return false, nil
	}
	if name == "nil" {
		return nil, nil
	}
	if v, ok := ctx.locals[name]; ok {
		return v, nil
	}
	if mod := ctx.rt.modules[ctx.module]; mod != nil {
		if v, ok := mod.consts[name]; ok {
			return v, nil
		}
		if fn, ok := mod.funcs[name]; ok {
			return functionValue{Module: ctx.module, Name: fn.Name}, nil
		}
		if _, ok := mod.types[name]; ok {
			return typeSentinel(ctx.module + "." + name), nil
		}
	}
	if builtinGlobal(name) != nil {
		return builtinValue{Name: name, Invoke: builtinGlobal(name)}, nil
	}
	if ctx.rt.modules[name] != nil || builtinModuleKnown(name) {
		return moduleSentinel(name), nil
	}
	return nil, fmt.Errorf("unknown identifier %q", name)
}

func (ctx *execContext) resolveSelector(sel *selectorExpr) (value, error) {
	left, err := ctx.eval(sel.Left)
	if err != nil {
		return nil, err
	}
	switch v := left.(type) {
	case moduleSentinel:
		if string(v) == "list" && sel.Name == "fold" {
			return nil, fmt.Errorf("list.fold was renamed to list.reduce; keep items, initial and reduce keys")
		}
		if builtin := builtinModuleFunc(string(v), sel.Name); builtin != nil {
			return builtinValue{Name: string(v) + "." + sel.Name, Invoke: builtin}, nil
		}
		if mod := ctx.rt.modules[string(v)]; mod != nil {
			if c, ok := mod.consts[sel.Name]; ok {
				return c, nil
			}
			if fn, ok := mod.funcs[sel.Name]; ok {
				return functionValue{Module: string(v), Name: fn.Name}, nil
			}
			if _, ok := mod.types[sel.Name]; ok {
				return typeSentinel(string(v) + "." + sel.Name), nil
			}
		}
	case typeSentinel:
		if constructor, ok := ctx.rt.variantConstructor(string(v)+"."+sel.Name, ctx.module); ok {
			return constructor, nil
		}
	case *structValue:
		if method, ok := ctx.rt.lookupMethod(ctx.rt.qualifyType(v.Type, v.Module), sel.Name); ok {
			return methodValue{Receiver: v, Method: method}, nil
		}
		if field, ok := v.Fields[sel.Name]; ok {
			return field, nil
		}
	case map[string]any:
		if field, ok := v[sel.Name]; ok {
			return field, nil
		}
	}
	return nil, fmt.Errorf("selector %q not available", sel.Name)
}

func (ctx *execContext) evalCall(call *callExpr) (value, error) {
	return ctx.evaluateCall(call, false)
}

func (ctx *execContext) evalBinary(node *binaryExpr) (value, error) {
	left, err := ctx.eval(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := ctx.eval(node.Right)
	if err != nil {
		return nil, err
	}
	if node.Op == "==" || node.Op == "!=" {
		equal := false
		switch l := left.(type) {
		case int, float64:
			if ctx.rt.validateValue(left, "i32") == nil && ctx.rt.validateValue(right, "i32") == nil {
				a, _ := asInt(left)
				b, _ := asInt(right)
				equal = a == b
			}
		case string:
			r, ok := right.(string)
			equal = ok && l == r
		case bool:
			r, ok := right.(bool)
			equal = ok && l == r
		case nil:
			equal = right == nil
		default:
			equal = valuesEqual(left, right)
		}
		if node.Op == "!=" {
			equal = !equal
		}
		return equal, nil
	}
	if l, ok := left.(string); ok && node.Op == "+" {
		if r, ok := right.(string); ok {
			return l + r, nil
		}
	}
	// JSON numeric fields use float64; accept only integral in-range i32 values.
	if err := ctx.rt.validateValue(left, "i32"); err != nil {
		return nil, fmt.Errorf("operator %s: %w", node.Op, err)
	}
	if err := ctx.rt.validateValue(right, "i32"); err != nil {
		return nil, fmt.Errorf("operator %s: %w", node.Op, err)
	}
	l, lok := asInt(left)
	r, rok := asInt(right)
	if !lok || !rok {
		return nil, fmt.Errorf("operator %s requires matching numeric operands", node.Op)
	}
	switch node.Op {
	case "<":
		return l < r, nil
	case ">":
		return l > r, nil
	case "<=":
		return l <= r, nil
	case ">=":
		return l >= r, nil
	}
	var result int64
	switch node.Op {
	case "+":
		result = int64(l) + int64(r)
	case "-":
		result = int64(l) - int64(r)
	case "*":
		result = int64(l) * int64(r)
	case "/":
		if r == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = int64(l) / int64(r)
	default:
		return nil, fmt.Errorf("unsupported operator %s", node.Op)
	}
	if result < -2147483648 || result > 2147483647 {
		return nil, fmt.Errorf("i32 arithmetic overflow")
	}
	return int(result), nil
}

func (ctx *execContext) evalStruct(node *structExpr) (value, error) {
	schema, err := ctx.rt.recordSchema(node.TypeName, ctx.module)
	if err != nil {
		return nil, err
	}
	if len(schema.Type.Variants) > 0 {
		return nil, fmt.Errorf("variant type %s requires a named constructor", node.TypeName)
	}
	keys := []string{}
	for _, field := range node.Fields {
		keys = append(keys, field.Name)
	}
	if err := checkRecordKeys(schema.Type, keys, true); err != nil {
		return nil, err
	}
	fields := map[string]any{}
	for _, field := range node.Fields {
		v, err := ctx.eval(field.Value)
		if err != nil {
			return nil, err
		}
		typ := schema.fieldType(field.Name)
		if err := ctx.rt.validateValueIn(v, typ, schema.Module); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", node.TypeName, field.Name, err)
		}
		fields[field.Name] = ascribeValue(v, ctx.rt.qualifyType(typ, schema.Module), schema.Module)
	}
	return &structValue{Type: schema.Type.Name, Module: schema.Module, Fields: fields}, nil
}

func (ctx *execContext) evalUnwrap(node *unwrapExpr) (value, error) {
	if parts, ok := genericParts(ctx.returnType, "R"); !ok || len(parts) != 2 {
		return nil, fmt.Errorf("? requires an R[T,E] return type in the enclosing function")
	}
	v, err := ctx.eval(node.Value)
	if err != nil {
		return nil, err
	}
	if result, ok := v.(resultValue); ok {
		switch result.Kind {
		case resultErr:
			return nil, &propagationSignal{Result: result}
		case resultOK:
			return result.Value, nil
		default:
			return nil, fmt.Errorf("invalid result tag")
		}
	}
	return nil, fmt.Errorf("invalid ? on non-result value\nhint: use ? only with R[T,E] values")
}

func (ctx *execContext) evalSpawn(node *spawnExpr) (value, error) {
	future := &futureValue{ch: make(chan futureResult, 1)}
	child := *ctx
	child.returnType = "" // Propagation belongs in the spawned function, then explicitly after await.
	child.locals = map[string]value{}
	for name, v := range ctx.locals {
		child.locals[name] = v
	}
	child.mutable = copyMutable(ctx.mutable)
	go func() {
		v, err := child.eval(node.Value)
		future.ch <- futureResult{val: v, err: err}
	}()
	return future, nil
}

func (ctx *execContext) evalAwait(node *awaitExpr) (value, error) {
	v, err := ctx.eval(node.Value)
	if err != nil {
		return nil, err
	}
	future, ok := v.(*futureValue)
	if !ok {
		return nil, fmt.Errorf("await requires a spawned task\nhint: call spawn first and await the returned task")
	}
	result := <-future.ch
	return result.val, result.err
}

func (rt *runtime) lookupMethod(receiverType, name string) (FuncSpec, bool) {
	for _, mod := range rt.modules {
		for _, fn := range mod.funcs {
			if fn.Name == name && fn.ReceiverType != "" && rt.qualifyType(fn.ReceiverType, fn.Module) == rt.qualifyType(receiverType, "") {
				return fn, true
			}
		}
	}
	return FuncSpec{}, false
}

type moduleSentinel string

type typeSentinel string

type builtinFn func(ctx *execContext, typeArgs []string, args []value) (value, error)

type builtinValue struct {
	Name   string
	Invoke builtinFn
}

func builtinGlobal(name string) builtinFn {
	switch name {
	case "print":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			for _, arg := range args {
				fmt.Fprint(ctx.rt.stdout, fmt.Sprint(toPlain(arg)))
			}
			return nil, nil
		}
	case "println":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			parts := make([]any, 0, len(args))
			for _, arg := range args {
				parts = append(parts, toPlain(arg))
			}
			fmt.Fprintln(ctx.rt.stdout, parts...)
			return nil, nil
		}
	case "Ok":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			var v value
			if len(args) > 0 {
				v = args[0]
			}
			return okResult(v), nil
		}
	case "Err":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			var v value
			if len(args) > 0 {
				v = args[0]
			}
			return errResult(v), nil
		}
	}
	return nil
}

func builtinModuleKnown(name string) bool {
	switch name {
	case "list", "env", "log", "test", "assert", "parse", "cli", "fs", "http", "json", "ctx":
		return true
	default:
		return false
	}
}
func builtinModuleFunc(module, name string) builtinFn {
	switch module + "." + name {
	case "list.mapping", "list.filtering", "list.compose", "list.transduce", "list.collect":
		return transformOperation(name)
	case "list.map", "list.filter", "list.reduce":
		return listOperation(name)
	case "env.get_or":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			key := fmt.Sprint(toPlain(args[0]))
			fallback := fmt.Sprint(toPlain(args[1]))
			if got, ok := os.LookupEnv(key); ok {
				return got, nil
			}
			return fallback, nil
		}
	case "log.info":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			fmt.Fprintln(ctx.rt.stderr, fmt.Sprint(toPlain(args[0])))
			return nil, nil
		}
	case "test.eq":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			if !valuesEqual(args[0], args[1]) {
				return nil, fmt.Errorf("assertion failed: %v != %v", toPlain(args[0]), toPlain(args[1]))
			}
			return nil, nil
		}
	case "assert.equal_i32":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			if !valuesEqual(args[0], args[1]) {
				return nil, fmt.Errorf("assertion failed: %v != %v", toPlain(args[0]), toPlain(args[1]))
			}
			return nil, nil
		}
	case "parse.i32":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			n, err := strconv.ParseInt(fmt.Sprint(toPlain(args[0])), 10, 32)
			if err != nil {
				return errResult(err.Error()), nil
			}
			return okResult(int(n)), nil
		}
	case "cli.args":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			out := make([]value, 0, len(ctx.rt.args))
			for _, arg := range ctx.rt.args {
				out = append(out, arg)
			}
			return out, nil
		}
	case "fs.read_file":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			body, err := os.ReadFile(fmt.Sprint(toPlain(args[0])))
			if err != nil {
				return errResult(err.Error()), nil
			}
			return okResult(string(body)), nil
		}
	case "fs.write_file":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			err := os.WriteFile(fmt.Sprint(toPlain(args[0])), []byte(fmt.Sprint(toPlain(args[1]))), 0o644)
			if err != nil {
				return errResult(err.Error()), nil
			}
			return okResult(nil), nil
		}
	case "http.text":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			code, _ := asInt(args[0])
			return &structValue{Type: "Resp", Fields: map[string]any{"code": code, "body": fmt.Sprint(toPlain(args[1])), "headers": map[string]any{"Content-Type": "text/plain"}}}, nil
		}
	case "http.json":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			code, _ := asInt(args[0])
			return &structValue{Type: "Resp", Fields: map[string]any{"code": code, "body": fmt.Sprint(toPlain(args[1])), "headers": map[string]any{"Content-Type": "application/json"}}}, nil
		}
	case "http.not_found":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			return &structValue{Type: "Resp", Fields: map[string]any{"code": 404, "body": "not found", "headers": map[string]any{"Content-Type": "text/plain"}}}, nil
		}
	case "http.with_header":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			resp, ok := args[0].(*structValue)
			if !ok {
				return nil, errors.New("http.with_header expects a response")
			}
			headers, _ := resp.Fields["headers"].(map[string]any)
			if headers == nil {
				headers = map[string]any{}
			}
			headers[fmt.Sprint(toPlain(args[1]))] = fmt.Sprint(toPlain(args[2]))
			resp.Fields["headers"] = headers
			return resp, nil
		}
	case "http.test_req":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			return &structValue{Type: "Req", Fields: map[string]any{"method": fmt.Sprint(toPlain(args[0])), "path": fmt.Sprint(toPlain(args[1])), "body": fmt.Sprint(toPlain(args[2]))}}, nil
		}
	case "http.query":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			if req, ok := args[0].(*structValue); ok {
				path := fmt.Sprint(req.Fields["path"])
				parts := strings.SplitN(path, "?", 2)
				if len(parts) < 2 {
					return nil, nil
				}
				for _, pair := range strings.Split(parts[1], "&") {
					kv := strings.SplitN(pair, "=", 2)
					if len(kv) == 2 && kv[0] == fmt.Sprint(toPlain(args[1])) {
						return kv[1], nil
					}
				}
			}
			return nil, nil
		}
	case "http.serve":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			addr := fmt.Sprint(toPlain(args[0]))
			handler := args[1]
			server := http.Server{Addr: addr}
			server.Handler = ctx.rt.httpHandler(handler)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return errResult(err.Error()), nil
			}
			return okResult(nil), nil
		}
	case "json.encode":
		return encodeJSON
	case "json.decode":
		return decodeJSON
	case "ctx.background":
		return func(ctx *execContext, _ []string, args []value) (value, error) { return "background", nil }
	case "ctx.with_timeout_ms":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			return fmt.Sprintf("timeout(%v,%v)", toPlain(args[0]), toPlain(args[1])), nil
		}
	}
	return nil
}

func readBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return string(body)
}

func valuesEqual(a, b value) bool {
	return reflect.DeepEqual(toPlain(a), toPlain(b))
}

func toPlain(v value) any {
	switch x := v.(type) {
	case *variantValue:
		return map[string]any{"_type": x.Module + "." + x.Type, "_variant": x.Tag, "fields": toPlain(x.Fields)}
	case *structValue:
		out := map[string]any{"_type": x.Type}
		for k, v := range x.Fields {
			out[k] = toPlain(v)
		}
		return out
	case []value:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, toPlain(item))
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for key, item := range x {
			out[key] = toPlain(item)
		}
		return out
	case resultValue:
		if x.Kind == resultErr {
			return map[string]any{"Err": toPlain(x.Err)}
		}
		if x.Kind != resultOK {
			return map[string]any{"InvalidResult": true}
		}
		return map[string]any{"Ok": toPlain(x.Value)}
	default:
		return x
	}
}

func asInt(v value) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case string:
		parsed, err := strconv.Atoi(n)
		return parsed, err == nil
	case map[string]any:
		if raw, ok := n["code"]; ok {
			return asInt(raw)
		}
	case *structValue:
		if raw, ok := n.Fields["code"]; ok {
			return asInt(raw)
		}
	}
	return 0, false
}
