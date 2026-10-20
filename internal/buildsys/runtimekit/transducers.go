package runtimekit

import (
	"fmt"
	"strings"
)

// Transform values are immutable plans. A plan contains no input collection and
// allocates no intermediate collection while executing. Steps run in written order.
type transformValue struct {
	InputType  string
	OutputType string
	Steps      []transformStep
}
type transformStep struct {
	Filter   bool
	Callback preparedCallback
}
type preparedCallback struct {
	Callee    value
	Signature FunctionType
	Order     []int
}

func prepareListCallback(rt *runtime, callee value, reducer, predicate bool) (preparedCallback, error) {
	sig, err := rt.callableType(callee)
	if err != nil {
		return preparedCallback{}, err
	}
	if len(sig.Effects) > 0 {
		return preparedCallback{}, fmt.Errorf("requires a pure callback")
	}
	keys := []string{"value"}
	if reducer {
		keys = []string{"state", "value"}
	}
	names := make([]string, len(sig.Params))
	for i, p := range sig.Params {
		names[i] = p.Name
	}
	order, err := ArgumentOrder(names, keys)
	if err != nil {
		return preparedCallback{}, err
	}
	if predicate && sig.ReturnType != "bool" {
		return preparedCallback{}, fmt.Errorf("filter callback must return bool")
	}
	if reducer {
		for _, p := range sig.Params {
			if p.Name == "state" && p.Type != sig.ReturnType {
				return preparedCallback{}, fmt.Errorf("reducer return type must match state type")
			}
		}
	}
	return preparedCallback{Callee: callee, Signature: sig, Order: order}, nil
}
func (p preparedCallback) parameter(name string) string {
	for _, param := range p.Signature.Params {
		if param.Name == name {
			return param.Type
		}
	}
	return "Any"
}
func (p preparedCallback) validateInitial(rt *runtime, initial value) error {
	if err := rt.validateValue(initial, p.parameter("state")); err != nil {
		return fmt.Errorf("reduce initial: %w", err)
	}
	return nil
}
func (p preparedCallback) invoke(rt *runtime, state, item value) (value, error) {
	var values [2]value
	values[0] = item
	if len(p.Order) == 2 {
		values[0] = state
		values[1] = item
	}
	var ordered [2]value
	for i, index := range p.Order {
		ordered[i] = values[index]
	}
	return rt.invokeCallable(p.Callee, ordered[:len(p.Order)])
}

// A known output must fit the receiving contract. Any accepts values of all types;
// an unknown output cannot promise a concrete type at a transform boundary.
func contractAcceptsType(accepted, provided string) bool {
	if accepted == "Any" || accepted == provided {
		return true
	}
	if provided == "Any" {
		return false
	}
	if strings.HasPrefix(accepted, "?") {
		if provided == "nil" {
			return true
		}
		return contractAcceptsType(accepted[1:], strings.TrimPrefix(provided, "?"))
	}
	if strings.HasPrefix(accepted, "Fn[") && strings.HasPrefix(provided, "Fn[") {
		expected, e1 := ParseFunctionType(accepted)
		actual, e2 := ParseFunctionType(provided)
		return e1 == nil && e2 == nil && compatibleFunctions(*actual, *expected) == nil
	}
	if a, ok := genericParts(accepted, "Transform"); ok && len(a) == 2 {
		if b, ok := genericParts(provided, "Transform"); ok && len(b) == 2 {
			return contractAcceptsType(b[0], a[0]) && contractAcceptsType(a[1], b[1])
		}
		return false
	}
	for _, generic := range []string{"List", "R"} {
		a, ok := genericParts(accepted, generic)
		b, ok2 := genericParts(provided, generic)
		if ok && ok2 && len(a) == len(b) {
			for i := range a {
				if !contractAcceptsType(a[i], b[i]) {
					return false
				}
			}
			return true
		}
	}
	// Preserve the existing nominal-name compatibility until module-distinct
	// record identity is implemented. No wildcard inference is used here.
	return strings.HasSuffix(accepted, "."+provided) || strings.HasSuffix(provided, "."+accepted)
}

