package runtimekit

import (
	"fmt"
	"io"
	"strings"
)

type inferredValue struct {
	Type     string
	Function *FunctionType
	Builtin  string
}
type functionalChecker struct {
	rt      *runtime
	module  string
	env     map[string]inferredValue
	effects []string
	returns string
	mutable map[string]bool
}

func ValidateDeclarations(spec ProgramSpec) error {
	rt := &runtime{}
	for _, mod := range spec.Modules {
		seenTypes := map[string]bool{}
		for _, typ := range mod.Types {
			if seenTypes[typ.Name] {
				return fmt.Errorf("%s: duplicate type %s", mod.Name, typ.Name)
			}
			seenTypes[typ.Name] = true
			if len(typ.Variants) > 0 && len(typ.Fields) > 0 {
				return fmt.Errorf("%s.%s cannot mix record fields and variants", mod.Name, typ.Name)
			}
			tags := map[string]bool{}
			for _, v := range typ.Variants {
				if tags[v.Name] {
					return fmt.Errorf("%s.%s: duplicate variant %s", mod.Name, typ.Name, v.Name)
				}
				tags[v.Name] = true
				fields := map[string]bool{}
				for _, f := range v.Fields {
					if fields[f.Name] {
						return fmt.Errorf("%s.%s.%s: duplicate field %s", mod.Name, typ.Name, v.Name, f.Name)
					}
					fields[f.Name] = true
					if err := validateTypeContract(f.Type); err != nil {
						return err
					}
				}
			}
			seen := map[string]bool{}
			for _, field := range typ.Fields {
				if seen[field.Name] {
					return fmt.Errorf("%s.%s: duplicate field %q", mod.Name, typ.Name, field.Name)
				}
				seen[field.Name] = true
				if err := validateTypeContract(field.Type); err != nil {
					return fmt.Errorf("%s.%s field %s: %w", mod.Name, typ.Name, field.Name, err)
				}
			}
		}
		for _, fn := range mod.Funcs {
			for _, param := range fn.Params {
				if err := validateTypeContract(param.Type); err != nil {
					return fmt.Errorf("%s.%s parameter %s: %w", mod.Name, fn.Name, param.Name, err)
				}
				if param.Default != "" {
					v, err := literalValue(param.Default)
					if err == nil {
						err = rt.validateValue(v, param.Type)
					}
					if err != nil {
						return fmt.Errorf("%s.%s default %s: %w", mod.Name, fn.Name, param.Name, err)
					}
				}
			}
			if err := validateTypeContract(fn.ReturnType); err != nil {
				return fmt.Errorf("%s.%s return type: %w", mod.Name, fn.Name, err)
			}
		}
	}
	return nil
}

// ValidateFunctionalContracts checks every body in the executable grammar.
// Inference remains incremental; return-path analysis is conservative.
func ValidateFunctionalContracts(spec ProgramSpec) error {
	if err := ValidateDeclarations(spec); err != nil {
		return err
	}
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	for _, mod := range spec.Modules {
		for _, fn := range mod.Funcs {
			parsed, err := rt.parseFunc(fn)
			if err != nil {
				return FunctionDiagnostic(fmt.Errorf("body: %w", err), fn, "KIRO_SYNTAX")
			}
			c := &functionalChecker{rt: rt, module: mod.Name, env: map[string]inferredValue{}, effects: fn.Effects, returns: rt.qualifyType(fn.ReturnType, mod.Name), mutable: map[string]bool{}}
			for _, p := range fn.Params {
				v := inferredValue{Type: rt.qualifyType(p.Type, mod.Name)}
				if strings.HasPrefix(p.Type, "Fn[") {
					v.Function, _ = ParseFunctionType(v.Type)
				}
				c.env[p.Name] = v
			}
			if err := c.statements(parsed.stmts); err != nil {
				return FunctionDiagnostic(err, fn, "KIRO_CHECK_FAILED")
			}
			if fn.BlockBody && needsReturn(fn.ReturnType) && !statementsReturn(parsed.stmts) {
				d := NewDiagnostic("KIRO_MISSING_RETURN", "not every path returns "+fn.ReturnType)
				d.Expected = fn.ReturnType
				d.Actual = "fallthrough nil"
				d.Hint = "return a value on every path; loops may be empty and ? may succeed"
				return FunctionDiagnostic(d, fn, "KIRO_CHECK_FAILED")
			}
		}
	}
	return nil
}

