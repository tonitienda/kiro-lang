package runtimekit

import "fmt"

func (c *functionalChecker) transformCall(name string, values map[string]inferredValue) (inferredValue, bool, error) {
	unknown := inferredValue{Type: "Any"}
	switch name {
	case "list.mapping", "list.filtering", "list.compose", "list.transduce", "list.collect":
	default:
		return unknown, false, nil
	}

	fail := func(err error) (inferredValue, bool, error) { return unknown, true, fmt.Errorf("%s: %w", name, err) }
	success := func(typ string) (inferredValue, bool, error) { return inferredValue{Type: typ}, true, nil }
	switch name {
	case "list.mapping", "list.filtering":
		key := "transform"
		if name == "list.filtering" {
			key = "predicate"
		}
		callback := values[key]
		if callback.Type == "Any" {
			return success("Any")
		}
		if callback.Function == nil {
			return fail(fmt.Errorf("%s must be a typed callable", key))
		}
		sig := callback.Function
		if len(sig.Effects) > 0 {
			return fail(fmt.Errorf("requires pure callbacks"))
		}
		if len(sig.Params) != 1 || sig.Params[0].Name != "value" {
			return fail(fmt.Errorf("callback requires exactly the value key"))
		}
		output := sig.ReturnType
		if key == "predicate" {
			if output != "bool" {
				return fail(fmt.Errorf("filter predicate must return bool"))
			}
			output = sig.Params[0].Type
		}
		return success("Transform[" + sig.Params[0].Type + "," + output + "]")
	case "list.compose":
		first, second := values["first"].Type, values["then"].Type
		if first == "Any" || second == "Any" {
			return success("Any")
		}
		a, ok := genericParts(first, "Transform")
		b, ok2 := genericParts(second, "Transform")
		if !ok || !ok2 || len(a) != 2 || len(b) != 2 {
			return fail(fmt.Errorf("first and then must be Transform values"))
		}
		if !contractAcceptsType(b[0], a[1]) {
			return fail(fmt.Errorf("cannot pass %s output to %s input", a[1], b[0]))
		}
		return success("Transform[" + a[0] + "," + b[1] + "]")
	case "list.transduce", "list.collect":
		through := values["through"].Type
		if through == "Any" {
			if name == "list.collect" {
				return success("List[Any]")
			}
			return success("Any")
		}
		types, ok := genericParts(through, "Transform")
		if !ok || len(types) != 2 {
			return fail(fmt.Errorf("through must be a Transform"))
		}
		items := values["items"].Type
		if element, ok := genericParts(items, "List"); ok && len(element) == 1 {
			// An empty or otherwise unresolved list is checked item by item at runtime.
			if element[0] != "Any" && !contractAcceptsType(types[0], element[0]) {
				return fail(fmt.Errorf("input expects %s, got %s", types[0], element[0]))
			}
		}
		if name == "list.collect" {
			return success("List[" + types[1] + "]")
		}
		callback := values["reduce"]
		if callback.Type == "Any" {
			return success("Any")
		}
		if callback.Function == nil {
			return fail(fmt.Errorf("reduce must be a typed callable"))
		}
		sig := callback.Function
		if len(sig.Effects) > 0 {
			return fail(fmt.Errorf("requires pure callbacks"))
		}
		names := []string{}
		parameters := map[string]string{}
		for _, p := range sig.Params {
			names = append(names, p.Name)
			parameters[p.Name] = p.Type
		}
		if _, err := ArgumentOrder(names, []string{"state", "value"}); err != nil {
			return fail(err)
		}
		if parameters["state"] != sig.ReturnType {
			return fail(fmt.Errorf("reducer return type must match state type"))
		}
		if !contractAcceptsType(parameters["value"], types[1]) {
			return fail(fmt.Errorf("reducer cannot accept %s output", types[1]))
		}
		if !compatibleValueTypes(values["initial"].Type, parameters["state"]) {
			return fail(fmt.Errorf("initial must have state type %s", parameters["state"]))
		}
		return success(sig.ReturnType)
	}
	return unknown, true, nil
}
