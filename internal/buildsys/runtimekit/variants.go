package runtimekit

import (
	"fmt"
	"strings"
)

type variantValue struct {
	Type   string
	Module string
	Tag    string
	Fields map[string]any
}

type variantConstructor struct {
	Schema  recordSchema
	Variant VariantSpec
}

func (v variantConstructor) signature() FunctionType {
	sig := FunctionType{ReturnType: v.Schema.Module + "." + v.Schema.Type.Name, Module: v.Schema.Module}
	for _, f := range v.Variant.Fields {
		sig.Params = append(sig.Params, ParamSpec{Name: f.Name, Type: f.Type})
	}
	return sig
}

func (rt *runtime) variantConstructor(name, module string) (variantConstructor, bool) {
	dot := strings.LastIndex(name, ".")
	if dot < 0 {
		return variantConstructor{}, false
	}
	schema, err := rt.recordSchema(name[:dot], module)
	if err != nil {
		return variantConstructor{}, false
	}
	for _, v := range schema.Type.Variants {
		if v.Name == name[dot+1:] {
			return variantConstructor{schema, v}, true
		}
	}
	return variantConstructor{}, false
}

func expressionName(e expr) string {
	switch x := e.(type) {
	case *identExpr:
		return x.Name
	case *selectorExpr:
		if left := expressionName(x.Left); left != "" {
			return left + "." + x.Name
		}
	}
	return ""
}

