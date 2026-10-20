package runtimekit

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func transducerSpec(body string) ProgramSpec {
	spec := functionalSpec(body)
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs,
		FuncSpec{Module: "main", Name: "text", BlockBody: true, ReturnType: "str", Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: `return "${value}"`},
		FuncSpec{Module: "main", Name: "nonempty", BlockBody: true, ReturnType: "bool", Params: []ParamSpec{{Name: "value", Type: "str"}}, Body: `return value != ""`},
		FuncSpec{Module: "main", Name: "join", BlockBody: true, ReturnType: "str", Params: []ParamSpec{{Name: "value", Type: "str"}, {Name: "state", Type: "str"}}, Body: `return state + value`},
		FuncSpec{Module: "main", Name: "pipeline", BlockBody: true, ReturnType: "Transform[i32,i32]", Params: []ParamSpec{{Name: "amount", Type: "i32"}}, Body: `return list.mapping(transform:partial add(amount:amount))`},
		FuncSpec{Module: "main", Name: "typed", BlockBody: true, ReturnType: "List[i32]", Params: []ParamSpec{{Name: "through", Type: "Transform[i32,i32]"}}, Body: `return list.collect(items:[1,2],through:through)`},
	)
	return spec
}

func TestTransducerRuntime(t *testing.T) {
	for _, tc := range []struct{ name, body, output, failure string }{
		{"fused", `let xf = list.compose(first:list.filtering(predicate:partial keep(minimum:2)),then:list.mapping(transform:partial add(amount:1)))
println(value:list.transduce(items:[1,2,3],through:xf,initial:0,reduce:sum))`, "7\n", ""},
		{"collect_order_and_reuse", `let xf = list.compose(first:list.mapping(transform:text),then:list.filtering(predicate:nonempty))
println(value:list.transduce(items:[1,2,3],through:xf,initial:"",reduce:join))
println(value:list.collect(items:[4,5],through:xf))`, "123\n[4 5]\n", ""},
		{"empty", `let xf = list.mapping(transform:partial add(amount:1))
println(value:list.transduce(items:[],through:xf,initial:9,reduce:sum))
println(value:list.collect(items:[],through:xf))`, "9\n[]\n", ""},
		{"filtered_poison", `let xf = list.compose(first:list.filtering(predicate:partial keep(minimum:10)),then:list.mapping(transform:wrong))
println(value:list.transduce(items:[1,2,3],through:xf,initial:5,reduce:sum))`, "5\n", ""},
		{"captures", `mut amount = 2
let xf = list.mapping(transform:partial add(amount:amount))
let run = partial list.transduce(through:xf,initial:0,reduce:sum)
amount = 100
println(value:run(items:[1,2]))
println(value:run(items:[1,2]))`, "7\n7\n", ""},
		{"typed_factory", `println(value:typed(through:pipeline(amount:2)))`, "[3 4]\n", ""},
		{"aliases", `let mapping = list.mapping
let collect = list.collect
println(value:collect(items:[1],through:mapping(transform:partial add(amount:2))))`, "[3]\n", ""},
		{"nested_compose", `let up = list.mapping(transform:partial add(amount:1))
let xf = list.compose(first:up,then:list.compose(first:up,then:up))
println(value:list.collect(items:[1,2],through:xf))`, "[4 5]\n", ""},
		{"effectful_mapper", `list.mapping(transform:show)`, "", "pure callback"},
		{"mapper_label", `list.mapping(transform:label)`, "", "unknown argument"},
		{"bad_predicate", `list.filtering(predicate:partial add(amount:1))`, "", "return bool"},
		{"incompatible_stages", `list.compose(first:list.mapping(transform:text),then:list.mapping(transform:partial add(amount:1)))`, "", "cannot pass str output to i32 input"},
		{"nontransform", `list.collect(items:[],through:1)`, "", "must be a Transform"},
		{"nontransform_compose", `list.compose(first:1,then:list.mapping(transform:text))`, "", "must be a Transform"},
		{"bad_reducer_empty", `list.transduce(items:[],through:list.mapping(transform:text),initial:0,reduce:sum)`, "", "cannot accept str output"},
		{"initial_empty", `list.transduce(items:[],through:list.mapping(transform:partial add(amount:1)),initial:"bad",reduce:sum)`, "", "initial"},
		{"impure_reducer_empty", `list.transduce(items:[],through:list.mapping(transform:text),initial:0,reduce:show)`, "", "pure callback"},
		{"state_return_empty", `list.transduce(items:[],through:list.mapping(transform:text),initial:0,reduce:bad_reduce)`, "", "match state type"},
		{"wrong_input", `list.collect(items:["wrong"],through:list.mapping(transform:partial add(amount:1)))`, "", "item 0"},
		{"result_type_check", `list.collect(items:[1],through:list.mapping(transform:wrong))`, "", "stage 0"},
		{"fold_removed", `let old = list.fold`, "", "renamed to list.reduce"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			rt, err := newRuntime(transducerSpec(tc.body), nil, &output, &output)
			if err != nil {
				t.Fatal(err)
			}
			_, err = rt.run()
			if tc.failure != "" {
				if err == nil || !strings.Contains(err.Error(), tc.failure) {
					t.Fatalf("want %q, got %v", tc.failure, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if output.String() != tc.output {
				t.Fatalf("output %q, want %q", output.String(), tc.output)
			}
		})
	}
}

func TestTransducerFailureOrder(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"eager", `let selected = list.filter(items:[1,2],predicate:predicate)
list.map(items:selected,transform:mapper)`, "i32 arithmetic overflow"},
		{"fused", `let xf = list.compose(first:list.filtering(predicate:predicate),then:list.mapping(transform:mapper))
list.transduce(items:[1,2],through:xf,initial:0,reduce:sum)`, "item 0: stage 1: division by zero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := transducerSpec(tc.body)
			spec.Modules[0].Funcs = append(spec.Modules[0].Funcs,
				FuncSpec{Module: "main", Name: "predicate", BlockBody: true, ReturnType: "bool", Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: "let n = value + 2147483646\nreturn n > 0"},
				FuncSpec{Module: "main", Name: "mapper", BlockBody: true, ReturnType: "i32", Params: []ParamSpec{{Name: "value", Type: "i32"}}, Body: "return 1 / (value - 1)"})
			rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			_, err = rt.run()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}

func TestTransformContractBoundaries(t *testing.T) {
	rt, err := newRuntime(transducerSpec(""), nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	makeMapping := func(name string) *transformValue {
		v, err := transformOperation("mapping")(&execContext{rt: rt}, nil, []value{functionValue{Module: "main", Name: name}})
		if err != nil {
			t.Fatal(err)
		}
		return v.(*transformValue)
	}
	xf := makeMapping("text")
	for _, tc := range []struct {
		typ string
		ok  bool
	}{{"Transform[i32,str]", true}, {"Transform[i32,Any]", true}, {"Transform[str,str]", false}, {"Transform[Any,str]", false}, {"Transform[i32,i32]", false}} {
		err := rt.validateValue(xf, tc.typ)
		if (err == nil) != tc.ok {
			t.Fatalf("%s: %v", tc.typ, err)
		}
	}
	// A widened output contract hides the concrete output type from composition.
	view := ascribeValue(xf, "Transform[i32,Any]").(*transformValue)
	if view.InputType != "i32" || view.OutputType != "Any" {
		t.Fatal("declared transform boundary was lost")
	}
	for _, typ := range []string{"Transform", "Transform[i32]", "Transform[i32,str,bool]", "Transform[,str]", "Transform[i32,Fn[bad]]", "Transform[i32,str!io]", "Transform[?,str]", "Transform[i32,str extra]"} {
		if err := validateTypeContract(typ); err == nil {
			t.Fatalf("accepted %s", typ)
		}
	}
}

// Compare both paths with an independent Go calculation over varied list lengths.
func TestFusedCollectionsMatchReference(t *testing.T) {
	rt, err := newRuntime(transducerSpec(""), nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &execContext{rt: rt}
	add := &partialValue{Target: functionValue{Module: "main", Name: "add"}, Bound: map[string]value{"amount": 1}, Signature: FunctionType{Params: []ParamSpec{{Name: "value", Type: "i32"}}, ReturnType: "i32"}}
	keep := &partialValue{Target: functionValue{Module: "main", Name: "keep"}, Bound: map[string]value{"minimum": 0}, Signature: add.Signature}
	keep.Signature.ReturnType = "bool"
	mapPlan, err := transformOperation("mapping")(ctx, nil, []value{add})
	if err != nil {
		t.Fatal(err)
	}
	filterPlan, err := transformOperation("filtering")(ctx, nil, []value{keep})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := transformOperation("compose")(ctx, nil, []value{filterPlan, mapPlan})
	if err != nil {
		t.Fatal(err)
	}
	sum := functionValue{Module: "main", Name: "sum"}
	for n := 0; n < 80; n++ {
		items := make([]value, n)
		want := 0
		for i := range items {
			number := (i*17+n)%41 - 20
			items[i] = number
			if number >= 0 {
				want += number + 1
			}
		}
		selected, err := listOperation("filter")(ctx, nil, []value{items, keep})
		if err != nil {
			t.Fatal(err)
		}
		mapped, err := listOperation("map")(ctx, nil, []value{selected, add})
		if err != nil {
			t.Fatal(err)
		}
		eager, err := listOperation("reduce")(ctx, nil, []value{mapped, 0, sum})
		if err != nil {
			t.Fatal(err)
		}
		fused, err := transformOperation("transduce")(ctx, nil, []value{items, plan, 0, sum})
		if err != nil {
			t.Fatal(err)
		}
		if eager != want || fused != want {
			t.Fatalf("length %d: eager=%v fused=%v reference=%v", n, eager, fused, want)
		}
	}
}

func BenchmarkCollectionPipelines(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		for _, fused := range []bool{false, true} {
			name := "eager"
			if fused {
				name = "fused"
			}
			b.Run(fmt.Sprintf("%s/%d", name, n), func(b *testing.B) {
				rt, err := newRuntime(transducerSpec(""), nil, io.Discard, io.Discard)
				if err != nil {
					b.Fatal(err)
				}
				ctx := &execContext{rt: rt}
				makePartial := func(name, key string, v int, ret string) *partialValue {
					return &partialValue{Target: functionValue{Module: "main", Name: name}, Bound: map[string]value{key: v}, Signature: FunctionType{Params: []ParamSpec{{Name: "value", Type: "i32"}}, ReturnType: ret}}
				}
				mapper, predicate := makePartial("add", "amount", 1, "i32"), makePartial("keep", "minimum", 0, "bool")
				mapPlan, err := transformOperation("mapping")(ctx, nil, []value{mapper})
				if err != nil {
					b.Fatal(err)
				}
				filterPlan, err := transformOperation("filtering")(ctx, nil, []value{predicate})
				if err != nil {
					b.Fatal(err)
				}
				plan, err := transformOperation("compose")(ctx, nil, []value{filterPlan, mapPlan})
				if err != nil {
					b.Fatal(err)
				}
				items := make([]value, n)
				for i := range items {
					items[i] = i - n/2
				}
				sum := functionValue{Module: "main", Name: "sum"}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var result value
					if fused {
						result, err = transformOperation("transduce")(ctx, nil, []value{items, plan, 0, sum})
					} else {
						var selected, mapped value
						selected, err = listOperation("filter")(ctx, nil, []value{items, predicate})
						if err == nil {
							mapped, err = listOperation("map")(ctx, nil, []value{selected, mapper})
						}
						if err == nil {
							result, err = listOperation("reduce")(ctx, nil, []value{mapped, 0, sum})
						}
					}
					if err != nil || result == nil {
						b.Fatalf("%v: %v", result, err)
					}
				}
			})
		}
	}
}

func TestTransformTypeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		accepted, provided string
		ok                 bool
	}{
		{"Any", "List[str]", true}, {"List[i32]", "List[Any]", false}, {"List[Any]", "List[i32]", true},
		{"?i32", "i32", true}, {"?i32", "nil", true}, {"i32", "?i32", false},
		{"R[i32,str]", "R[Any,str]", false}, {"R[Any,str]", "R[i32,str]", true},
		{"Transform[i32,str]", "Transform[Any,str]", true}, {"Transform[Any,str]", "Transform[i32,str]", false},
		{"Fn[(value:i32)->str]", "Fn[(value:Any)->str]", true},
	} {
		if got := contractAcceptsType(tc.accepted, tc.provided); got != tc.ok {
			t.Fatalf("accept %s from %s: %v", tc.accepted, tc.provided, got)
		}
	}
}
