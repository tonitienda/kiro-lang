package runtimekit

import (
	"fmt"
	"strings"
)

// A declared Fn boundary exposes exactly its promised signature, including its
// effect bound and absence of default metadata, even if the target is more permissive.
type contractedValue struct {
	Target    value
	Signature FunctionType
}

func ascribeValue(v value, typ string, modules ...string) value {
	module := ""
	if len(modules) > 0 {
		module = modules[0]
	}
	if parts, ok := genericParts(typ, "Transform"); ok && len(parts) == 2 {
		if transform, ok := v.(*transformValue); ok {
			return &transformValue{InputType: parts[0], OutputType: parts[1], Steps: transform.Steps}
		}
	}
	if !strings.HasPrefix(typ, "Fn[") {
		return v
	}
	if signature, err := ParseFunctionType(typ); err == nil {
		signature.Module = module
		if existing, ok := v.(*contractedValue); ok && existing.Signature.String() == signature.String() && existing.Signature.Module == signature.Module {
			return v
		}
		return &contractedValue{Target: v, Signature: *signature}
	}
	return v
}

type partialValue struct {
	TypeArgs  []string
	Target    value
	Bound     map[string]value
	Signature FunctionType
}

func (rt *runtime) callableType(callee value) (FunctionType, error) {
	switch fn := callee.(type) {
	case builtinValue:
		s, ok := builtinFunctionType(fn.Name)
		if !ok {
			return s, fmt.Errorf("missing signature for %s", fn.Name)
		}
		return s, nil
	case functionValue:
		mod := rt.modules[fn.Module]
		if mod == nil {
			return FunctionType{}, fmt.Errorf("unknown module %s", fn.Module)
		}
		f, ok := mod.funcs[fn.Name]
		if !ok {
			return FunctionType{}, fmt.Errorf("unknown function %s.%s", fn.Module, fn.Name)
		}
		return rt.functionType(f, false), nil
	case methodValue:
		return rt.functionType(fn.Method, true), nil
	case *contractedValue:
		return fn.Signature, nil
	case *partialValue:
		return fn.Signature, nil
	case typeSentinel:
		return FunctionType{}, fmt.Errorf("record constructors use %s{field:value}, not function-call syntax", string(fn))
	}
	return FunctionType{}, fmt.Errorf("expression is not callable: %T", callee)
}

func (rt *runtime) invokeCallable(callee value, args []value) (value, error) {
	sig, err := rt.callableType(callee)
	if err != nil {
		return nil, err
	}
	if len(args) != len(sig.Params) {
		return nil, fmt.Errorf("callable argument count mismatch")
	}
	for i, param := range sig.Params {
		if err := rt.validateValueIn(args[i], param.Type, sig.Module); err != nil {
			return nil, fmt.Errorf("argument %s: %w", param.Name, err)
		}
	}
	var result value
	switch fn := callee.(type) {
	case builtinValue:
		ctx := &execContext{rt: rt, module: sig.Module, locals: map[string]value{}, effects: sig.Effects}
		result, err = fn.Invoke(ctx, nil, args)
	case functionValue:
		result, err = rt.callUser(rt.modules[fn.Module].funcs[fn.Name], fn.Module, nil, args)
	case methodValue:
		result, err = rt.callUser(fn.Method, fn.Method.Module, fn.Receiver, args)
	case *contractedValue:
		actual, e := rt.callableType(fn.Target)
		if e != nil {
			return nil, e
		}
		keys := []string{}
		names := []string{}
		for _, p := range fn.Signature.Params {
			keys = append(keys, p.Name)
		}
		for _, p := range actual.Params {
			names = append(names, p.Name)
		}
		order, e := ArgumentOrder(names, keys)
		if e != nil {
			return nil, e
		}
		reordered := make([]value, len(order))
		for i, index := range order {
			reordered[i] = args[index]
		}
		result, err = rt.invokeCallable(fn.Target, reordered)
	case *partialValue:
		original, e := rt.callableType(fn.Target)
		if e != nil {
			return nil, e
		}
		values := map[string]value{}
		for name, v := range fn.Bound {
			values[name], err = freezeValue(v)
			if err != nil {
				return nil, err
			}
		}
		for i, param := range fn.Signature.Params {
			values[param.Name] = args[i]
		}
		combined := make([]value, len(original.Params))
		for i, param := range original.Params {
			combined[i] = values[param.Name]
		}
		if builtin, ok := fn.Target.(builtinValue); ok && len(fn.TypeArgs) > 0 {
			context := &execContext{rt: rt, locals: map[string]value{}, effects: original.Effects}
			result, err = builtin.Invoke(context, fn.TypeArgs, combined)
		} else {
			result, err = rt.invokeCallable(fn.Target, combined)
		}
	}
	if err != nil {
		return nil, err
	}
	if err = rt.validateValueIn(result, sig.ReturnType, sig.Module); err != nil {
		return nil, fmt.Errorf("return value: %w", err)
	}
	return ascribeValue(result, sig.ReturnType, sig.Module), nil
}