func transformOperation(name string) builtinFn {
	return func(ctx *execContext, _ []string, args []value) (value, error) {
		switch name {
		case "mapping", "filtering":
			predicate := name == "filtering"
			callback, err := prepareListCallback(ctx.rt, args[0], false, predicate)
			if err != nil {
				return nil, fmt.Errorf("list.%s: %w", name, err)
			}
			// Snapshot a bound receiver or partial once, before the plan can be reused.
			callback.Callee, err = freezeValue(callback.Callee)
			if err != nil {
				return nil, err
			}
			input, output := callback.parameter("value"), callback.Signature.ReturnType
			if predicate {
				output = input
			}
			return &transformValue{InputType: input, OutputType: output, Steps: []transformStep{{Filter: predicate, Callback: callback}}}, nil
		case "compose":
			first, ok := args[0].(*transformValue)
			if !ok {
				return nil, fmt.Errorf("list.compose first must be a Transform")
			}
			then, ok := args[1].(*transformValue)
			if !ok {
				return nil, fmt.Errorf("list.compose then must be a Transform")
			}
			if !contractAcceptsType(then.InputType, first.OutputType) {
				return nil, fmt.Errorf("list.compose cannot pass %s output to %s input", first.OutputType, then.InputType)
			}
			steps := make([]transformStep, 0, len(first.Steps)+len(then.Steps))
			steps = append(steps, first.Steps...)
			steps = append(steps, then.Steps...)
			return &transformValue{InputType: first.InputType, OutputType: then.OutputType, Steps: steps}, nil
		case "transduce", "collect":
			items, ok := args[0].([]value)
			if !ok {
				return nil, fmt.Errorf("list.%s items must be a list", name)
			}
			through, ok := args[1].(*transformValue)
			if !ok {
				return nil, fmt.Errorf("list.%s through must be a Transform", name)
			}
			var reducer preparedCallback
			var state value
			if name == "transduce" {
				var err error
				reducer, err = prepareListCallback(ctx.rt, args[3], true, false)
				if err != nil {
					return nil, fmt.Errorf("list.transduce: %w", err)
				}
				if !contractAcceptsType(reducer.parameter("value"), through.OutputType) {
					return nil, fmt.Errorf("list.transduce reducer cannot accept %s output", through.OutputType)
				}
				state = args[2]
				if err := reducer.validateInitial(ctx.rt, state); err != nil {
					return nil, err
				}
			}
			// Only collect allocates an output collection. Transduce holds one current
			// element and accumulator, independent of the number of source elements.
			var output []value
			if name == "collect" {
				output = make([]value, 0, len(items))
			}
			for index, item := range items {
				if err := ctx.rt.validateValue(item, through.InputType); err != nil {
					return nil, fmt.Errorf("list.%s item %d: %w", name, index, err)
				}
				current, keep, err := through.apply(ctx.rt, item)
				if err != nil {
					return nil, fmt.Errorf("list.%s item %d: %w", name, index, err)
				}
				if !keep {
					continue
				}
				if err := ctx.rt.validateValue(current, through.OutputType); err != nil {
					return nil, err
				}
				if name == "collect" {
					output = append(output, current)
				} else {
					state, err = reducer.invoke(ctx.rt, state, current)
					if err != nil {
						return nil, fmt.Errorf("list.transduce item %d reducer: %w", index, err)
					}
				}
			}
			if name == "collect" {
				return output, nil
			}
			return state, nil
		}
		return nil, fmt.Errorf("unknown transform operation %s", name)
	}
}

func (t *transformValue) apply(rt *runtime, item value) (value, bool, error) {
	current := item
	for index, step := range t.Steps {
		result, err := step.Callback.invoke(rt, nil, current)
		if err != nil {
			return nil, false, fmt.Errorf("stage %d: %w", index, err)
		}
		if step.Filter {
			if !result.(bool) {
				return nil, false, nil
			}
		} else {
			current = result
		}
	}
	return current, true, nil
}
