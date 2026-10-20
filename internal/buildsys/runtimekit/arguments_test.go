package runtimekit

import (
	"bytes"
	"strings"
	"testing"
)

func TestNamedCalls(t *testing.T) {
	for _, tc := range []struct{ name, body, want, failure string }{
		{"reordered", `println(value: pair(right:"B", left:"A"))`, "A|B\n", ""},
		{"source_order", `println(value: pair(right:mark(value:"B"), left:mark(value:"A")))`, "B\nA\nA|B\n", ""},
		{"alias", `let f = pair
println(value: f(right:"B", left:"A"))`, "A|B\n", ""},
		{"builtin_alias", `let emit = println
emit(value:"alias")`, "alias\n", ""},
		{"method", `let user = User{name:"Toni"}
println(value:user.greet(suffix:"!", prefix:"Hi"))`, "Hi Toni!\n", ""},
		{"multiline", `println(
value: pair(
right: "B",
left: "A",
),
)`, "A|B\n", ""},
		{"missing", `pair(left:mark(value:"must not run"))`, "", `missing argument "right"`},
		{"unknown", `pair(wrong:mark(value:"must not run"), right:"B")`, "", `unknown argument "wrong"`},
		{"empty", `pair()`, "", `missing argument "left"`},
		{"duplicate", `pair(left:"A", left:"B")`, "", `duplicate argument "left"`},
		{"positional", `pair("A", "B")`, "", "named arguments are required"},
		{"mixed", `pair(left:"A", "B")`, "", "named arguments are required"},
		{"missing_comma", `pair(left:"A" right:"B")`, "", "expected comma"},
		{"builtin_wrong_key", `println(message:mark(value:"must not run"))`, "", `unknown argument "message"`},
		{"aliased_wrong_key", `let f = pair
f(wrong:mark(value:"must not run"), right:"B")`, "", `unknown argument "wrong"`},
		{"constructor", `User(name:"Toni")`, "", "record constructors use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			functions := []FuncSpec{
				{Module: "main", Name: "main", Effects: []string{"io"}, BlockBody: true, ReturnType: "i32", Body: tc.body + "\nreturn 0"},
				{Module: "main", Name: "pair", BlockBody: true, ReturnType: "str", Params: []ParamSpec{{Name: "left", Type: "str"}, {Name: "right", Type: "str"}}, Body: `return "${left}|${right}"`},
				{Module: "main", Name: "mark", Effects: []string{"io"}, BlockBody: true, ReturnType: "str", Params: []ParamSpec{{Name: "value", Type: "str"}}, Body: "println(value:value)\nreturn value"},
				{Module: "main", Name: "greet", ReceiverType: "User", BlockBody: true, ReturnType: "str", Params: []ParamSpec{{Name: "u", Type: "User"}, {Name: "prefix", Type: "str"}, {Name: "suffix", Type: "str"}}, Body: `return "${prefix} ${u.name}${suffix}"`},
			}
			rt, err := newRuntime(ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Types: []TypeSpec{{Name: "User", Fields: []FieldSpec{{Name: "name", Type: "str"}}}}, Funcs: functions}}}, nil, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			code, err := rt.run()
			if tc.failure != "" {
				if err == nil || !strings.Contains(err.Error(), tc.failure) {
					t.Fatalf("expected %q, got %v", tc.failure, err)
				}
			} else if err != nil || code != 0 {
				t.Fatalf("code=%d err=%v", code, err)
			}
			if stdout.String() != tc.want {
				t.Fatalf("output %q, want %q", stdout.String(), tc.want)
			}
		})
	}
}

func TestInspectNestedCalls(t *testing.T) {
	sites, err := InspectCalls(`json.decode[Input](text: json.encode(value: Input{name:"a,b:()"})?)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 2 || sites[0].Target != "json.decode" || sites[0].Keys[0] != "text" || sites[1].Target != "json.encode" || sites[1].Keys[0] != "value" {
		t.Fatalf("unexpected calls: %#v", sites)
	}
}
