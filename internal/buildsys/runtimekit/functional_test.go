package runtimekit

import (
	"bytes"
	"strings"
	"testing"
)

func functionalSpec(body string) ProgramSpec {
	params := func(p ...ParamSpec) []ParamSpec { return p }
	return ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Types: []TypeSpec{{Name: "Box", Fields: []FieldSpec{{Name: "value", Type: "i32"}}}}, Funcs: []FuncSpec{
		{Module: "main", Name: "main", ReturnType: "i32", Effects: []string{"io"}, BlockBody: true, Body: body + "\nreturn 0"},
		{Module: "main", Name: "add", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}, ParamSpec{Name: "amount", Type: "i32", Default: "1"}), Body: "return value + amount"},
		{Module: "main", Name: "apply", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}, ParamSpec{Name: "transform", Type: "Fn[(value:i32)->i32]"}), Body: "return transform(value:value)"},
		{Module: "main", Name: "sum", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}, ParamSpec{Name: "state", Type: "i32"}), Body: "return state + value"},
		{Module: "main", Name: "keep", ReturnType: "bool", BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}, ParamSpec{Name: "minimum", Type: "i32"}), Body: "return value >= minimum"},
		{Module: "main", Name: "wrong", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}), Body: `return "oops"`},
		{Module: "main", Name: "label", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "item", Type: "i32"}), Body: "return item"},
		{Module: "main", Name: "bad_reduce", ReturnType: "str", BlockBody: true, Params: params(ParamSpec{Name: "state", Type: "i32"}, ParamSpec{Name: "value", Type: "i32"}), Body: `return "oops"`},
		{Module: "main", Name: "show", ReturnType: "nil", Effects: []string{"io"}, BlockBody: true, Params: params(ParamSpec{Name: "value", Type: "i32"}), Body: "println(value:value)\nreturn nil"},
		{Module: "main", Name: "pure", ReturnType: "nil", BlockBody: true, Body: "let output = println\noutput(value:123)\nreturn nil"},
		{Module: "main", Name: "box", ReturnType: "i32", BlockBody: true, Params: params(ParamSpec{Name: "box", Type: "Box"}, ParamSpec{Name: "extra", Type: "i32"}), Body: "return box.value + extra"},
	}}}}
}

