package runtimekit

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func schemaRuntime(t *testing.T) *runtime {
	t.Helper()
	spec := resultSpec("")
	spec.Modules[0].Types = append(spec.Modules[0].Types,
		TypeSpec{Name: "Child", Fields: []FieldSpec{{Name: "quantity", Type: "i32"}}},
		TypeSpec{Name: "Input", Fields: []FieldSpec{{Name: "child", Type: "Child"}, {Name: "labels", Type: "List[str]"}, {Name: "note", Type: "?str"}}},
	)
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestStrictJSONSchemas(t *testing.T) {
	rt := schemaRuntime(t)
	ctx := &execContext{rt: rt, module: "main"}
	for _, tc := range []struct{ name, typ, text, failure string }{
		{"nested_optional", "Input", `{"child":{"quantity":2},"labels":["a","b"]}`, ""},
		{"null_optional", "Input", `{"child":{"quantity":2},"labels":[],"note":null}`, ""},
		{"missing", "Input", `{"labels":[]}`, `missing required field "child"`},
		{"nested_missing", "Input", `{"child":{},"labels":[]}`, `$.child: Child: missing required field "quantity"`},
		{"unknown", "Input", `{"child":{"quantity":2},"labels":[],"extra":1}`, `unknown field "extra"`},
		{"spoofed_tag", "Child", `{"quantity":2,"_type":"Child"}`, `unknown field "_type"`},
		{"wrong_scalar", "Child", `{"quantity":"2"}`, "$.quantity: expected i32"},
		{"wrong_item", "Input", `{"child":{"quantity":2},"labels":[1]}`, "$.labels[0]: expected str"},
		{"required_null", "Child", `{"quantity":null}`, "expected i32"},
		{"fraction", "i32", `1.5`, "expected i32 integer"},
		{"decimal_integer", "i32", `1.0`, "expected i32 integer"},
		{"exponent", "i32", `1e0`, "expected i32 integer"},
		{"overflow", "i32", `2147483648`, "expected i32 integer"},
		{"large_exact", "i32", `9007199254740993`, "expected i32 integer"},
		{"maximum", "i32", `2147483647`, ""}, {"minimum", "i32", `-2147483648`, ""},
		{"string", "str", `"text"`, ""}, {"boolean", "bool", `true`, ""},
		{"duplicate", "Child", `{"quantity":1,"quantity":2}`, "$.quantity: duplicate JSON key"},
		{"nested_duplicate", "Input", `{"child":{"quantity":1,"quantity":2},"labels":[]}`, "$.child.quantity: duplicate JSON key"},
		{"trailing", "Child", `{"quantity":1} {}`, "trailing JSON value"},
		{"malformed", "Child", `{"quantity":`, "EOF"},
		{"wrong_root", "Child", `[]`, "expected main.Child"},
		{"null_root", "Child", `null`, "expected main.Child"},
		{"result_nil", "R[i32,nil]", `{"Err":null}`, ""},
		{"result_success", "R[i32,str]", `{"Ok":2}`, ""},
		{"both_tags", "R[i32,str]", `{"Ok":2,"Err":"bad"}`, "expected R"},
		{"wrong_tag", "R[i32,str]", `{"Other":2}`, "expected R"},
		{"wrong_error", "R[i32,str]", `{"Err":2}`, "$.Err: expected str"},
		{"list_records", "List[Child]", `[{"quantity":1},{"quantity":2}]`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := decodeJSON(ctx, []string{tc.typ}, []value{tc.text})
			if err != nil {
				t.Fatal(err)
			}
			result := v.(resultValue)
			if tc.failure == "" {
				if result.Kind != resultOK {
					t.Fatalf("unexpected Err: %v", result.Err)
				}
				if err := rt.validateValueIn(result.Value, tc.typ, "main"); err != nil {
					t.Fatal(err)
				}
				if tc.name == "nested_optional" {
					if item := result.Value.(*structValue).Fields["note"]; item != nil {
						t.Fatalf("optional: %v", item)
					}
				}
			} else if result.Kind != resultErr || !strings.Contains(fmt.Sprint(result.Err), tc.failure) {
				t.Fatalf("want %q, got %v", tc.failure, toPlain(result))
			}
		})
	}
}

