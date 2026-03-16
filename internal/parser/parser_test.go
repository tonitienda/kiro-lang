package parser

import (
	"testing"

	"github.com/kiro-lang/kiro/internal/ast"
)

func TestParseFile(t *testing.T) {
	src := `mod main

import app/router

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
	if len(file.Imports) != 1 || file.Imports[0] != "app/router" {
		t.Fatalf("imports = %#v", file.Imports)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("decl count = %d", len(file.Decls))
	}
}

func TestParseBlockFunction(t *testing.T) {
	src := `mod main

fn route(path:str) -> Resp {
  if path == "/health" => {
    return text(200, "ok")
  }

  return text(404, "not found")
}
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fd, ok := file.Decls[0].(ast.FuncDecl)
	if !ok {
		t.Fatalf("decl[0] type = %T", file.Decls[0])
	}
	if !fd.BlockBody {
		t.Fatalf("expected block body")
	}
	if fd.Body == "" {
		t.Fatalf("expected non-empty body")
	}
}