func (ctx *execContext) evaluateCall(call *callExpr, partial bool) (value, error) {
	evaluation := *ctx
	if partial {
		evaluation.effects = nil
	}
	callee, err := evaluation.eval(call.Callee)
	if err != nil {
		return nil, err
	}
	sig, err := ctx.rt.callableType(callee)
	if err != nil {
		return nil, err
	}
	name := ""
	if builtin, ok := callee.(builtinValue); ok {
		name = builtin.Name
	}
	typeArgs := append([]string{}, call.TypeArgs...)
	if name == "json.decode" {
		if len(typeArgs) != 1 {
			return nil, fmt.Errorf("json.decode requires one explicit schema type: json.decode[T](text:body)")
		}
		qualified, err := ctx.rt.jsonSchemaType(typeArgs[0], ctx.module, map[string]bool{})
		if err != nil {
			return nil, err
		}
		typeArgs[0] = qualified
	}
	sig, err = specializeSignature(sig, name, typeArgs)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(call.Args))
	names := make([]string, len(sig.Params))
	byName := map[string]ParamSpec{}
	for i, param := range sig.Params {
		names[i] = param.Name
		byName[param.Name] = param
	}
	for i, arg := range call.Args {
		keys[i] = arg.Name
	}
	var order []int
	if partial {
		if len(keys) == 0 {
			return nil, fmt.Errorf("partial must bind at least one argument")
		}
		for _, name := range keys {
			if _, ok := byName[name]; !ok {
				return nil, fmt.Errorf("unknown argument %q in partial application", name)
			}
		}
		if _, err = ArgumentOrder(keys, keys); err != nil {
			return nil, err
		}
	} else {
		order, err = ArgumentOrder(names, keys)
		if err != nil {
			return nil, err
		}
		if err = effectsAllowed(sig.Effects, ctx.effects); err != nil {
			return nil, err
		}
	}
	for _, arg := range call.Args {
		if _, defaulted := arg.Value.(*defaultExpr); defaulted && byName[arg.Name].Default == "" {
			return nil, fmt.Errorf("argument %s has no declared default", arg.Name)
		}
	}
	values := make([]value, len(call.Args))
	bound := map[string]value{}
	for i, arg := range call.Args {
		if _, defaulted := arg.Value.(*defaultExpr); defaulted {
			values[i], err = literalValue(byName[arg.Name].Default)
		} else if partial {
			pure := *ctx
			pure.effects = nil
			values[i], err = pure.eval(arg.Value)
		} else {
			values[i], err = ctx.eval(arg.Value)
		}
		if err != nil {
			return nil, err
		}
		if err = ctx.rt.validateValueIn(values[i], byName[arg.Name].Type, sig.Module); err != nil {
			return nil, fmt.Errorf("argument %s: %w", arg.Name, err)
		}
		if partial {
			bound[arg.Name], err = freezeValue(values[i])
			if err != nil {
				return nil, err
			}
		}
	}
	if partial {
		target, e := freezeValue(callee)
		if e != nil {
			return nil, e
		}
		remaining := FunctionType{ReturnType: sig.ReturnType, Effects: append([]string{}, sig.Effects...), Module: sig.Module}
		for _, param := range sig.Params {
			if _, bound := bound[param.Name]; !bound {
				remaining.Params = append(remaining.Params, param)
			}
		}
		return &partialValue{Target: target, Bound: bound, Signature: remaining, TypeArgs: typeArgs}, nil
	}
	args := make([]value, len(order))
	for i, index := range order {
		args[i] = values[index]
	}
	// json.decode carries a source-level type argument, which is not a value argument.
	if fn, ok := callee.(builtinValue); ok && len(typeArgs) > 0 {
		result, e := fn.Invoke(ctx, typeArgs, args)
		if e != nil {
			return nil, e
		}
		if e = ctx.rt.validateValueIn(result, sig.ReturnType, sig.Module); e != nil {
			return nil, e
		}
		return result, nil
	}
	return ctx.rt.invokeCallable(callee, args)
}

