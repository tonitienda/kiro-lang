package runtimekit

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func resultSpec(body string) ProgramSpec {
	spec := functionalSpec(body)
	spec.Modules[0].Types = append(spec.Modules[0].Types, TypeSpec{Name: "Failure", Fields: []FieldSpec{{Name: "reason", Type: "str"}}})
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs,
		FuncSpec{Module: "main", Name: "leaf", ReturnType: "R[i32,str]", BlockBody: true, Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: `when (value) {0 => {return Err(error:"domain failure")} _ => {return Ok(value:value)} }`},
		FuncSpec{Module: "main", Name: "parent", ReturnType: "R[i32,str]", BlockBody: true, Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: "let n = leaf(value:value)?\nreturn Ok(value:n + 1)"},
		FuncSpec{Module: "main", Name: "nil_error", ReturnType: "R[i32,nil]", BlockBody: true, Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: "return Err(error:nil)"},
		FuncSpec{Module: "main", Name: "stop", ReturnType: "R[i32,str]", BlockBody: true, Body: "let n = leaf(value:0)?\nreturn Ok(value:1 / 0)"},
		FuncSpec{Module: "main", Name: "typed_error", ReturnType: "R[i32,Failure]", BlockBody: true, Body: `return Err(error:Failure{reason:"failed"})`},
		FuncSpec{Module: "main", Name: "typed_parent", ReturnType: "R[str,Failure]", BlockBody: true, Body: "let n = typed_error()?\nreturn Ok(value:\"unreachable\")"},
		FuncSpec{Module: "main", Name: "parse_text", ReturnType: "R[i32,str]", BlockBody: true, Params: []ParamSpec{{Name: "value", Type: "str"}}, Body: "return parse.i32(text:value)"},
		FuncSpec{Module: "main", Name: "accumulate_results", ReturnType: "R[i32,str]", BlockBody: true, Params: []ParamSpec{{Name: "value", Type: "R[i32,str]"}, {Name: "state", Type: "R[i32,str]"}}, Body: "let a = state?\nlet b = value?\nreturn Ok(value:a + b)"},
		FuncSpec{Module: "main", Name: "spawn_parent", ReturnType: "R[i32,str]", BlockBody: true, Body: "let future = spawn leaf(value:0)\nlet result = await future\nlet n = result?\nreturn Ok(value:n)"},
		FuncSpec{Module: "main", Name: "assertion_failure", ReturnType: "R[nil,str]", BlockBody: true, Body: "test.eq(actual:1,expected:2)\nreturn Ok(value:nil)"},
		FuncSpec{Module: "main", Name: "mismatched_error", ReturnType: "R[i32,str]", BlockBody: true, Body: "let n = nil_error(value:0)?\nreturn Ok(value:n)"},
	)
	return spec
}

func TestCompositionalResults(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"spawn_await", `test.eq(actual:spawn_parent(),expected:Err(error:"domain failure"))`, ""},
		{"parent_failure", `test.eq(actual:parent(value:0),expected:Err(error:"domain failure"))`, ""},
		{"parent_success", `test.eq(actual:parent(value:2),expected:Ok(value:3))`, ""},
		{"short_circuit", `test.eq(actual:stop(),expected:Err(error:"domain failure"))`, ""},
		{"nil_payload", `test.eq(actual:nil_error(value:0),expected:Err(error:nil))
test.eq(actual:Err(error:nil) == Ok(value:nil),expected:false)`, ""},
		{"partial_nil", `let f = partial nil_error(value:0)
test.eq(actual:f(),expected:Err(error:nil))`, ""},
		{"typed_record_error", `test.eq(actual:typed_parent(),expected:Err(error:Failure{reason:"failed"}))`, ""},
		{"map_preserves_results", `test.eq(actual:list.map(items:[0,2],transform:parent),expected:[Err(error:"domain failure"),Ok(value:3)])`, ""},
		{"transduce_preserves_results", `let through = list.mapping(transform:parse_text)
let result = list.transduce(items:["1","bad","3"],through:through,initial:Ok(value:0),reduce:accumulate_results)
test.eq(actual:result,expected:parse.i32(text:"bad"))`, ""},
		{"partial_callback", `let source = partial parent(value:0)
test.eq(actual:source(),expected:Err(error:"domain failure"))`, ""},
		{"invalid_boundary", `leaf(value:1)?`, "? requires an R[T,E] return type"},
		{"mismatched_error_channel", `mismatched_error()`, "expected str"},
		{"assertions_not_domain_errors", `assertion_failure()`, "assertion failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			rt, err := newRuntime(resultSpec(tc.body), nil, &output, &output)
			if err != nil {
				t.Fatal(err)
			}
			code, err := rt.run()
			if tc.want == "" {
				if err != nil || code != 0 {
					t.Fatalf("code=%d, err=%v", code, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if output.Len() != 0 {
				t.Fatalf("unexpected effects: %s", output.String())
			}
		})
	}
}

