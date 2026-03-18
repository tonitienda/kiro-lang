package parser

import (
	"testing"

	"github.com/kiro-lang/kiro/internal/ast"
)

func TestParseFile(t *testing.T) {
	src := `mod main

import app/router

const Version = "0.4"

type Resp {
  code:i32
  body:?str
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
	if len(file.Decls) != 3 {
		t.Fatalf("decl count = %d", len(file.Decls))
	}
	cd, ok := file.Decls[0].(ast.ConstDecl)
	if !ok {
		t.Fatalf("decl[0] type = %T", file.Decls[0])
	}
	if cd.Name != "Version" || cd.Value != "0.4" {
		t.Fatalf("const = %#v", cd)
	}
	td, ok := file.Decls[1].(ast.TypeDecl)
	if !ok {
		t.Fatalf("decl[1] type = %T", file.Decls[1])
	}
	if td.Fields[1].Type != "?str" {
		t.Fatalf("optional field type = %q", td.Fields[1].Type)
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

func TestParseMethodDecl(t *testing.T) {
	src := `mod main

type User {
  name:str
}

fn (u:User) display() -> ?str =
  u.name
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("decl count = %d", len(file.Decls))
	}
	fd, ok := file.Decls[1].(ast.FuncDecl)
	if !ok {
		t.Fatalf("decl[1] type = %T", file.Decls[1])
	}
	if fd.Receiver == nil {
		t.Fatalf("expected receiver")
	}
	if fd.Receiver.Name != "u" || fd.Receiver.Type != "User" {
		t.Fatalf("receiver = %#v", *fd.Receiver)
	}
	if fd.Name != "display" {
		t.Fatalf("name = %q", fd.Name)
	}
	if fd.ReturnType != "?str" {
		t.Fatalf("return type = %q", fd.ReturnType)
	}
}

func TestParseFuncEffects(t *testing.T) {
	src := `mod main

fn main() -> i32 !net !env !log {
  return 0
}
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fd := file.Decls[0].(ast.FuncDecl)
	if got := fd.EffectNames(); len(got) != 3 || got[0] != "net" || got[1] != "env" || got[2] != "log" {
		t.Fatalf("effects = %#v", got)
	}
}

func TestParseDocCommentOnFunc(t *testing.T) {
	src := `mod main

/// greet returns a greeting.
fn greet(name:str) -> str =
  "hello ${name}"
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fd, ok := file.Decls[0].(ast.FuncDecl)
	if !ok {
		t.Fatalf("decl[0] type = %T", file.Decls[0])
	}
	if len(fd.Doc) != 1 || fd.Doc[0] != "greet returns a greeting." {
		t.Fatalf("doc = %#v", fd.Doc)
	}
}

func TestParseDocCommentBeforeImportFails(t *testing.T) {
	src := `mod main

/// bad location
import env
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseGenericTypeRef(t *testing.T) {
	src := `mod main

fn load() -> R[Config,str] =
  Ok(Config{})
`
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fd, ok := file.Decls[0].(ast.FuncDecl)
	if !ok {
		t.Fatalf("decl[0] type = %T", file.Decls[0])
	}
	if fd.ReturnType != "R[Config,str]" {
		t.Fatalf("return type = %q", fd.ReturnType)
	}
}
