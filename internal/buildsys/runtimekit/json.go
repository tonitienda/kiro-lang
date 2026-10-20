package runtimekit

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func decodeJSON(ctx *execContext, typeArgs []string, args []value) (value, error) {
	if len(typeArgs) != 1 {
		return nil, fmt.Errorf("json.decode requires one explicit schema type")
	}
	typ, err := ctx.rt.jsonSchemaType(typeArgs[0], ctx.module, map[string]bool{})
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(args[0].(string)) {
		return errResult("$: JSON text must be valid UTF-8"), nil
	}
	decoder := json.NewDecoder(strings.NewReader(args[0].(string)))
	decoder.UseNumber()
	raw, err := readJSON(decoder, "$", 0)
	if err == nil {
		if _, trailing := decoder.Token(); trailing != io.EOF {
			if trailing != nil {
				err = trailing
			} else {
				err = fmt.Errorf("$: trailing JSON value")
			}
		}
	}
	if err != nil {
		return errResult(err.Error()), nil
	}
	v, err := ctx.rt.decodeJSONValue(raw, typ, ctx.module, "$")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return okResult(v), nil
}

func readJSON(decoder *json.Decoder, path string, depth int) (value, error) {
	if depth > 128 {
		return nil, fmt.Errorf("%s: JSON nesting exceeds 128", path)
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	switch delimiter := token.(type) {
	case json.Delim:
		switch delimiter {
		case '{':
			object := map[string]any{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("%s: expected object key", path)
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("%s.%s: duplicate JSON key", path, key)
				}
				item, err := readJSON(decoder, path+"."+key, depth+1)
				if err != nil {
					return nil, err
				}
				object[key] = item
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim('}') {
				return nil, fmt.Errorf("%s: unterminated object", path)
			}
			return object, nil
		case '[':
			items := []value{}
			for decoder.More() {
				item, err := readJSON(decoder, fmt.Sprintf("%s[%d]", path, len(items)), depth+1)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim(']') {
				return nil, fmt.Errorf("%s: unterminated list", path)
			}
			return items, nil
		}
		return nil, fmt.Errorf("%s: unexpected delimiter", path)
	default:
		return token, nil
	}
}

func (rt *runtime) decodeJSONValue(raw value, typ, module, path string) (value, error) {
	failure := func() (value, error) { return nil, fmt.Errorf("%s: expected %s, got %T", path, typ, raw) }
	if typ == "Any" {
		return raw, nil
	}
	if strings.HasPrefix(typ, "?") {
		if raw == nil {
			return nil, nil
		}
		return rt.decodeJSONValue(raw, typ[1:], module, path)
	}
	if parts, ok := genericParts(typ, "List"); ok && len(parts) == 1 {
		items, ok := raw.([]value)
		if !ok {
			return failure()
		}
		out := make([]value, len(items))
		for i, item := range items {
			v, err := rt.decodeJSONValue(item, parts[0], module, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	if parts, ok := genericParts(typ, "R"); ok && len(parts) == 2 {
		object, ok := raw.(map[string]any)
		if !ok || len(object) != 1 {
			return failure()
		}
		if payload, ok := object["Ok"]; ok {
			v, err := rt.decodeJSONValue(payload, parts[0], module, path+".Ok")
			return okResult(v), err
		}
		if payload, ok := object["Err"]; ok {
			v, err := rt.decodeJSONValue(payload, parts[1], module, path+".Err")
			return errResult(v), err
		}
		return failure()
	}
	switch typ {
	case "nil":
		if raw == nil {
			return nil, nil
		}
		return failure()
	case "str":
		if v, ok := raw.(string); ok {
			return v, nil
		}
		return failure()
	case "bool":
		if v, ok := raw.(bool); ok {
			return v, nil
		}
		return failure()
	case "i32":
		number, ok := raw.(json.Number)
		if !ok {
			return failure()
		}
		v, err := strconv.ParseInt(string(number), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%s: expected i32 integer in [-2147483648,2147483647]", path)
		}
		return int(v), nil
	}
	schema, err := rt.recordSchema(typ, module)
	if err != nil {
		return nil, err
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return failure()
	}
	if len(schema.Type.Variants) > 0 {
		if len(object) != 1 {
			return nil, fmt.Errorf("%s: expected exactly one variant tag for %s", path, typ)
		}
		for _, tag := range schema.Type.Variants {
			raw, exists := object[tag.Name]
			if !exists {
				continue
			}
			payload, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s.%s: expected payload object", path, tag.Name)
			}
			keys := []string{}
			for key := range payload {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			if err := checkRecordKeys(TypeSpec{Name: tag.Name, Fields: tag.Fields}, keys, false); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			fields := map[string]any{}
			for _, f := range tag.Fields {
				item, exists := payload[f.Name]
				if !exists {
					fields[f.Name] = nil
					continue
				}
				v, err := rt.decodeJSONValue(item, f.Type, schema.Module, path+"."+tag.Name+"."+f.Name)
				if err != nil {
					return nil, err
				}
				fields[f.Name] = v
			}
			return &variantValue{Type: schema.Type.Name, Module: schema.Module, Tag: tag.Name, Fields: fields}, nil
		}
		return nil, fmt.Errorf("%s: unknown variant tag for %s", path, typ)
	}
	keys := []string{}
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if err := checkRecordKeys(schema.Type, keys, false); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	fields := map[string]any{}
	for _, field := range schema.Type.Fields {
		item, exists := object[field.Name]
		if !exists {
			fields[field.Name] = nil
			continue
		}
		v, err := rt.decodeJSONValue(item, field.Type, schema.Module, path+"."+field.Name)
		if err != nil {
			return nil, err
		}
		fields[field.Name] = v
	}
	return &structValue{Type: schema.Type.Name, Module: schema.Module, Fields: fields}, nil
}

func encodeJSON(ctx *execContext, _ []string, args []value) (value, error) {
	wire, err := jsonWire(args[0])
	if err != nil {
		return errResult(err.Error()), nil
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return okResult(string(encoded)), nil
}
func jsonWire(v value) (any, error) {
	switch x := v.(type) {
	case nil, string, int, float64, bool, json.Number:
		return x, nil
	case *variantValue:
		fields, err := jsonWire(x.Fields)
		return map[string]any{x.Tag: fields}, err
	case *structValue:
		return jsonWire(x.Fields)
	case resultValue:
		key := "Ok"
		payload := x.Value
		if x.Kind == resultErr {
			key = "Err"
			payload = x.Err
		} else if x.Kind != resultOK {
			return nil, fmt.Errorf("invalid result tag")
		}
		item, err := jsonWire(payload)
		return map[string]any{key: item}, err
	case []value:
		out := make([]any, len(x))
		for i, v := range x {
			item, err := jsonWire(v)
			if err != nil {
				return nil, err
			}
			out[i] = item
		}
		return out, nil
	case map[string]any:
		out := map[string]any{}
		keys := []string{}
		for key := range x {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			item, err := jsonWire(x[key])
			if err != nil {
				return nil, err
			}
			out[key] = item
		}
		return out, nil
	}
	return nil, fmt.Errorf("%T has no JSON representation", v)
}