func (rt *runtime) validateVariant(v value, schema recordSchema) error {
	x, ok := v.(*variantValue)
	if !ok || x.Module != schema.Module || x.Type != schema.Type.Name {
		return fmt.Errorf("expected variant of %s.%s, got %T", schema.Module, schema.Type.Name, v)
	}
	for _, tag := range schema.Type.Variants {
		if tag.Name != x.Tag {
			continue
		}
		fields := TypeSpec{Name: x.Type + "." + x.Tag, Fields: tag.Fields}
		keys := []string{}
		for k := range x.Fields {
			keys = append(keys, k)
		}
		if err := checkRecordKeys(fields, keys, true); err != nil {
			return err
		}
		for _, f := range tag.Fields {
			if err := rt.validateValueIn(x.Fields[f.Name], f.Type, schema.Module); err != nil {
				return fmt.Errorf("%s.%s: %w", x.Tag, f.Name, err)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown variant %s.%s", x.Type, x.Tag)
}

// Patterns never execute calls. Explicit let binders and _ accept a payload;
// literal patterns compare values. Nested constructor patterns use the same rules.
func (ctx *execContext) matchPattern(pattern expr, v value, bindings map[string]value) (bool, error) {
	switch p := pattern.(type) {
	case *bindingExpr:
		if _, exists := bindings[p.Name]; exists {
			return false, fmt.Errorf("duplicate pattern binding %s", p.Name)
		}
		bindings[p.Name] = v
		return true, nil
	case *identExpr:
		if p.Name == "_" {
			return true, nil
		}
		if p.Name != "true" && p.Name != "false" {
			return false, fmt.Errorf("use let %s to bind a pattern payload", p.Name)
		}
	case *callExpr:
		name := expressionName(p.Callee)
		fields := map[string]any{}
		names := []string{}
		switch name {
		case "Ok", "Err":
			x, ok := v.(resultValue)
			if !ok {
				return false, fmt.Errorf("%s pattern requires a result", name)
			}
			if name == "Ok" {
				if x.Kind != resultOK {
					return false, nil
				}
				fields["value"] = x.Value
				names = []string{"value"}
			} else {
				if x.Kind != resultErr {
					return false, nil
				}
				fields["error"] = x.Err
				names = []string{"error"}
			}
		default:
			constructor, ok := ctx.rt.variantConstructor(name, ctx.module)
			if !ok {
				return false, fmt.Errorf("unknown variant pattern %s", name)
			}
			x, ok := v.(*variantValue)
			if !ok || x.Module != constructor.Schema.Module || x.Type != constructor.Schema.Type.Name || x.Tag != constructor.Variant.Name {
				return false, nil
			}
			fields = x.Fields
			for _, f := range constructor.Variant.Fields {
				names = append(names, f.Name)
			}
		}
		keys := []string{}
		for _, a := range p.Args {
			keys = append(keys, a.Name)
		}
		if len(p.TypeArgs) > 0 {
			return false, fmt.Errorf("patterns do not accept type arguments")
		}
		if _, err := ArgumentOrder(names, keys); err != nil {
			return false, err
		}
		for _, a := range p.Args {
			matched, err := ctx.matchPattern(a.Value, fields[a.Name], bindings)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	case *intExpr, *stringExpr, *nilExpr:
	default:
		return false, fmt.Errorf("unsupported pattern %T", pattern)
	}
	// A string pattern is literal data, including ${...}; never interpolate it.
	if s, ok := pattern.(*stringExpr); ok {
		return valuesEqual(v, s.Value), nil
	}
	literal, err := ctx.eval(pattern)
	return valuesEqual(v, literal), err
}

func (c *functionalChecker) pattern(pattern expr, typ string, child *functionalChecker, bindings map[string]bool) (string, bool, error) {
	switch p := pattern.(type) {
	case *bindingExpr:
		if bindings[p.Name] {
			return "", false, fmt.Errorf("duplicate pattern binding %s", p.Name)
		}
		bindings[p.Name] = true
		v := inferredValue{Type: typ}
		if strings.HasPrefix(typ, "Fn[") {
			v.Function, _ = ParseFunctionType(typ)
		}
		child.env[p.Name] = v
		child.mutable[p.Name] = false
		return "", true, nil
	case *identExpr:
		if p.Name == "_" {
			return "", true, nil
		}
		if p.Name != "true" && p.Name != "false" {
			return "", false, fmt.Errorf("use let %s to bind a pattern payload", p.Name)
		}
	case *callExpr:
		name := expressionName(p.Callee)
		params := []ParamSpec{}
		tag := name
		if name == "Ok" || name == "Err" {
			parts, ok := genericParts(typ, "R")
			if !ok || len(parts) != 2 {
				return "", false, fmt.Errorf("%s pattern requires a known R[T,E] subject", name)
			}
			if name == "Ok" {
				params = []ParamSpec{{Name: "value", Type: parts[0]}}
			} else {
				params = []ParamSpec{{Name: "error", Type: parts[1]}}
			}
		} else {
			ctor, ok := c.rt.variantConstructor(name, c.module)
			if !ok {
				return "", false, fmt.Errorf("unknown variant pattern %s", name)
			}
			expected := ctor.Schema.Module + "." + ctor.Schema.Type.Name
			if typ != expected {
				return "", false, fmt.Errorf("pattern %s belongs to %s, subject has type %s", name, expected, typ)
			}
			tag = ctor.Variant.Name
			for _, f := range ctor.Variant.Fields {
				params = append(params, ParamSpec{Name: f.Name, Type: c.rt.qualifyType(f.Type, ctor.Schema.Module)})
			}
		}
		if len(p.TypeArgs) > 0 {
			return "", false, fmt.Errorf("patterns do not accept type arguments")
		}
		names, keys := []string{}, []string{}
		types := map[string]string{}
		for _, f := range params {
			names = append(names, f.Name)
			types[f.Name] = f.Type
		}
		for _, a := range p.Args {
			keys = append(keys, a.Name)
		}
		if _, err := ArgumentOrder(names, keys); err != nil {
			return "", false, err
		}
		total := true
		for _, a := range p.Args {
			_, all, err := c.pattern(a.Value, types[a.Name], child, bindings)
			if err != nil {
				return "", false, err
			}
			total = total && all
		}
		return tag, total, nil
	case *intExpr, *stringExpr, *nilExpr:
	default:
		return "", false, fmt.Errorf("unsupported pattern %T", pattern)
	}
	actual := "str"
	switch pattern.(type) {
	case *intExpr:
		actual = "i32"
	case *nilExpr:
		actual = "nil"
	case *identExpr:
		actual = "bool"
	}
	if !compatibleValueTypes(actual, typ) {
		return "", false, fmt.Errorf("pattern type %s does not match %s", actual, typ)
	}
	return "", false, nil
}

func (c *functionalChecker) checkWhen(subject expr, arms []whenCase) (inferredValue, error) {
	unknown := inferredValue{Type: "Any"}
	value, err := c.infer(subject)
	if err != nil {
		return unknown, err
	}
	required := []string{}
	strict := false
	if parts, ok := genericParts(value.Type, "R"); ok && len(parts) == 2 {
		required = []string{"Ok", "Err"}
	}
	if schema, e := c.rt.recordSchema(value.Type, c.module); e == nil && len(schema.Type.Variants) > 0 {
		strict = true
		for _, v := range schema.Type.Variants {
			required = append(required, v.Name)
		}
	}
	covered := map[string]bool{}
	wildcard, excludedNil := false, false
	resultType := ""
	for _, arm := range arms {
		if wildcard {
			return unknown, fmt.Errorf("unreachable when arm after wildcard")
		}
		child := c.branch(subject, arm, excludedNil)
		if arm.Wildcard {
			if strict {
				return unknown, fmt.Errorf("variant matches require explicit exhaustive arms; wildcard hides future variants")
			}
			wildcard = true
		} else {
			tag, total, e := c.pattern(arm.Pattern, value.Type, child, map[string]bool{})
			if e != nil {
				return unknown, e
			}
			if tag != "" && covered[tag] {
				return unknown, fmt.Errorf("unreachable duplicate variant arm %s", tag)
			}
			if tag != "" && total {
				covered[tag] = true
			}
		}
		if e := child.statements(arm.Body); e != nil {
			return unknown, e
		}
		if arm.Expr != nil {
			v, e := child.infer(arm.Expr)
			if e != nil {
				return unknown, e
			}
			if resultType == "" {
				resultType = v.Type
			} else if resultType != v.Type {
				return unknown, fmt.Errorf("when arm types differ: %s and %s", resultType, v.Type)
			}
		}
		if _, ok := arm.Pattern.(*nilExpr); ok {
			excludedNil = true
		}
	}
	missing := []string{}
	for _, tag := range required {
		if !covered[tag] && !wildcard {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		return unknown, fmt.Errorf("non-exhaustive when on %s: missing %s (bind payloads with let or _; use a nested when for payload decisions)", value.Type, strings.Join(missing, ", "))
	}
	if resultType != "" {
		v := inferredValue{Type: resultType}
		if strings.HasPrefix(resultType, "Fn[") {
			v.Function, _ = ParseFunctionType(resultType)
		}
		return v, nil
	}
	return unknown, nil
}
