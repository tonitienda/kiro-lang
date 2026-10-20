package runtimekit

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type FunctionType struct {
	Module     string
	Params     []ParamSpec
	ReturnType string
	Effects    []string
}

func (s FunctionType) String() string {
	var b strings.Builder
	b.WriteString("Fn[(")
	for i, p := range s.Params {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(p.Name + ":" + p.Type)
	}
	b.WriteString(")->" + s.ReturnType)
	effects := append([]string{}, s.Effects...)
	sort.Strings(effects)
	for _, effect := range effects {
		b.WriteString("!" + effect)
	}
	b.WriteString("]")
	return b.String()
}

func ParseFunctionType(raw string) (*FunctionType, error) {
	p, err := newBodyParser(raw)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(TokenIdent, "Fn"); err != nil {
		return nil, err
	}
	if _, err = p.expect(TokenSymbol, "["); err != nil {
		return nil, err
	}
	if _, err = p.expect(TokenSymbol, "("); err != nil {
		return nil, err
	}
	s := &FunctionType{}
	seen := map[string]bool{}
	for !p.peekText(")") {
		name, err := p.expect(TokenIdent, "")
		if err != nil {
			return nil, err
		}
		if seen[name.Text] {
			return nil, fmt.Errorf("duplicate function-type parameter %q", name.Text)
		}
		seen[name.Text] = true
		if _, err = p.expect(TokenSymbol, ":"); err != nil {
			return nil, err
		}
		typ, err := readValueType(p)
		if err != nil {
			return nil, err
		}
		s.Params = append(s.Params, ParamSpec{Name: name.Text, Type: typ})
		if p.peekText(")") {
			break
		}
		if _, err = p.expect(TokenSymbol, ","); err != nil {
			return nil, err
		}
	}
	p.next()
	if _, err = p.expect(TokenSymbol, "-"); err != nil {
		return nil, err
	}
	if _, err = p.expect(TokenSymbol, ">"); err != nil {
		return nil, err
	}
	s.ReturnType, err = readValueType(p)
	if err != nil {
		return nil, err
	}
	effects := map[string]bool{}
	for p.peekText("!") {
		p.next()
		name, e := p.expect(TokenIdent, "")
		if e != nil {
			return nil, e
		}
		if !knownRuntimeEffect(name.Text) || effects[name.Text] {
			return nil, fmt.Errorf("invalid or duplicate function-type effect %q", name.Text)
		}
		effects[name.Text] = true
		s.Effects = append(s.Effects, name.Text)
	}
	if _, err = p.expect(TokenSymbol, "]"); err != nil {
		return nil, err
	}
	if p.peek().Kind != TokenEOF {
		return nil, fmt.Errorf("trailing tokens in function type")
	}
	return s, nil
}

func readValueType(p *bodyParser) (string, error) {
	start := p.pos
	if p.peekText("?") {
		p.next()
	}
	base := p.next()
	if base.Kind != TokenIdent && !(base.Kind == TokenKeyword && base.Text == "nil") {
		return "", fmt.Errorf("expected value type, got %q", base.Text)
	}
	for p.peekText(".") {
		p.next()
		if _, err := p.expect(TokenIdent, ""); err != nil {
			return "", err
		}
	}
	if p.peekText("[") {
		depth := 1
		p.next()
		for depth > 0 {
			t := p.next()
			if t.Kind == TokenEOF {
				return "", fmt.Errorf("unterminated type")
			}
			if t.Kind == TokenSymbol && t.Text == "[" {
				depth++
			}
			if t.Kind == TokenSymbol && t.Text == "]" {
				depth--
			}
		}
	}
	var b strings.Builder
	for _, t := range p.tokens[start:p.pos] {
		b.WriteString(t.Text)
	}
	result := b.String()
	if err := validateTypeContract(result); err != nil {
		return "", err
	}
	return result, nil
}

func knownRuntimeEffect(name string) bool {
	switch name {
	case "env", "fs", "io", "log", "net", "panic", "proc", "time":
		return true
	}
	return false
}

func effectsAllowed(actual, allowed []string) error {
	for _, effect := range actual {
		found := false
		for _, bound := range allowed {
			if effect == bound {
				found = true
				break
			}
		}
		if !found {
			d := NewDiagnostic("KIRO_EFFECT_REQUIRED", fmt.Sprintf("effect !%s is not allowed by the caller/function-type contract", effect))
			d.Missing = []string{effect}
			d.Hint = "declare the effect or use a callable with a compatible effect contract"
			return d
		}
	}
	return nil
}

