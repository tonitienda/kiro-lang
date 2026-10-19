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

func ascribeCallable(v value, typ string) value {
	if !strings.HasPrefix(typ, "Fn[") {
		return v
	}
	if signature, err := ParseFunctionType(typ); err == nil {
		if existing, ok := v.(*contractedValue); ok && existing.Signature.String() == signature.String() {
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
		return signatureOf(f, false), nil
	case methodValue:
		return signatureOf(fn.Method, true), nil
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
		if err := rt.validateValue(args[i], param.Type); err != nil {
			return nil, fmt.Errorf("argument %s: %w", param.Name, err)
		}
	}
	var result value
	switch fn := callee.(type) {
	case builtinValue:
		ctx := &execContext{rt: rt, locals: map[string]value{}, effects: sig.Effects}
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
	if err = rt.validateValue(result, sig.ReturnType); err != nil {
		return nil, fmt.Errorf("return value: %w", err)
	}
	return ascribeCallable(result, sig.ReturnType), nil
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
	sig, err = specializeSignature(sig, name, call.TypeArgs)
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
		if err = ctx.rt.validateValue(values[i], byName[arg.Name].Type); err != nil {
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
		remaining := FunctionType{ReturnType: sig.ReturnType, Effects: append([]string{}, sig.Effects...)}
		for _, param := range sig.Params {
			if _, bound := bound[param.Name]; !bound {
				remaining.Params = append(remaining.Params, param)
			}
		}
		return &partialValue{Target: target, Bound: bound, Signature: remaining, TypeArgs: append([]string{}, call.TypeArgs...)}, nil
	}
	args := make([]value, len(order))
	for i, index := range order {
		args[i] = values[index]
	}
	// json.decode carries a source-level type argument, which is not a value argument.
	if fn, ok := callee.(builtinValue); ok && len(call.TypeArgs) > 0 {
		result, e := fn.Invoke(ctx, call.TypeArgs, args)
		if e != nil {
			return nil, e
		}
		if e = ctx.rt.validateValue(result, sig.ReturnType); e != nil {
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
		return &structValue{Type: x.Type, Fields: fields}, nil
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
		return resultValue{Value: a, Err: b}, err
	case methodValue:
		receiver, err := freezeValue(x.Receiver)
		return methodValue{Receiver: receiver, Method: x.Method}, err
	case *contractedValue:
		target, err := freezeValue(x.Target)
		if err != nil {
			return nil, err
		}
		return &contractedValue{Target: target, Signature: x.Signature}, nil
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
		if name == "fold" {
			callback = args[2]
		}
		sig, err := ctx.rt.callableType(callback)
		if err != nil {
			return nil, err
		}
		if len(sig.Effects) > 0 {
			return nil, fmt.Errorf("list.%s requires a pure callback", name)
		}
		keys := []string{"value"}
		if name == "fold" {
			keys = []string{"state", "value"}
		}
		params := make([]string, len(sig.Params))
		for i, param := range sig.Params {
			params[i] = param.Name
		}
		order, err := ArgumentOrder(params, keys)
		if err != nil {
			return nil, fmt.Errorf("list.%s callback: %w", name, err)
		}
		if name == "filter" && sig.ReturnType != "bool" {
			return nil, fmt.Errorf("list.filter callback must return bool")
		}
		out := []value{}
		var state value
		if name == "fold" {
			state = args[1]
			for _, parameter := range sig.Params {
				if parameter.Name == "state" {
					if parameter.Type != sig.ReturnType {
						return nil, fmt.Errorf("list.fold reducer return type must match state type")
					}
					if err := ctx.rt.validateValue(state, parameter.Type); err != nil {
						return nil, fmt.Errorf("list.fold initial: %w", err)
					}
				}
			}
		}
		for _, item := range items {
			values := []value{item}
			if name == "fold" {
				values = []value{state, item}
			}
			bound := make([]value, len(order))
			for i, index := range order {
				bound[i] = values[index]
			}
			result, err := ctx.rt.invokeCallable(callback, bound)
			if err != nil {
				return nil, err
			}
			switch name {
			case "map":
				out = append(out, result)
			case "filter":
				keep, ok := result.(bool)
				if !ok {
					return nil, fmt.Errorf("filter callback returned a non-bool")
				}
				if keep {
					out = append(out, item)
				}
			case "fold":
				state = result
			}
		}
		if name == "fold" {
			return state, nil
		}
		return out, nil
	}
}
