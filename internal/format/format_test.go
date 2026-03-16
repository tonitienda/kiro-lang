package format

import "testing"

func TestSource(t *testing.T) {
	in := `mod main

import app/router

type Resp { code:i32
body:str }

fn main()->i32= 0`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

import app/router

type Resp {
  code:i32
  body:str
}

fn main() -> i32 =
  0
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_BlockFunction(t *testing.T) {
	in := `mod main

fn route(path:str)->Resp{if path=="/health"=>{return text(200,"ok")}
return text(404,"not found")}`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

fn route(path:str) -> Resp {
  if path == "/health" => { return text ( 200 , "ok" ) }
  return text ( 404 , "not found" )
}
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}

func TestSource_Method(t *testing.T) {
	in := `mod main

type User {name:str}

fn (u:User)display()->str=u.name`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

type User {
  name:str
}

fn (u:User) display() -> str =
  u.name
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}