func compatibleFunctions(actual, expected FunctionType) error {
	if len(actual.Params) != len(expected.Params) {
		return fmt.Errorf("function parameter count differs: expected %s, got %s", expected.String(), actual.String())
	}
	for _, wanted := range expected.Params {
		found := false
		for _, provided := range actual.Params {
			if provided.Name == wanted.Name && contractAcceptsType(provided.Type, wanted.Type) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("function parameter %s:%s is incompatible with %s", wanted.Name, wanted.Type, actual.String())
		}
	}
	if !contractAcceptsType(expected.ReturnType, actual.ReturnType) {
		return fmt.Errorf("function return type: expected %s, got %s", expected.ReturnType, actual.ReturnType)
	}
	return effectsAllowed(actual.Effects, expected.Effects)
}

func builtinFunctionType(name string) (FunctionType, bool) {
	keys, ok := BuiltinParameters(name)
	if !ok {
		return FunctionType{}, false
	}
	s := FunctionType{ReturnType: "Any"}
	for _, key := range keys {
		s.Params = append(s.Params, ParamSpec{Name: key, Type: "Any"})
	}
	set := func(ret string, types ...string) {
		s.ReturnType = ret
		for i, t := range types {
			s.Params[i].Type = t
		}
	}
	switch name {
	case "list.map", "list.filter":
		set("List[Any]", "List[Any]", "Any")
	case "list.collect":
		set("List[Any]", "List[Any]", "Any")
	case "list.transduce":
		set("Any", "List[Any]", "Any", "Any", "Any")
	case "list.reduce":
		set("Any", "List[Any]", "Any", "Any")
	case "print", "println":
		set("nil", "Any")
		s.Effects = []string{"io"}
	case "Ok", "Err":
		set("R[Any,Any]", "Any")
	case "test.eq", "assert.equal_i32":
		set("nil", "Any", "Any")
	case "env.get_or":
		set("str", "str", "str")
		s.Effects = []string{"env"}
	case "log.info":
		set("nil", "str")
		s.Effects = []string{"log"}
	case "parse.i32":
		set("R[i32,str]", "str")
	case "cli.args":
		set("List[str]")
		s.Effects = []string{"proc"}
	case "fs.read_file":
		set("R[str,str]", "str")
		s.Effects = []string{"fs"}
	case "fs.write_file":
		set("R[nil,str]", "str", "str")
		s.Effects = []string{"fs"}
	case "http.text", "http.json":
		set("http.Resp", "i32", "str")
	case "http.not_found":
		set("http.Resp")
	case "http.with_header":
		set("http.Resp", "http.Resp", "str", "str")
	case "http.test_req":
		set("http.Req", "str", "str", "str")
	case "http.query":
		set("?str", "http.Req", "str")
	case "http.serve":
		set("R[nil,str]", "str", "Any")
		s.Effects = []string{"net"}
	case "json.encode":
		set("R[str,str]", "Any")
	case "json.decode":
		set("R[Any,str]", "str")
	case "ctx.background":
		set("ctx.Ctx")
	case "ctx.with_timeout_ms":
		set("ctx.Ctx", "ctx.Ctx", "i32")
	}
	return s, true
}

func signatureOf(fn FuncSpec, method bool) FunctionType {
	params := fn.Params
	if method && len(params) > 0 {
		params = params[1:]
	}
	return FunctionType{Params: append([]ParamSpec{}, params...), ReturnType: fn.ReturnType, Effects: fn.Effects, Module: fn.Module}
}

