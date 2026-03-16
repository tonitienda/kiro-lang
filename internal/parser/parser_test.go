package parser

import "testing"

func TestParseFile(t *testing.T) {
	src := `mod main

type Resp {
  code:i32
  body:str
}

fn main() -> i32 =
  0
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if file.Module != "main" {
		t.Fatalf("module = %q", file.Module)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("decl count = %d", len(file.Decls))
	}
}
