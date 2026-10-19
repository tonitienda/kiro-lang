package runtimekit

import (
	"bytes"
	"strings"
	"testing"
)

func TestBracedWhenExecution(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"continuation", `let value = "hit"
when (value) {
  "hit" => { println(value : "first") }
  _ => { println(value : "wrong") }
}
println(value : "after")
return 0`, "first\nafter\n"},
		{"nested", `when ("outer") {
  "outer" => {
    when ("inner") {
      "inner" => { println(value : "nested") }
      _ => { println(value : "wrong inner") }
    }
    println(value : "after inner")
  }
  _ => { println(value : "wrong outer") }
}
println(value : "after outer")
return 0`, "nested\nafter inner\nafter outer\n"},
		{"selector", `let item = Box{name:"hit"}
when (item.name) {
  "hit" => { println(value : "selector") }
  _ => { println(value : "wrong") }
}
return 0`, "selector\n"},
		{"call_with_record", `when (label(item : Box{name:"hit"})) {
  "hit" => { println(value : "call") }
  _ => { println(value : "wrong") }
}
return 0`, "call\n"},
		{"parenthesized_record", `when ((Box{name:"hit"}).name) {
  "hit" => { println(value : "record") }
  _ => { println(value : "wrong") }
}
return 0`, "record\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			spec := ProgramSpec{EntryModule: "main", Modules: []ModuleSpec{{Name: "main", Funcs: []FuncSpec{
				{Module: "main", Name: "main", Effects: []string{"io"}, ReturnType: "i32", BlockBody: true, Body: tc.body},
				{Module: "main", Name: "label", Params: []ParamSpec{{Name: "item", Type: "Box"}}, ReturnType: "str", BlockBody: true, Body: "return item.name"},
			}}}}
			rt, err := newRuntime(spec, nil, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			code, err := rt.run()
			if err != nil || code != 0 || stdout.String() != tc.want {
				t.Fatalf("code=%d err=%v stdout=%q, want %q", code, err, stdout.String(), tc.want)
			}
		})
	}
}

func TestUnbracedWhenRejected(t *testing.T) {
	for _, source := range []string{
		"when 1\n1 => { return 0 }\n",
		"when 1 {\n1 => { return 0 }\n}\n",
		"when (1)\n1 => { return 0 }\n",
	} {
		p, err := newBodyParser(source)
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.parseStatementsUntil("")
		if err == nil || !strings.Contains(err.Error(), "when requires a delimited subject and case block") {
			t.Fatalf("source %q: expected migration diagnostic, got %v", source, err)
		}
	}
}
