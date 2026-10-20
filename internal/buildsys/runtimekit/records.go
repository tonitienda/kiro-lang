package runtimekit

import (
	"fmt"
	"sort"
	"strings"
)

type recordSchema struct {
	Module string
	Type   TypeSpec
}

func (s recordSchema) fieldType(name string) string {
	for _, f := range s.Type.Fields {
		if f.Name == name {
			return f.Type
		}
	}
	return ""
}

var httpRecords = map[string]TypeSpec{
	"Req":  {Name: "Req", Fields: []FieldSpec{{Name: "method", Type: "str"}, {Name: "path", Type: "str"}, {Name: "body", Type: "str"}, {Name: "ctx", Type: "?ctx.Ctx"}}},
	"Resp": {Name: "Resp", Fields: []FieldSpec{{Name: "code", Type: "i32"}, {Name: "body", Type: "str"}, {Name: "headers", Type: "?Any"}}},
}

func (rt *runtime) recordSchema(name, module string) (recordSchema, error) {
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		module = name[:dot]
		name = name[dot+1:]
		if module == "http" {
			if typ, ok := httpRecords[name]; ok {
				return recordSchema{Module: module, Type: typ}, nil
			}
		}
		if mod := rt.modules[module]; mod != nil {
			if typ, ok := mod.types[name]; ok {
				return recordSchema{Module: module, Type: typ}, nil
			}
		}
		return recordSchema{}, fmt.Errorf("unknown record type %s.%s", module, name)
	}
	if mod := rt.modules[module]; mod != nil {
		if typ, ok := mod.types[name]; ok {
			return recordSchema{Module: module, Type: typ}, nil
		}
	}
	candidates := []string{}
	for key, mod := range rt.modules {
		if _, ok := mod.types[name]; ok {
			candidates = append(candidates, key)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 1 {
		owner := candidates[0]
		return recordSchema{Module: owner, Type: rt.modules[owner].types[name]}, nil
	}
	if len(candidates) > 1 {
		return recordSchema{}, fmt.Errorf("ambiguous record type %s; qualify its module (%s)", name, strings.Join(candidates, ", "))
	}
	if typ, ok := httpRecords[name]; ok {
		return recordSchema{Module: "http", Type: typ}, nil
	}
	return recordSchema{}, fmt.Errorf("unknown record type %s", name)
}

func checkRecordKeys(typ TypeSpec, keys []string, all bool) error {
	expected := map[string]string{}
	for _, field := range typ.Fields {
		expected[field.Name] = field.Type
	}
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			return fmt.Errorf("%s: duplicate field %q", typ.Name, key)
		}
		seen[key] = true
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("%s: unknown field %q", typ.Name, key)
		}
	}
	for _, field := range typ.Fields {
		if !seen[field.Name] && (all || !strings.HasPrefix(field.Type, "?")) {
			return fmt.Errorf("%s: missing required field %q", typ.Name, field.Name)
		}
	}
	return nil
}

func (rt *runtime) validateRecord(v value, typ, module string) error {
	record, ok := v.(*structValue)
	if !ok {
		return fmt.Errorf("expected %s record, got %T", typ, v)
	}
	if module == "" && record.Module != "" {
		module = record.Module
	}
	schema, err := rt.recordSchema(typ, module)
	if err != nil {
		return err
	}
	if record.Type != schema.Type.Name || (record.Module != "" && record.Module != schema.Module) {
		return fmt.Errorf("expected %s.%s, got %s.%s", schema.Module, schema.Type.Name, record.Module, record.Type)
	}
	keys := make([]string, 0, len(record.Fields))
	for key := range record.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if err := checkRecordKeys(schema.Type, keys, false); err != nil {
		return err
	}
	for _, field := range schema.Type.Fields {
		item, exists := record.Fields[field.Name]
		if !exists {
			continue
		}
		if err := rt.validateValueIn(item, field.Type, schema.Module); err != nil {
			return fmt.Errorf("%s.%s: %w", schema.Type.Name, field.Name, err)
		}
	}
	return nil
}

// Resolve a JSON schema in its declaration context before a partial/alias can
// move it to another module. Recursive record schemas are allowed.
func (rt *runtime) jsonSchemaType(typ, module string, seen map[string]bool) (string, error) {
	if strings.HasPrefix(typ, "?") {
		inner, err := rt.jsonSchemaType(typ[1:], module, seen)
		return "?" + inner, err
	}
	switch typ {
	case "str", "bool", "nil", "i32", "Any":
		return typ, nil
	}
	for _, generic := range []string{"List", "R"} {
		if parts, ok := genericParts(typ, generic); ok {
			want := 1
			if generic == "R" {
				want = 2
			}
			if len(parts) != want {
				return "", fmt.Errorf("%s requires %d type arguments", generic, want)
			}
			for i, part := range parts {
				resolved, err := rt.jsonSchemaType(part, module, seen)
				if err != nil {
					return "", err
				}
				parts[i] = resolved
			}
			return generic + "[" + strings.Join(parts, ",") + "]", nil
		}
	}
	if typ == "ctx.Ctx" || strings.HasPrefix(typ, "Fn[") || strings.HasPrefix(typ, "Transform[") {
		return "", fmt.Errorf("%s is not a JSON schema type", typ)
	}
	schema, err := rt.recordSchema(typ, module)
	if err != nil {
		return "", err
	}
	full := schema.Module + "." + schema.Type.Name
	if seen[full] {
		return full, nil
	}
	seen[full] = true
	for _, field := range schema.Type.Fields {
		if _, err := rt.jsonSchemaType(field.Type, schema.Module, seen); err != nil {
			return "", fmt.Errorf("%s.%s: %w", full, field.Name, err)
		}
	}
	return full, nil
}

func (rt *runtime) qualifyType(typ, module string) string {
	if strings.HasPrefix(typ, "?") {
		return "?" + rt.qualifyType(typ[1:], module)
	}
	switch typ {
	case "Any", "i32", "str", "bool", "nil", "ctx.Ctx":
		return typ
	}
	if strings.HasPrefix(typ, "Fn[") {
		if fn, err := ParseFunctionType(typ); err == nil {
			return rt.qualifySignature(*fn, module).String()
		}
		return typ
	}
	for _, generic := range []string{"List", "R", "Transform"} {
		if parts, ok := genericParts(typ, generic); ok {
			for i, part := range parts {
				parts[i] = rt.qualifyType(part, module)
			}
			return generic + "[" + strings.Join(parts, ",") + "]"
		}
	}
	if schema, err := rt.recordSchema(typ, module); err == nil {
		return schema.Module + "." + schema.Type.Name
	}
	return typ
}
func (rt *runtime) qualifySignature(sig FunctionType, module string) FunctionType {
	sig.Module = module
	sig.Params = append([]ParamSpec{}, sig.Params...)
	for i, p := range sig.Params {
		sig.Params[i].Type = rt.qualifyType(p.Type, module)
	}
	sig.ReturnType = rt.qualifyType(sig.ReturnType, module)
	return sig
}
func (rt *runtime) functionType(fn FuncSpec, method bool) FunctionType {
	return rt.qualifySignature(signatureOf(fn, method), fn.Module)
}