func (c *functionalChecker) statements(stmts []stmt) error {
	for _, statement := range stmts {
		switch n := statement.(type) {
		case *letStmt:
			v, err := c.infer(n.Value)
			if err != nil {
				return err
			}
			c.env[n.Name] = v
			c.mutable[n.Name] = n.Mutable
		case *assignStmt:
			if !c.mutable[n.Name] {
				return fmt.Errorf("cannot assign to immutable or unknown binding %s", n.Name)
			}
			v, err := c.infer(n.Value)
			if err != nil {
				return err
			}
			if !compatibleValueTypes(v.Type, c.env[n.Name].Type) {
				return fmt.Errorf("assignment changes type of %s from %s to %s", n.Name, c.env[n.Name].Type, v.Type)
			}
			c.env[n.Name] = v
		case *exprStmt:
			if _, err := c.infer(n.Value); err != nil {
				return err
			}
		case *returnStmt:
			v, err := c.infer(n.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(c.returns, "Fn[") && !compatibleValueTypes(v.Type, c.returns) {
				return TypeMismatch(fmt.Sprintf("return type: expected %s, got %s", c.returns, v.Type), c.returns, v.Type)
			}
			if strings.HasPrefix(c.returns, "Fn[") {
				wanted, _ := ParseFunctionType(c.returns)
				if v.Function == nil {
					if v.Type == "Any" {
						continue
					}
					return fmt.Errorf("expected a function return value")
				}
				if err := compatibleFunctions(*v.Function, *wanted); err != nil {
					return err
				}
			}
		case *forStmt:
			items, err := c.infer(n.Items)
			if err != nil {
				return err
			}
			parts, ok := genericParts(items.Type, "List")
			if !ok || len(parts) != 1 {
				return fmt.Errorf("for requires a known List, got %s", items.Type)
			}
			child := *c
			child.env = copyInferred(c.env)
			child.mutable = copyMutable(c.mutable)
			child.env[n.Name] = inferredValue{Type: parts[0]}
			child.mutable[n.Name] = false
			if err := child.statements(n.Body); err != nil {
				return err
			}
		case *groupStmt:
			if err := c.statements(n.Body); err != nil {
				return err
			}
		case *whenStmt:
			if _, err := c.checkWhen(n.Value, n.Cases); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyInferred(env map[string]inferredValue) map[string]inferredValue {
	copy := map[string]inferredValue{}
	for key, v := range env {
		copy[key] = v
	}
	return copy
}
func functionInference(sig FunctionType) inferredValue {
	return inferredValue{Type: sig.String(), Function: &sig}
}

func (c *functionalChecker) infer(expression expr) (inferredValue, error) {
	unknown := inferredValue{Type: "Any"}
	switch n := expression.(type) {
	case *intExpr:
		return inferredValue{Type: "i32"}, nil
	case *stringExpr:
		_, err := interpolateText(n.Value, func(raw string) (string, error) {
			expression, err := interpolationExpr(raw)
			if err != nil {
				return "", err
			}
			_, err = c.infer(expression)
			return "", err
		})
		return inferredValue{Type: "str"}, err
	case *nilExpr:
		return inferredValue{Type: "nil"}, nil
	case *defaultExpr:
		return unknown, fmt.Errorf("default is only valid as an explicit call argument")
	case *identExpr:
		if v, ok := c.env[n.Name]; ok {
			return v, nil
		}
		if n.Name == "true" || n.Name == "false" {
			return inferredValue{Type: "bool"}, nil
		}
		if fn, ok := c.rt.modules[c.module].funcs[n.Name]; ok {
			return functionInference(c.rt.functionType(fn, false)), nil
		}
		if sig, ok := builtinFunctionType(n.Name); ok {
			v := functionInference(sig)
			v.Builtin = n.Name
			return v, nil
		}
		if v, ok := c.rt.modules[c.module].consts[n.Name]; ok {
			switch v.(type) {
			case string:
				return inferredValue{Type: "str"}, nil
			case int:
				return inferredValue{Type: "i32"}, nil
			case bool:
				return inferredValue{Type: "bool"}, nil
			}
		}
		d := NewDiagnostic("KIRO_UNKNOWN_SYMBOL", fmt.Sprintf("unknown identifier %q", n.Name))
		d.Target = n.Name
		return unknown, d
	case *bindingExpr:
		return unknown, fmt.Errorf("let payload bindings are only valid in when patterns")
	case *selectorExpr:
		if name := expressionName(n); name != "" {
			root := strings.Split(name, ".")[0]
			if _, local := c.env[root]; !local {
				if ctor, ok := c.rt.variantConstructor(name, c.module); ok {
					return functionInference(c.rt.qualifySignature(ctor.signature(), ctor.Schema.Module)), nil
				}
				if schema, err := c.rt.recordSchema(expressionName(n.Left), c.module); err == nil && len(schema.Type.Variants) > 0 {
					return unknown, fmt.Errorf("unknown variant %s", name)
				}
			}
		}
		if root, ok := n.Left.(*identExpr); ok {
			if _, local := c.env[root.Name]; !local {
				if root.Name == "list" && n.Name == "fold" {
					return unknown, fmt.Errorf("list.fold was renamed to list.reduce; keep items, initial and reduce keys")
				}
				if sig, ok := builtinFunctionType(root.Name + "." + n.Name); ok {
					v := functionInference(sig)
					v.Builtin = root.Name + "." + n.Name
					return v, nil
				}
				if mod := c.rt.modules[root.Name]; mod != nil {
					if fn, ok := mod.funcs[n.Name]; ok {
						return functionInference(c.rt.functionType(fn, false)), nil
					}
				}
			}
		}
		left, err := c.infer(n.Left)
		if err != nil {
			return unknown, err
		}
		if method, ok := c.rt.lookupMethod(left.Type, n.Name); ok {
			return functionInference(c.rt.functionType(method, true)), nil
		}
		module := c.module
		typeName := left.Type
		if dot := strings.LastIndex(typeName, "."); dot >= 0 {
			module = typeName[:dot]
			typeName = typeName[dot+1:]
		}
		if mod := c.rt.modules[module]; mod != nil {
			if record, ok := mod.types[typeName]; ok {
				for _, field := range record.Fields {
					if field.Name == n.Name {
						if strings.HasPrefix(field.Type, "Fn[") {
							sig, err := ParseFunctionType(c.rt.qualifyType(field.Type, module))
							if err != nil {
								return unknown, err
							}
							return functionInference(*sig), nil
						}
						return inferredValue{Type: c.rt.qualifyType(field.Type, module)}, nil
					}
				}
			}
		}
		return unknown, nil
	case *partialExpr:
		return c.call(n.Call, true)
	case *callExpr:
		return c.call(n, false)
	case *unwrapExpr:
		enclosing, ok := genericParts(c.returns, "R")
		if !ok || len(enclosing) != 2 {
			return unknown, fmt.Errorf("? requires an R[T,E] return type in the enclosing function")
		}
		v, err := c.infer(n.Value)
		if err != nil {
			return unknown, err
		}
		if parts, ok := genericParts(v.Type, "R"); ok && len(parts) == 2 {
			if parts[1] != "Any" && !contractAcceptsType(enclosing[1], parts[1]) {
				return unknown, TypeMismatch(fmt.Sprintf("? error type %s is incompatible with enclosing error type %s", parts[1], enclosing[1]), enclosing[1], parts[1])
			}
			if strings.HasPrefix(parts[0], "Fn[") {
				signature, err := ParseFunctionType(parts[0])
				if err != nil {
					return unknown, err
				}
				return functionInference(*signature), nil
			}
			return inferredValue{Type: parts[0]}, nil
		}
		if v.Type != "Any" {
			return unknown, TypeMismatch(fmt.Sprintf("invalid ? on non-result value of type %s", v.Type), "R[T,E]", v.Type)
		}
		return unknown, nil
	case *structExpr:
		schema, err := c.rt.recordSchema(n.TypeName, c.module)
		if err != nil {
			return unknown, err
		}
		if len(schema.Type.Variants) > 0 {
			return unknown, fmt.Errorf("variant type %s requires a named constructor", n.TypeName)
		}
		keys := []string{}
		for _, field := range n.Fields {
			keys = append(keys, field.Name)
		}
		if err := checkRecordKeys(schema.Type, keys, true); err != nil {
			return unknown, err
		}
		for _, field := range n.Fields {
			v, err := c.infer(field.Value)
			if err != nil {
				return unknown, err
			}
			expected := c.rt.qualifyType(schema.fieldType(field.Name), schema.Module)
			if !compatibleValueTypes(v.Type, expected) {
				return unknown, TypeMismatch(fmt.Sprintf("%s.%s: expected %s, got %s", n.TypeName, field.Name, expected, v.Type), expected, v.Type)
			}
		}
		return inferredValue{Type: schema.Module + "." + schema.Type.Name}, nil
	case *listExpr:
		element := "Any"
		for i, item := range n.Items {
			v, err := c.infer(item)
			if err != nil {
				return unknown, err
			}
			if i == 0 {
				element = v.Type
			} else if v.Type != element {
				element = "Any"
			}
		}
		return inferredValue{Type: "List[" + element + "]"}, nil
	case *binaryExpr:
		left, err := c.infer(n.Left)
		if err != nil {
			return unknown, err
		}
		if _, err := c.infer(n.Right); err != nil {
			return unknown, err
		}
		switch n.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			return inferredValue{Type: "bool"}, nil
		}
		return left, nil
	case *spawnExpr:
		child := *c
		child.returns = ""
		return child.infer(n.Value)
	case *awaitExpr:
		return c.infer(n.Value)
	case *whenExpr:
		return c.checkWhen(n.Value, n.Cases)
	case *sequenceExpr:
		var last = inferredValue{Type: "nil"}
		for _, item := range n.Items {
			v, err := c.infer(item)
			if err != nil {
				return unknown, err
			}
			last = v
		}
		return last, nil
	}
	return unknown, nil
}

func (c *functionalChecker) call(call *callExpr, partial bool) (inferredValue, error) {
	unknown := inferredValue{Type: "Any"}
	inference := *c
	if partial {
		inference.effects = nil
	}
	target, err := inference.infer(call.Callee)
	if err != nil {
		return unknown, err
	}
	if target.Function == nil {
		for _, arg := range call.Args {
			if _, err := c.infer(arg.Value); err != nil {
				return unknown, err
			}
		}
		return unknown, nil
	}
	sig := *target.Function
	name := target.Builtin
	typeArgs := append([]string{}, call.TypeArgs...)
	if name == "json.decode" {
		if len(typeArgs) != 1 {
			return unknown, fmt.Errorf("json.decode requires one explicit schema type")
		}
		qualified, e := c.rt.jsonSchemaType(typeArgs[0], c.module, map[string]bool{})
		if e != nil {
			return unknown, e
		}
		typeArgs[0] = qualified
	}
	sig, err = specializeSignature(sig, name, typeArgs)
	if err != nil {
		return unknown, err
	}
	params := map[string]ParamSpec{}
	names := []string{}
	keys := []string{}
	for _, p := range sig.Params {
		params[p.Name] = p
		names = append(names, p.Name)
	}
	for _, arg := range call.Args {
		keys = append(keys, arg.Name)
	}
	if _, err := CheckArgumentKeys(names, keys, partial); err != nil {
		return unknown, err
	}
	if !partial {
		if err := effectsAllowed(sig.Effects, c.effects); err != nil {
			return unknown, err
		}
	}
	bound := map[string]bool{}
	values := map[string]inferredValue{}
	for _, arg := range call.Args {
		parameter := params[arg.Name]
		bound[arg.Name] = true
		if _, defaulted := arg.Value.(*defaultExpr); defaulted {
			if parameter.Default == "" {
				return unknown, fmt.Errorf("argument %s has no declared default", arg.Name)
			}
			continue
		}
		child := *c
		if partial {
			child.effects = nil
		}
		v, err := child.infer(arg.Value)
		if err != nil {
			return unknown, err
		}
		values[arg.Name] = v
		if !strings.HasPrefix(parameter.Type, "Fn[") && !compatibleValueTypes(v.Type, parameter.Type) {
			return unknown, TypeMismatch(fmt.Sprintf("argument %s: expected %s, got %s", arg.Name, parameter.Type, v.Type), parameter.Type, v.Type)
		}
		if strings.HasPrefix(parameter.Type, "Fn[") {
			wanted, err := ParseFunctionType(parameter.Type)
			if err != nil {
				return unknown, err
			}
			if v.Function == nil {
				if v.Type == "Any" {
					continue
				}
				return unknown, fmt.Errorf("argument %s must be a typed function", arg.Name)
			}
			if err := compatibleFunctions(*v.Function, *wanted); err != nil {
				return unknown, err
			}
		}
	}
	if partial {
		remaining := FunctionType{ReturnType: sig.ReturnType, Effects: sig.Effects}
		for _, p := range sig.Params {
			if !bound[p.Name] {
				remaining.Params = append(remaining.Params, p)
			}
		}
		return functionInference(remaining), nil
	}
	if name == "Ok" {
		return inferredValue{Type: "R[" + values["value"].Type + ",Any]"}, nil
	}
	if name == "Err" {
		return inferredValue{Type: "R[Any," + values["error"].Type + "]"}, nil
	}
	if result, handled, err := c.transformCall(name, values); handled {
		return result, err
	}
	if strings.HasPrefix(name, "list.") {
		key := "transform"
		if name == "list.filter" {
			key = "predicate"
		}
		if name == "list.reduce" {
			key = "reduce"
		}
		var callback *FunctionType
		var itemType = "Any"
		var initialType = "Any"
		for _, arg := range call.Args {
			v := values[arg.Name]
			if arg.Name == key {
				if v.Type == "Any" {
					return inferredValue{Type: sig.ReturnType}, nil
				}
				callback = v.Function
			}
			if arg.Name == "items" {
				if parts, ok := genericParts(v.Type, "List"); ok && len(parts) == 1 {
					itemType = parts[0]
				}
			}
			if arg.Name == "initial" {
				initialType = v.Type
			}
		}
		if callback == nil {
			return unknown, fmt.Errorf("list callback must be a typed callable")
		}
		if len(callback.Effects) > 0 {
			return unknown, fmt.Errorf("list operations require pure callbacks")
		}
		expected := []string{"value"}
		if key == "reduce" {
			expected = []string{"state", "value"}
		}
		names := []string{}
		for _, p := range callback.Params {
			names = append(names, p.Name)
		}
		if _, e := ArgumentOrder(names, expected); e != nil {
			return unknown, e
		}
		if key == "predicate" && callback.ReturnType != "bool" {
			return unknown, fmt.Errorf("filter predicate must return bool")
		}
		for _, p := range callback.Params {
			if p.Name == "value" && !compatibleValueTypes(itemType, p.Type) {
				return unknown, fmt.Errorf("list callback value: expected %s, got %s", p.Type, itemType)
			}
			if p.Name == "state" {
				if p.Type != callback.ReturnType {
					return unknown, fmt.Errorf("reduce reducer return type must match state type")
				}
				if !compatibleValueTypes(initialType, p.Type) {
					return unknown, fmt.Errorf("reduce initial: expected %s, got %s", p.Type, initialType)
				}
			}
		}
		switch key {
		case "transform":
			return inferredValue{Type: "List[" + callback.ReturnType + "]"}, nil
		case "predicate":
			return inferredValue{Type: "List[" + itemType + "]"}, nil
		case "reduce":
			return inferredValue{Type: callback.ReturnType}, nil
		}
	}

	if strings.HasPrefix(sig.ReturnType, "Fn[") {
		ret, err := ParseFunctionType(sig.ReturnType)
		if err != nil {
			return unknown, err
		}
		return functionInference(*ret), nil
	}
	return inferredValue{Type: sig.ReturnType}, nil
}

func copyMutable(source map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, v := range source {
		out[key] = v
	}
	return out
}

// Unknown expression types are deferred to checked runtime boundaries.
func compatibleValueTypes(actual, expected string) bool {
	if actual == "Any" || expected == "Any" || actual == expected {
		return true
	}
	// Qualified record references retain the existing nominal runtime boundary;
	// module-distinct type identity remains a separate implementation step.
	if strings.HasSuffix(actual, "."+expected) || strings.HasSuffix(expected, "."+actual) {
		return true
	}
	if strings.HasPrefix(expected, "?") {
		return actual == "nil" || compatibleValueTypes(actual, expected[1:])
	}
	if strings.HasPrefix(actual, "Fn[") && strings.HasPrefix(expected, "Fn[") {
		a, e1 := ParseFunctionType(actual)
		b, e2 := ParseFunctionType(expected)
		return e1 == nil && e2 == nil && compatibleFunctions(*a, *b) == nil
	}
	if actualParts, ok := genericParts(actual, "Transform"); ok && len(actualParts) == 2 {
		if expectedParts, ok := genericParts(expected, "Transform"); ok && len(expectedParts) == 2 {
			return contractAcceptsType(actualParts[0], expectedParts[0]) && contractAcceptsType(expectedParts[1], actualParts[1])
		}
	}
	for _, generic := range []string{"R", "List"} {
		a, ok := genericParts(actual, generic)
		b, ok2 := genericParts(expected, generic)
		if ok && ok2 && len(a) == len(b) {
			for i := range a {
				if !compatibleValueTypes(a[i], b[i]) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (c *functionalChecker) branch(subject expr, arm whenCase, excludedNil bool) *functionalChecker {
	child := *c
	child.env = copyInferred(c.env)
	child.mutable = copyMutable(c.mutable)
	if id, ok := subject.(*identExpr); ok {
		current := child.env[id.Name]
		if strings.HasPrefix(current.Type, "?") {
			if _, isNil := arm.Pattern.(*nilExpr); isNil {
				current.Type = "nil"
			} else if arm.Wildcard && excludedNil {
				current.Type = current.Type[1:]
			}
			child.env[id.Name] = current
		}
	}
	return &child
}