func freezeValue(v value) (value, error) {
	switch x := v.(type) {
	case nil, string, int, float64, bool, functionValue, builtinValue:
		return v, nil
	case *structValue:
		fields := map[string]any{}
		for key, val := range x.Fields {
			copy, err := freezeValue(val)
			if err != nil {
				return nil, err
			}
			fields[key] = copy
		}
		return &structValue{Type: x.Type, Module: x.Module, Fields: fields}, nil
	case map[string]any:
		result := map[string]any{}
		for key, val := range x {
			copy, err := freezeValue(val)
			if err != nil {
				return nil, err
			}
			result[key] = copy
		}
		return result, nil
	case []value:
		result := make([]value, len(x))
		for i, val := range x {
			copy, err := freezeValue(val)
			if err != nil {
				return nil, err
			}
			result[i] = copy
		}
		return result, nil
	case resultValue:
		a, err := freezeValue(x.Value)
		if err != nil {
			return nil, err
		}
		b, err := freezeValue(x.Err)
		return resultValue{Kind: x.Kind, Value: a, Err: b}, err
	case methodValue:
		receiver, err := freezeValue(x.Receiver)
		return methodValue{Receiver: receiver, Method: x.Method}, err
	case *contractedValue:
		target, err := freezeValue(x.Target)
		if err != nil {
			return nil, err
		}
		return &contractedValue{Target: target, Signature: x.Signature}, nil
	case *transformValue:
		steps := make([]transformStep, len(x.Steps))
		for i, step := range x.Steps {
			callback, err := freezeValue(step.Callback.Callee)
			if err != nil {
				return nil, err
			}
			steps[i] = step
			steps[i].Callback.Callee = callback
		}
		return &transformValue{InputType: x.InputType, OutputType: x.OutputType, Steps: steps}, nil
	case *partialValue:
		target, err := freezeValue(x.Target)
		if err != nil {
			return nil, err
		}
		bound := map[string]value{}
		for key, val := range x.Bound {
			copy, err := freezeValue(val)
			if err != nil {
				return nil, err
			}
			bound[key] = copy
		}
		return &partialValue{Target: target, Bound: bound, Signature: x.Signature, TypeArgs: append([]string{}, x.TypeArgs...)}, nil
	}
	return nil, fmt.Errorf("cannot capture mutable/unsupported runtime value %T", v)
}

func listOperation(name string) builtinFn {
	return func(ctx *execContext, _ []string, args []value) (value, error) {
		items, ok := args[0].([]value)
		if !ok {
			return nil, fmt.Errorf("list.%s expects a list", name)
		}
		callback := args[1]
		if name == "reduce" {
			callback = args[2]
		}
		prepared, err := prepareListCallback(ctx.rt, callback, name == "reduce", name == "filter")
		if err != nil {
			return nil, fmt.Errorf("list.%s: %w", name, err)
		}
		var state value
		if name == "reduce" {
			state = args[1]
			if err := prepared.validateInitial(ctx.rt, state); err != nil {
				return nil, err
			}
		}
		out := []value{}
		if name == "map" {
			out = make([]value, 0, len(items))
		}
		for _, item := range items {
			result, err := prepared.invoke(ctx.rt, state, item)
			if err != nil {
				return nil, err
			}
			switch name {
			case "map":
				out = append(out, result)
			case "filter":
				if result.(bool) {
					out = append(out, item)
				}
			case "reduce":
				state = result
			}
		}
		if name == "reduce" {
			return state, nil
		}
		return out, nil
	}
}
