package format

import (
	"strings"
	"testing"
)

func TestSource(t *testing.T) {
	in := `mod main

import app/router

const Version="0.4"

type Resp { code:i32
body:?str }

fn main()->i32{return 0}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

import app/router

const Version = "0.4"

type Resp {
  code:i32
  body:?str
}

fn main() -> i32 {
  return 0
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_BlockFunction(t *testing.T) {
	in := `mod main

fn route(path:str)->Resp{if path=="/health"=>{return text(code : 200,body : "ok")}
return text(code : 404,body : "not found")}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

fn route(path:str) -> Resp {
  if path == "/health" => { return text ( code : 200 , body : "ok" ) }
  return text ( code : 404 , body : "not found" )
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_SortsEffects(t *testing.T) {
	in := `mod main

fn main()->i32!net!env!log{return 0}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

fn main() -> i32 !env !log !net {
  return 0
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_InterpolationStringBody(t *testing.T) {
	in := `mod main

fn main()->i32!io{println(value : "kiro ${Version}")
return 0}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

fn main() -> i32 !io {
  println ( value : "kiro ${Version}" )
  return 0
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_DocComment(t *testing.T) {
	in := `mod main

///greet returns a greeting.
fn greet(name:str)->str{return "hello ${name}"}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

/// greet returns a greeting.
fn greet(name:str) -> str {
  return "hello ${name}"
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_RejectsExpressionBodyFunctions(t *testing.T) {
	in := `mod main

fn main() -> i32 = 0`
	_, err := Source(in)
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "expression-bodied functions were removed") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestSource_PreservesLiteralContents(t *testing.T) {
	const literal = "A  B = > C . D - > E = = F"
	in := "mod main\n\nfn main() -> i32 !io {\n  println(value: \"" + literal + "\")\n  return 0\n}\n"
	once, err := Source(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(once, "\""+literal+"\"") {
		t.Fatalf("formatter changed a literal: %s", once)
	}
	twice, err := Source(once)
	if err != nil || once != twice {
		t.Fatalf("formatting is not idempotent: %v\n%s", err, twice)
	}
}

func TestSource_PreservesNamedArgumentOrder(t *testing.T) {
	in := "mod main\nfn main() -> i32 {\nf(right:mark(value:2),left:mark(value:1))\nreturn 0\n}\n"
	out, err := Source(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(out, "right :") >= strings.Index(out, "left :") {
		t.Fatalf("formatter reordered argument expressions: %s", out)
	}
}