func TestJSONRoundTripKeepsRuntimeTags(t *testing.T) {
	rt := schemaRuntime(t)
	ctx := &execContext{rt: rt, module: "main"}
	for _, original := range []resultValue{okResult(nil), errResult(nil)} {
		wire, err := encodeJSON(ctx, nil, []value{original})
		if err != nil {
			t.Fatal(err)
		}
		encoded := wire.(resultValue)
		decoded, err := decodeJSON(ctx, []string{"R[nil,nil]"}, []value{encoded.Value})
		if err != nil {
			t.Fatal(err)
		}
		if decoded.(resultValue).Value.(resultValue).Kind != original.Kind {
			t.Fatal("result tag lost during JSON round trip")
		}
	}
	original := &structValue{Type: "Child", Module: "main", Fields: map[string]any{"quantity": 2}}
	wire, err := encodeJSON(ctx, nil, []value{original})
	if err != nil {
		t.Fatal(err)
	}
	encoded := wire.(resultValue).Value.(string)
	if encoded != `{"quantity":2}` {
		t.Fatalf("internal metadata escaped onto wire: %s", encoded)
	}
	decoded, err := decodeJSON(ctx, []string{"Child"}, []value{encoded})
	if err != nil {
		t.Fatal(err)
	}
	if !valuesEqual(original, decoded.(resultValue).Value) {
		t.Fatal("record round trip changed value")
	}
}

func TestRuntimeRecordContracts(t *testing.T) {
	rt := schemaRuntime(t)
	var effects bytes.Buffer
	rt.stdout = &effects
	for _, tc := range []struct{ name, body, failure string }{
		{"missing", `Box{}`, "missing required field"},
		{"unknown", `Box{value:1 other:2}`, "unknown field"},
		{"duplicate", `Box{value:1 value:2}`, "duplicate field"},
		{"wrong", `Box{value:"bad"}`, "expected i32"},
		{"shape_before_effects", `Box{other:println(value:"must not run")}`, "unknown field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			effects.Reset()
			p, err := newBodyParser(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			ex, err := p.parseExpr(0)
			if err != nil {
				t.Fatal(err)
			}
			_, err = (&execContext{rt: rt, module: "main", effects: []string{"io"}}).eval(ex)
			if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("want %s, got %v", tc.failure, err)
			}
			if effects.Len() != 0 {
				t.Fatalf("invalid record shape produced effects: %s", effects.String())
			}
		})
	}
	// Plain JSON objects cannot impersonate runtime records through an internal-looking key.
	if err := rt.validateValue(map[string]any{"_type": "Box", "value": 1}, "Box"); err == nil {
		t.Fatal("record forgery accepted")
	}
	if err := rt.validateValue(&structValue{Type: "Box", Module: "main", Fields: map[string]any{"value": "bad"}}, "Box"); err == nil {
		t.Fatal("invalid record accepted at function boundary")
	}
}

func FuzzTypedJSONRoundTrip(f *testing.F) {
	for _, seed := range []string{`{"quantity":0}`, `{"quantity":-2147483648}`, `{"quantity":2147483647}`, `{}`, `{"quantity":1,"quantity":2}`, `{"quantity":1.5}`, `null`, `[]`, `{"quantity":9007199254740993}`, string([]byte{0xff})} {
		f.Add(seed)
	}
	spec := ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Types: []TypeSpec{{Name: "Input", Fields: []FieldSpec{{Name: "quantity", Type: "i32"}}}}}}}
	f.Fuzz(func(t *testing.T, raw string) {
		rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &execContext{rt: rt, module: "main"}
		decoded, err := decodeJSON(ctx, []string{"Input"}, []value{raw})
		if err != nil {
			t.Fatalf("invalid data escaped the result channel: %v", err)
		}
		result := decoded.(resultValue)
		if result.Kind == resultErr {
			return
		}
		if result.Kind != resultOK {
			t.Fatal("invalid result discriminator")
		}
		if err := rt.validateValueIn(result.Value, "Input", "main"); err != nil {
			t.Fatal(err)
		}
		encoded, err := encodeJSON(ctx, nil, []value{result.Value})
		if err != nil {
			t.Fatal(err)
		}
		wire := encoded.(resultValue)
		if wire.Kind != resultOK {
			t.Fatal(wire.Err)
		}
		again, err := decodeJSON(ctx, []string{"Input"}, []value{wire.Value})
		if err != nil {
			t.Fatal(err)
		}
		if !valuesEqual(decoded, again) {
			t.Fatalf("round trip changed decoded value: %v, %v", toPlain(decoded), toPlain(again))
		}
	})
}