func TestFunctionalRuntime(t *testing.T) {
	for _, tc := range []struct{ name, body, output, failure string }{
		{"explicit_default", `println(value:add(amount:default,value:4))`, "5\n", ""},
		{"negative", `println(value:add(value:-4,amount:default))`, "-3\n", ""},
		{"nested_partial", `let increase = partial add(amount:default)
let thunk = partial increase(value:4)
println(value:thunk())`, "5\n", ""},
		{"typed_argument", `println(value:apply(value:8,transform:partial add(amount:2)))`, "10\n", ""},
		{"snapshot", `mut number = 2
let increase = partial add(amount:number)
number = 100
println(value:increase(value:3))`, "5\n", ""},
		{"record_snapshot", `mut config = Box{value:2}
let quote = partial box(box:config)
config = Box{value:100}
println(value:quote(extra:3))`, "5\n", ""},
		{"eager_pipeline", `let up = partial add(amount:default)
let xs = list.map(items:[1,2,3],transform:up)
let ys = list.filter(items:xs,predicate:partial keep(minimum:3))
println(value:list.reduce(items:ys,initial:0,reduce:sum))`, "7\n", ""},
		{"empty_reduce", `println(value:list.reduce(items:[],initial:9,reduce:sum))`, "9\n", ""},
		{"latent_effect", `let output = partial show(value:3)
output()`, "3\n", ""},
		{"generic_partial", `let decode = partial json.decode[Box](text:"{}")
println(value:decode()?)`, "map[_type:Box]\n", ""},
		{"missing_default_key", `add(value:1)`, "", `missing argument "amount"`},
		{"no_default", `add(value:default,amount:1)`, "", "no declared default"},
		{"default_outside_call", `let number = default`, "", "only valid as an explicit call argument"},
		{"empty_partial", `partial add()`, "", "at least one"},
		{"unknown_partial", `partial add(other:1)`, "", "unknown argument"},
		{"duplicate_partial", `partial add(amount:1,amount:2)`, "", "duplicate argument"},
		{"wrong_capture", `partial add(amount:"bad")`, "", "expected i32"},
		{"effectful_capture", `partial add(amount:show(value:1))`, "", "effect !io"},
		{"capture_eager", `partial add(amount:wrong(value:1))`, "", "expected i32"},
		{"wrong_label", `apply(value:1,transform:label)`, "", "function parameter value:i32"},
		{"wrong_return", `apply(value:1,transform:wrong)`, "", "expected i32"},
		{"bad_callback_label_empty", `list.map(items:[],transform:label)`, "", "unknown argument"},
		{"bad_predicate_empty", `list.filter(items:[],predicate:partial add(amount:1))`, "", "return bool"},
		{"effectful_callback_empty", `list.map(items:[],transform:show)`, "", "pure callback"},
		{"bad_reduce_initial_empty", `list.reduce(items:[],initial:"oops",reduce:sum)`, "", "initial"},
		{"bad_reduce_return_empty", `list.reduce(items:[],initial:0,reduce:bad_reduce)`, "", "match state"},
		{"immutable_before_rhs", `let number = 2
number = show(value:1)`, "", "immutable binding"},
		{"alias_effect", `pure()`, "", "effect !io"},
		{"overflow", `add(value:2147483647,amount:1)`, "", "overflow"},
		{"wrong_arg", `add(value:"1",amount:1)`, "", "expected i32"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			rt, err := newRuntime(functionalSpec(tc.body), nil, &out, &out)
			if err != nil {
				t.Fatal(err)
			}
			_, err = rt.run()
			if tc.failure != "" {
				if err == nil || !strings.Contains(err.Error(), tc.failure) {
					t.Fatalf("want %q; got %v", tc.failure, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.output {
				t.Fatalf("output %q; want %q", out.String(), tc.output)
			}
		})
	}
}

func TestFunctionTypeContracts(t *testing.T) {
	for _, raw := range []string{"Fn[()->nil]", "Fn[(value:i32)->i32]", "Fn[(transform:Fn[(value:i32)->i32])->Fn[()->nil!io]!log]", "Fn[(state:List[i32],value:?str)->R[i32,str]!fs!io]"} {
		parsed, err := ParseFunctionType(raw)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if parsed.String() != raw {
			t.Fatalf("got %s, want %s", parsed.String(), raw)
		}
	}
	for _, raw := range []string{"Fn[i32]", "Fn[(value:i32,value:str)->i32]", "Fn[(value:i32)->nil!madeup]", "Fn[()->nil!io!io]", "Fn[(value:Fn[bad])->i32]", "Fn[()->nil]extra"} {
		if _, err := ParseFunctionType(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestCaptureDeepCopy(t *testing.T) {
	original := &structValue{Type: "Box", Fields: map[string]any{"nested": []value{map[string]any{"value": 2}}}}
	captured, err := freezeValue(original)
	if err != nil {
		t.Fatal(err)
	}
	original.Fields["nested"].([]value)[0].(map[string]any)["value"] = 100
	copy := captured.(*structValue).Fields["nested"].([]value)[0].(map[string]any)
	if copy["value"] != 2 {
		t.Fatal("capture shared nested data")
	}
	// Each invocation also gets an independent snapshot of the stored capture.
	second, err := freezeValue(captured)
	if err != nil {
		t.Fatal(err)
	}
	copy["value"] = 200
	if second.(*structValue).Fields["nested"].([]value)[0].(map[string]any)["value"] != 2 {
		t.Fatal("invocation shared nested data")
	}
}

func TestDeclaredFunctionBoundary(t *testing.T) {
	for _, tc := range []struct{ name, body, signature, argument, failure string }{
		{"input_type", `f(value:"wrong")`, "Fn[(value:i32)->nil]", "println", "expected i32"},
		{"effect_bound", `f(value:1)`, "Fn[(value:i32)->i32!io]", "partial add(amount:1)", "effect !io"},
		{"default_erasure", `f(value:1,amount:default)`, "Fn[(value:i32,amount:i32)->i32]", "add", "no declared default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := functionalSpec("boundary(f:" + tc.argument + ")")
			if tc.name == "input_type" {
				tc.signature = "Fn[(value:i32)->nil!io]"
			}
			effects := []string{}
			if tc.name == "input_type" {
				effects = []string{"io"}
			}
			spec.Modules[0].Funcs = append(spec.Modules[0].Funcs, FuncSpec{Effects: effects, Module: "main", Name: "boundary", BlockBody: true, ReturnType: "nil", Params: []ParamSpec{{Name: "f", Type: tc.signature}}, Body: tc.body + "\nreturn nil"})
			var out bytes.Buffer
			rt, err := newRuntime(spec, nil, &out, &out)
			if err != nil {
				t.Fatal(err)
			}
			_, err = rt.run()
			if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("want %s, got %v", tc.failure, err)
			}
			if out.Len() != 0 {
				t.Fatalf("invalid call produced output: %s", out.String())
			}
		})
	}
}

func TestSpawnCallsUseLocalSnapshot(t *testing.T) {
	spec := functionalSpec(`mut amount = 2
let first = spawn add(value:amount,amount:1)
amount = 100
let second = spawn add(value:amount,amount:1)
let a = await first
let b = await second
println(value:a + b)`)
	var out bytes.Buffer
	rt, err := newRuntime(spec, nil, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.run()
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "104\n" {
		t.Fatalf("spawn output: %s", out.String())
	}
}

func TestFunctionRolesSurviveTypeBoundary(t *testing.T) {
	spec := functionalSpec("println(value:roles(f:subtract))")
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs,
		FuncSpec{Module: "main", Name: "subtract", BlockBody: true, ReturnType: "i32", Params: []ParamSpec{{Name: "value", Type: "i32"}, {Name: "amount", Type: "i32"}}, Body: "return value - amount"},
		FuncSpec{Module: "main", Name: "roles", BlockBody: true, ReturnType: "i32", Params: []ParamSpec{{Name: "f", Type: "Fn[(amount:i32,value:i32)->i32]"}}, Body: "return f(value:7,amount:2)"},
	)
	var out bytes.Buffer
	rt, err := newRuntime(spec, nil, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.run()
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "5\n" {
		t.Fatalf("argument roles changed: %s", out.String())
	}
}

func TestPartialMethodCapturesReceiver(t *testing.T) {
	spec := functionalSpec(`mut box = Box{value:2}
let calculate = partial box.plus(amount:default)
box = Box{value:100}
println(value:calculate(value:3))`)
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs, FuncSpec{Module: "main", Name: "plus", ReceiverType: "Box", BlockBody: true, ReturnType: "i32", Params: []ParamSpec{{Name: "receiver", Type: "Box"}, {Name: "value", Type: "i32"}, {Name: "amount", Type: "i32", Default: "1"}}, Body: "return receiver.value + value + amount"})
	var out bytes.Buffer
	rt, err := newRuntime(spec, nil, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.run()
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "6\n" {
		t.Fatalf("method receiver was rebound: %s", out.String())
	}
}