func genericParts(typ string, prefix string) ([]string, bool) {
	if !strings.HasPrefix(typ, prefix+"[") || !strings.HasSuffix(typ, "]") {
		return nil, false
	}
	raw := typ[len(prefix)+1 : len(typ)-1]
	depth := 0
	start := 0
	var out []string
	for i, c := range raw {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, raw[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, raw[start:])
	return out, true
}

func literalValue(raw string) (value, error) {
	p, err := newBodyParser(raw)
	if err != nil {
		return nil, err
	}
	ex, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if p.peek().Kind != TokenEOF {
		return nil, fmt.Errorf("defaults must be scalar literals")
	}
	switch x := ex.(type) {
	case *nilExpr:
		return nil, nil
	case *intExpr:
		return x.Value, nil
	case *stringExpr:
		if !strings.Contains(x.Value, "${") {
			return x.Value, nil
		}
	case *identExpr:
		if x.Name == "true" {
			return true, nil
		}
		if x.Name == "false" {
			return false, nil
		}
	}
	return nil, fmt.Errorf("defaults must be scalar literals without interpolation")
}

func (rt *runtime) validateValue(v value, typ string) error { return rt.validateValueIn(v, typ, "") }

func (rt *runtime) validateValueIn(v value, typ, module string) error {
	if typ == "Any" || typ == "" {
		return nil
	}
	if strings.HasPrefix(typ, "?") {
		if v == nil {
			return nil
		}
		return rt.validateValueIn(v, typ[1:], module)
	}
	if strings.HasPrefix(typ, "Fn[") {
		wanted, err := ParseFunctionType(typ)
		if err != nil {
			return err
		}
		actual, err := rt.callableType(v)
		if err != nil {
			return err
		}
		return compatibleFunctions(actual, rt.qualifySignature(*wanted, module))
	}
	if parts, ok := genericParts(typ, "R"); ok && len(parts) == 2 {
		res, ok := v.(resultValue)
		if !ok {
			return fmt.Errorf("expected %s, got %T", typ, v)
		}
		if res.Kind == resultErr {
			return rt.validateValueIn(res.Err, parts[1], module)
		}
		if res.Kind != resultOK {
			return fmt.Errorf("invalid result tag")
		}
		return rt.validateValueIn(res.Value, parts[0], module)
	}
	if parts, ok := genericParts(typ, "Transform"); ok && len(parts) == 2 {
		transform, ok := v.(*transformValue)
		if !ok {
			return fmt.Errorf("expected %s, got %T", typ, v)
		}
		if !contractAcceptsType(transform.InputType, parts[0]) || !contractAcceptsType(parts[1], transform.OutputType) {
			return fmt.Errorf("expected %s, got Transform[%s,%s]", typ, transform.InputType, transform.OutputType)
		}
		return nil
	}
	if parts, ok := genericParts(typ, "List"); ok && len(parts) == 1 {
		items, ok := v.([]value)
		if !ok {
			return fmt.Errorf("expected %s, got %T", typ, v)
		}
		if parts[0] == "Any" {
			return nil
		}
		for _, item := range items {
			if err := rt.validateValueIn(item, parts[0], module); err != nil {
				return err
			}
		}
		return nil
	}
	ok := false
	switch typ {
	case "nil":
		ok = v == nil
	case "str":
		_, ok = v.(string)
	case "bool":
		_, ok = v.(bool)
	case "i32":
		switch n := v.(type) {
		case int:
			ok = int64(n) >= math.MinInt32 && int64(n) <= math.MaxInt32
		case float64:
			ok = n == math.Trunc(n) && n >= math.MinInt32 && n <= math.MaxInt32
		}
	case "ctx.Ctx":
		_, ok = v.(string)
	default:
		return rt.validateRecord(v, typ, module)
	}
	if !ok {
		return fmt.Errorf("expected %s, got %T (%v)", typ, v, toPlain(v))
	}
	return nil
}

func validateTypeContract(raw string) error {
	raw = strings.TrimPrefix(raw, "?")
	if raw == "Transform" {
		return fmt.Errorf("Transform requires two types: Transform[Input,Output]")
	}
	if strings.HasPrefix(raw, "Transform[") {
		parts, ok := genericParts(raw, "Transform")
		if !ok || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("Transform requires two types: Transform[Input,Output]")
		}
		for _, part := range parts {
			parser, err := newBodyParser(part)
			if err != nil {
				return err
			}
			if _, err = readValueType(parser); err != nil {
				return fmt.Errorf("invalid Transform type argument: %w", err)
			}
			if parser.peek().Kind != TokenEOF {
				return fmt.Errorf("unexpected trailing tokens in Transform type argument %q", part)
			}
		}
		return nil
	}
	if raw == "Fn" {
		return fmt.Errorf("Fn requires a keyed signature: Fn[(value:i32)->i32]")
	}
	if strings.HasPrefix(raw, "Fn[") {
		_, err := ParseFunctionType(raw)
		return err
	}
	if start := strings.Index(raw, "["); start >= 0 {
		if parts, ok := genericParts(raw, raw[:start]); ok {
			for _, part := range parts {
				if err := validateTypeContract(part); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func specializeSignature(sig FunctionType, name string, typeArgs []string) (FunctionType, error) {
	if len(typeArgs) == 0 {
		return sig, nil
	}
	if name != "json.decode" || len(typeArgs) != 1 {
		return sig, fmt.Errorf("type arguments are supported only by json.decode[T]")
	}
	sig.ReturnType = "R[" + typeArgs[0] + ",str]"
	return sig, validateTypeContract(typeArgs[0])
}

// BuiltinContracts exposes the same unspecialized contracts used for execution.
// Generic/callback-dependent checks remain in specialization and collection rules.
func BuiltinContracts() map[string]FunctionType {
	out := map[string]FunctionType{}
	for name := range builtinParameters {
		signature, _ := builtinFunctionType(name)
		out[name] = signature
	}
	return out
}
