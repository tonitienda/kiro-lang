package runtimekit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Fields map[string]any `json:"fields"`
}

type resultValue struct {
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
	rt       *runtime
	module   string
	locals   map[string]value
	receiver value
}

type returnSignal struct {
	value value
}

func newRuntime(spec ProgramSpec, args []string, stdout, stderr io.Writer) (*runtime, error) {
	rt := &runtime{spec: spec, stdout: stdout, stderr: stderr, parserMemo: map[string]parsedFunction{}}
	rt.modules = map[string]*moduleRuntime{}
	for _, mod := range spec.Modules {
		mr := &moduleRuntime{spec: mod, consts: map[string]value{}, types: map[string]TypeSpec{}, funcs: map[string]FuncSpec{}}
		for _, c := range mod.Consts {
			switch c.ValueKind {
			case "STRING":
				mr.consts[c.Name] = c.Value
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
			if _, err := rt.callUser(fn, mod.Name, nil, nil); err != nil {
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
	parsed, err := rt.parseFunc(fn)
	if err != nil {
		return nil, err
	}
	ctx := &execContext{rt: rt, module: module, locals: map[string]value{}, receiver: receiver}
	for i, param := range fn.Params {
		if i < len(args) {
			ctx.locals[param.Name] = args[i]
		} else {
			ctx.locals[param.Name] = nil
		}
	}
	if receiver != nil && fn.ReceiverType != "" && len(fn.Params) > 0 {
		ctx.locals[fn.Params[0].Name] = receiver
	}
	if fn.BlockBody {
		val, returned, err := ctx.execStatements(parsed.stmts)
		if err != nil {
			return nil, err
		}
		if returned {
			return val, nil
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
		return val, false, nil
	case *assignStmt:
		val, err := ctx.eval(node.Value)
		if err != nil {
			return nil, false, err
		}
		if _, ok := ctx.locals[node.Name]; ok {
			ctx.locals[node.Name] = val
			return val, false, nil
		}
		return nil, false, fmt.Errorf("unknown variable %q", node.Name)
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
		if sig, ok := val.(returnSignal); ok {
			return sig.value, true, nil
		}
		return val, false, nil
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
		matched := c.Wildcard
		if !matched {
			pat, err := ctx.eval(c.Pattern)
			if err != nil {
				return nil, err
			}
			matched = valuesEqual(matchValue, pat)
		}
		if !matched {
			continue
		}
		if len(c.Body) > 0 {
			val, returned, err := ctx.execStatements(c.Body)
			if err != nil {
				return nil, err
			}
			if returned {
				return returnSignal{value: val}, nil
			}
			return val, nil
		}
		return ctx.eval(c.Expr)
	}
	return nil, nil
}

func (ctx *execContext) eval(e expr) (value, error) {
	switch node := e.(type) {
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

func (ctx *execContext) interpolate(src string) (string, error) {
	out := src
	for {
		start := strings.Index(out, "${")
		if start < 0 {
			return out, nil
		}
		end := strings.Index(out[start+2:], "}")
		if end < 0 {
			return "", fmt.Errorf("unterminated interpolation in %q", src)
		}
		raw := out[start+2 : start+2+end]
		p, err := newBodyParser(raw)
		if err != nil {
			return "", err
		}
		ex, err := p.parseExpr(0)
		if err != nil {
			return "", err
		}
		val, err := ctx.eval(ex)
		if err != nil {
			return "", err
		}
		out = out[:start] + fmt.Sprint(toPlain(val)) + out[start+2+end+1:]
	}
}

func (ctx *execContext) resolveIdent(name string) (value, error) {
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
			return typeSentinel(name), nil
		}
	}
	if builtinGlobal(name) != nil {
		return builtinGlobal(name), nil
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
		if builtin := builtinModuleFunc(string(v), sel.Name); builtin != nil {
			return builtin, nil
		}
		if mod := ctx.rt.modules[string(v)]; mod != nil {
			if c, ok := mod.consts[sel.Name]; ok {
				return c, nil
			}
			if fn, ok := mod.funcs[sel.Name]; ok {
				return functionValue{Module: string(v), Name: fn.Name}, nil
			}
			if _, ok := mod.types[sel.Name]; ok {
				return typeSentinel(sel.Name), nil
			}
		}
	case *structValue:
		if method, ok := ctx.rt.lookupMethod(v.Type, sel.Name); ok {
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
	callee, err := ctx.eval(call.Callee)
	if err != nil {
		return nil, err
	}
	args := make([]value, 0, len(call.Args))
	for _, arg := range call.Args {
		v, err := ctx.eval(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	switch fn := callee.(type) {
	case builtinFn:
		return fn(ctx, call.TypeArgs, args)
	case functionValue:
		mod := ctx.rt.modules[fn.Module]
		if mod == nil {
			return nil, fmt.Errorf("unknown module %q", fn.Module)
		}
		spec, ok := mod.funcs[fn.Name]
		if !ok {
			return nil, fmt.Errorf("unknown function %s.%s", fn.Module, fn.Name)
		}
		return ctx.rt.callUser(spec, fn.Module, nil, args)
	case methodValue:
		return ctx.rt.callUser(fn.Method, fn.Method.Module, fn.Receiver, args)
	case typeSentinel:
		return ctx.constructStruct(string(fn), args)
	default:
		return nil, fmt.Errorf("expression is not callable: %T", callee)
	}
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
	switch node.Op {
	case "+":
		if li, lok := asInt(left); lok {
			if ri, rok := asInt(right); rok {
				return li + ri, nil
			}
		}
		return fmt.Sprint(toPlain(left)) + fmt.Sprint(toPlain(right)), nil
	default:
		return nil, fmt.Errorf("unsupported operator %q", node.Op)
	}
}

func (ctx *execContext) evalStruct(node *structExpr) (value, error) {
	fields := map[string]any{}
	for _, field := range node.Fields {
		v, err := ctx.eval(field.Value)
		if err != nil {
			return nil, err
		}
		fields[field.Name] = toPlain(v)
	}
	return &structValue{Type: node.TypeName, Fields: fields}, nil
}

func (ctx *execContext) evalUnwrap(node *unwrapExpr) (value, error) {
	v, err := ctx.eval(node.Value)
	if err != nil {
		return nil, err
	}
	if res, ok := v.(resultValue); ok {
		if res.Err != nil {
			return nil, fmt.Errorf("%v", res.Err)
		}
		return res.Value, nil
	}
	return v, nil
}

func (ctx *execContext) evalSpawn(node *spawnExpr) (value, error) {
	future := &futureValue{ch: make(chan futureResult, 1)}
	go func() {
		v, err := ctx.eval(node.Value)
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
		return nil, fmt.Errorf("await requires a spawned task")
	}
	result := <-future.ch
	return result.val, result.err
}

func (ctx *execContext) constructStruct(typeName string, args []value) (value, error) {
	return &structValue{Type: typeName, Fields: map[string]any{}}, nil
}

func (rt *runtime) lookupMethod(receiverType, name string) (FuncSpec, bool) {
	for _, mod := range rt.modules {
		for _, fn := range mod.funcs {
			if fn.Name == name && fn.ReceiverType == receiverType {
				return fn, true
			}
		}
	}
	return FuncSpec{}, false
}

type moduleSentinel string

type typeSentinel string

type builtinFn func(ctx *execContext, typeArgs []string, args []value) (value, error)

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
			return resultValue{Value: v}, nil
		}
	case "Err":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			var v value
			if len(args) > 0 {
				v = args[0]
			}
			return resultValue{Err: v}, nil
		}
	}
	return nil
}

func builtinModuleKnown(name string) bool {
	switch name {
	case "env", "log", "test", "assert", "parse", "cli", "fs", "http", "json", "ctx":
		return true
	default:
		return false
	}
}
func builtinModuleFunc(module, name string) builtinFn {
	switch module + "." + name {
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
			n, err := strconv.Atoi(fmt.Sprint(toPlain(args[0])))
			if err != nil {
				return resultValue{Err: err.Error()}, nil
			}
			return resultValue{Value: n}, nil
		}
	case "cli.args":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			out := make([]value, 0, len(ctx.rt.args))
			for _, arg := range ctx.rt.args {
				out = append(out, arg)
			}
			return out, nil
		}
	case "fs.read", "fs.read_file":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			body, err := os.ReadFile(fmt.Sprint(toPlain(args[0])))
			if err != nil {
				return resultValue{Err: err.Error()}, nil
			}
			return resultValue{Value: string(body)}, nil
		}
	case "fs.write_file":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			err := os.WriteFile(fmt.Sprint(toPlain(args[0])), []byte(fmt.Sprint(toPlain(args[1]))), 0o644)
			if err != nil {
				return resultValue{Err: err.Error()}, nil
			}
			return resultValue{Value: nil}, nil
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
			server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				req := &structValue{Type: "Req", Fields: map[string]any{"method": r.Method, "path": r.URL.Path, "body": readBody(r), "ctx": "request"}}
				var resp value
				var err error
				switch hv := handler.(type) {
				case functionValue:
					fn := ctx.rt.modules[hv.Module].funcs[hv.Name]
					resp, err = ctx.rt.callUser(fn, hv.Module, nil, []value{req})
				case methodValue:
					resp, err = ctx.rt.callUser(hv.Method, hv.Method.Module, hv.Receiver, []value{req})
				default:
					err = fmt.Errorf("http handler is not callable: %T", handler)
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				resp = unwrapResult(resp)
				sv, ok := resp.(*structValue)
				if !ok {
					http.Error(w, "handler did not return Resp", http.StatusInternalServerError)
					return
				}
				if headers, ok := sv.Fields["headers"].(map[string]any); ok {
					for k, v := range headers {
						w.Header().Set(k, fmt.Sprint(v))
					}
				}
				code, _ := asInt(sv.Fields["code"])
				if code == 0 {
					code = 200
				}
				w.WriteHeader(code)
				_, _ = io.WriteString(w, fmt.Sprint(sv.Fields["body"]))
			})
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return resultValue{Err: err.Error()}, nil
			}
			return resultValue{Value: nil}, nil
		}
	case "json.encode":
		return func(ctx *execContext, _ []string, args []value) (value, error) {
			buf := &bytes.Buffer{}
			if err := json.NewEncoder(buf).Encode(toPlain(args[0])); err != nil {
				return resultValue{Err: err.Error()}, nil
			}
			return resultValue{Value: strings.TrimSpace(buf.String())}, nil
		}
	case "json.decode":
		return func(ctx *execContext, typeArgs []string, args []value) (value, error) {
			var raw any
			if err := json.Unmarshal([]byte(fmt.Sprint(toPlain(args[0]))), &raw); err != nil {
				return resultValue{Err: err.Error()}, nil
			}
			if len(typeArgs) > 0 {
				if m, ok := raw.(map[string]any); ok {
					return resultValue{Value: &structValue{Type: typeArgs[0], Fields: m}}, nil
				}
			}
			return resultValue{Value: raw}, nil
		}
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

func unwrapResult(v value) value {
	if res, ok := v.(resultValue); ok {
		return res.Value
	}
	return v
}

func valuesEqual(a, b value) bool {
	return fmt.Sprint(toPlain(a)) == fmt.Sprint(toPlain(b))
}

func toPlain(v value) any {
	switch x := v.(type) {
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
	case resultValue:
		if x.Err != nil {
			return map[string]any{"Err": toPlain(x.Err)}
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