func TestResultHostBoundaries(t *testing.T) {
	for _, mode := range []string{"build", "test"} {
		for _, kind := range []resultKind{resultOK, resultErr} {
			spec := ProgramSpec{Mode: mode, EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Funcs: []FuncSpec{{Module: "main", Name: "main", ReturnType: "R[i32,nil]", BlockBody: true, Body: "return Ok(value:0)"}}}}}
			fn := &spec.Modules[0].Funcs[0]
			if mode == "test" {
				fn.Name = "test_result"
			}
			if kind == resultErr {
				fn.Body = "return Err(error:nil)"
			}
			var output bytes.Buffer
			rt, err := newRuntime(spec, nil, &output, &output)
			if err != nil {
				t.Fatal(err)
			}
			code, err := rt.run()
			if kind == resultOK && (err != nil || code != 0) {
				t.Fatalf("success code=%d err=%v", code, err)
			}
			if kind == resultErr && code == 0 {
				t.Fatalf("Err(nil) passed %s: %v %s", mode, err, output.String())
			}
		}
	}
	rt, err := newRuntime(resultSpec(""), nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.validateValue(resultValue{}, "R[Any,Any]"); err == nil {
		t.Fatal("zero result tag accepted")
	}
	if err := rt.validateValue(errResult(nil), "R[i32,nil]"); err != nil {
		t.Fatal(err)
	}
	if valuesEqual(1, "1") {
		t.Fatal("typed equality coerced integer and string")
	}
}

func TestInterpolationDataIsNotExecuted(t *testing.T) {
	rt, err := newRuntime(resultSpec(""), nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &execContext{rt: rt, module: "main", locals: map[string]value{"payload": "${unknown_function()}"}}
	output, err := ctx.interpolate("payload=${payload}")
	if err != nil || output != "payload=${unknown_function()}" {
		t.Fatalf("interpolated data executed: %q %v", output, err)
	}
}

func TestDecoderSchemaSurvivesModuleBoundary(t *testing.T) {
	spec := resultSpec(`let decode = codec.decoder(text:"{\"value\":2}")
test.eq(actual:decode(),expected:Ok(value:codec.Payload{value:2}))`)
	spec.Modules = append(spec.Modules,
		ModuleSpec{Name: "codec", Types: []TypeSpec{{Name: "Payload", Fields: []FieldSpec{{Name: "value", Type: "i32"}}}}, Funcs: []FuncSpec{{Module: "codec", Name: "decoder", ReturnType: "Fn[()->R[Payload,str]]", BlockBody: true, Params: []ParamSpec{{Name: "text", Type: "str"}}, Body: "return partial json.decode[Payload](text:text)"}}},
		ModuleSpec{Name: "other", Types: []TypeSpec{{Name: "Payload", Fields: []FieldSpec{{Name: "value", Type: "str"}}}}},
	)
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if code, err := rt.run(); err != nil || code != 0 {
		t.Fatalf("code %d: %v", code, err)
	}
	if err := rt.validateValueIn(&structValue{Type: "Payload", Module: "other", Fields: map[string]any{"value": 2}}, "Payload", "codec"); err == nil {
		t.Fatal("record module identity lost")
	}
}
