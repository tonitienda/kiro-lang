package runtimekit

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func variantSpec(body string) ProgramSpec {
	return ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Types: []TypeSpec{
		{Name: "Failure", Variants: []VariantSpec{{Name: "Missing", Fields: []FieldSpec{{Name: "key", Type: "str"}}}, {Name: "Invalid", Fields: []FieldSpec{{Name: "amount", Type: "i32"}, {Name: "note", Type: "?str"}}}, {Name: "Offline"}}},
	}, Funcs: []FuncSpec{
		{Module: "main", Name: "main", ReturnType: "i32", Effects: []string{"io"}, BlockBody: true, Body: body + "\nreturn 0"},
		{Module: "main", Name: "fail", ReturnType: "R[i32,Failure]", BlockBody: true, Body: `return Err(error:Failure.Missing(key:"item"))`},
		{Module: "main", Name: "early", ReturnType: "R[i32,Failure]", BlockBody: true, Body: "for n in [1,2] {let value = fail()?\nreturn Ok(value:value)}\nreturn Ok(value:0)"},
		{Module: "main", Name: "first", ReturnType: "i32", BlockBody: true, Body: "for n in [3,4] {return n}\nreturn 0"},
	}}}}
}

func TestVariantAndLoopRuntime(t *testing.T) {
	for _, tc := range []struct{ name, body, failure string }{
		{"typed_result", `when (fail()) {Ok(value:_) => {test.eq(actual:1,expected:0)}
Err(error:let error) => {when (error) {Failure.Missing(key:let key) => {test.eq(actual:key,expected:"item")}
Failure.Invalid(amount:_,note:_) => {test.eq(actual:1,expected:0)}
Failure.Offline() => {test.eq(actual:1,expected:0)}}}}`, ""},
		{"constructor_order", `Failure.Invalid(note:println(value:"first"),amount:println(value:"second"))`, "expected i32"},
		{"keys_before_effects", `Failure.Missing(wrong:println(value:1))`, "unknown argument"},
		{"duplicate_keys", `Failure.Missing(key:"a",key:"b")`, "duplicate argument"},
		{"field_type", `Failure.Missing(key:1)`, "expected str"},
		{"unknown_tag", `Failure.Unknown()`, "not available"},
		{"sum_not_record", `Failure{}`, "requires a named constructor"},
		{"loop_mutation", `mut total = 0
for n in [1,2] {for m in [3,4] {total = total + n * m}}
test.eq(actual:total,expected:21)
test.eq(actual:first(),expected:3)
test.eq(actual:early(),expected:Err(error:Failure.Missing(key:"item")))`, ""},
		{"loop_nonlist", `for n in 1 {println(value:n)}`, "for requires a List"},
		{"loop_iteration_binding", `for n in [1] {n = 2}`, "immutable"},
		{"loop_local_scope", `for n in [1] {let local = n}
local`, "unknown identifier"},
		{"return_in_expression", `let result = when (Failure.Offline()) {
Failure.Offline() => {return 0}
Failure.Missing(key:_) => {return 1}
Failure.Invalid(amount:_,note:_) => {return 1}
}
return 1`, ""},
		{"nested_expression_return", `let n = 2 + when (Ok(value:1)) {
Ok(value:let value) => {return 0}
Err(error:_) => {return 1}
}
return 1`, ""},
		{"shadow_isolation", `mut n = 8
for n in [1] {test.eq(actual:n,expected:1)}
test.eq(actual:n,expected:8)
when (Failure.Offline()) {Failure.Missing(key:_) => {return 1}
Failure.Invalid(amount:_,note:_) => {return 1}
Failure.Offline() => {mut n = 3
n = 4}}
test.eq(actual:n,expected:8)
for x in [1] {mut n = 5
n = 6}
test.eq(actual:n,expected:8)`, ""},
		{"literal_pattern_is_opaque", `when ("1") {"${1}" => {return 1}
_ => {return 0}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			rt, err := newRuntime(variantSpec(tc.body), nil, &output, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			code, runErr := rt.run()
			err = runErr
			if tc.failure == "" {
				if err != nil || code != 0 {
					t.Fatalf("code=%d err=%v", code, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("want %q, got %v", tc.failure, err)
			}
			if tc.name == "keys_before_effects" && output.Len() != 0 {
				t.Fatalf("invalid keys executed effects: %s", output.String())
			}
		})
	}
}

func TestVariantSchemaBoundaries(t *testing.T) {
	rt, err := newRuntime(variantSpec(""), nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		v    value
	}{
		{"wrong_owner", &variantValue{Type: "Failure", Module: "other", Tag: "Offline", Fields: map[string]any{}}},
		{"unknown_tag", &variantValue{Type: "Failure", Module: "main", Tag: "Unknown", Fields: map[string]any{}}},
		{"missing_payload", &variantValue{Type: "Failure", Module: "main", Tag: "Missing", Fields: map[string]any{}}},
		{"wrong_payload", &variantValue{Type: "Failure", Module: "main", Tag: "Missing", Fields: map[string]any{"key": 1}}},
		{"extra_payload", &variantValue{Type: "Failure", Module: "main", Tag: "Offline", Fields: map[string]any{"key": "a"}}},
		{"record_spoof", &structValue{Type: "Failure", Module: "main", Fields: map[string]any{}}},
		{"map_spoof", map[string]any{"_type": "Failure", "_variant": "Offline"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := rt.validateValueIn(tc.v, "Failure", "main"); err == nil {
				t.Fatal("accepted invalid variant")
			}
		})
	}
	ctx := &execContext{rt: rt, module: "main"}
	for _, tc := range []struct {
		wire    string
		success bool
	}{
		{`{"Missing":{"key":"a"}}`, true},
		{`{"Invalid":{"amount":0}}`, true},
		{`{"Offline":{}}`, true},
		{`{"Missing":{}}`, false},
		{`{"Missing":{"key":1}}`, false},
		{`{"Missing":{"key":"a","other":1}}`, false},
		{`{"Missing":{"key":"a","key":"b"}}`, false},
		{`{"Invalid":{"amount":2147483648}}`, false},
		{`{"Invalid":{"amount":1e0}}`, false},
		{`{"Offline":null}`, false},
		{`{"Unknown":{}}`, false},
		{`{"Offline":{},"Missing":{"key":"a"}}`, false},
		{`{}`, false},
	} {
		t.Run(tc.wire, func(t *testing.T) {
			v, err := decodeJSON(ctx, []string{"Failure"}, []value{tc.wire})
			if err != nil {
				t.Fatal(err)
			}
			result := v.(resultValue)
			if (result.Kind == resultOK) != tc.success {
				t.Fatalf("result: %v", toPlain(result))
			}
			if !tc.success {
				return
			}
			if err := rt.validateValueIn(result.Value, "Failure", "main"); err != nil {
				t.Fatal(err)
			}
			wire, err := encodeJSON(ctx, nil, []value{result.Value})
			if err != nil {
				t.Fatal(err)
			}
			encoded := wire.(resultValue)
			if encoded.Kind != resultOK {
				t.Fatal(encoded)
			}
			again, err := decodeJSON(ctx, []string{"Failure"}, []value{encoded.Value})
			if err != nil || !valuesEqual(v, again) {
				t.Fatalf("round trip: %v %v", again, err)
			}
		})
	}
}

func TestVariantCaptureRetainsIdentity(t *testing.T) {
	original := &variantValue{Type: "Failure", Module: "main", Tag: "Missing", Fields: map[string]any{"key": "before"}}
	captured, err := freezeValue(original)
	if err != nil {
		t.Fatal(err)
	}
	original.Fields["key"] = "after"
	if valuesEqual(original, captured) {
		t.Fatal("capture was not isolated")
	}
	other := *captured.(*variantValue)
	other.Module = "other"
	if valuesEqual(captured, &other) {
		t.Fatal("equality lost module identity")
	}
}

func FuzzVariantJSONRoundTrip(f *testing.F) {
	for _, wire := range []string{`{"Leaf":{"value":0}}`, `{"Branch":{"children":[{"Leaf":{"value":3}},{"Branch":{"children":[]}}]}}`, `{"Unknown":{}}`, `{"Leaf":{"value":1e0}}`, `{"Branch":{"children":null}}`} {
		f.Add(wire)
	}
	spec := variantSpec("")
	spec.Modules[0].Types = append(spec.Modules[0].Types, TypeSpec{Name: "Tree", Variants: []VariantSpec{
		{Name: "Leaf", Fields: []FieldSpec{{Name: "value", Type: "i32"}}},
		{Name: "Branch", Fields: []FieldSpec{{Name: "children", Type: "List[Tree]"}}},
	}})
	f.Fuzz(func(t *testing.T, wire string) {
		if len(wire) > 32768 {
			t.Skip()
		}
		rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &execContext{rt: rt, module: "main"}
		v, err := decodeJSON(ctx, []string{"Tree"}, []value{wire})
		if err != nil {
			t.Fatal(err)
		}
		result := v.(resultValue)
		if result.Kind == resultErr {
			return
		}
		if err := rt.validateValueIn(result.Value, "Tree", "main"); err != nil {
			t.Fatal(err)
		}
		encoded, err := encodeJSON(ctx, nil, []value{result.Value})
		if err != nil {
			t.Fatal(err)
		}
		output := encoded.(resultValue)
		if output.Kind != resultOK {
			t.Fatal(output)
		}
		decoded, err := decodeJSON(ctx, []string{"Tree"}, []value{output.Value})
		if err != nil || !valuesEqual(v, decoded) {
			t.Fatalf("round trip failed: %v", err)
		}
	})
}
