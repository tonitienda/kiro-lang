package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

func TestReturnPaths(t *testing.T) {
	const prelude = `mod main
import test
type Failure {variant Missing {} variant Offline {}}
fn leaf()->R[i32,str] {return Ok(value:1)}
`
	for _, tc := range []struct {
		name, source string
		missing      bool
	}{
		{"empty", `fn run()->i32 {}`, true},
		{"empty_result", `fn run()->R[nil,str] {}`, true},
		{"nil", `fn run()->nil {}`, false},
		{"optional", `fn run()->?str {}`, false},
		{"any", `fn run()->Any {}`, false},
		{"unwrap_may_succeed", `fn run()->R[i32,str] {let n = leaf()?}`, true},
		{"loop_may_be_empty", `fn run(items:List[i32])->i32 {for value in items {return value}}`, true},
		{"literal_loop_conservative", `fn run()->i32 {for value in [1] {return value}}`, true},
		{"loop_fallback", `fn run(items:List[i32])->i32 {for value in items {return value}
return 0}`, false},
		{"primitive_partial", `fn run(value:i32)->i32 {when (value) {1 => {return 1}}}`, true},
		{"bool_total", `fn run(value:bool)->i32 {when (value) {true => {return 1}
false => {return 0}}}`, false},
		{"one_fallthrough_arm", `fn run(value:i32)->i32 {when (value) {1 => {return 1}
_ => {test.eq(actual:1,expected:1)}}}`, true},
		{"group", `fn run()->i32 {group {return 1}}`, false},
		{"variant_total", `fn run(error:Failure)->i32 {when (error) {Failure.Missing() => {return 1}
Failure.Offline() => {return 0}}}`, false},
		{"result_total", `fn run()->i32 {when (leaf()) {Ok(value:let n) => {return n}
Err(error:_) => {return 0}}}`, false},
		{"expression_value_is_not_return", `fn run()->i32 {let n = when (leaf()) {Ok(value:let value) => value
Err(error:_) => 0}}`, true},
		{"expression_control_return", `fn run()->i32 {let n = 2 + when (leaf()) {Ok(value:_) => {return 1}
Err(error:_) => {return 0}}}`, false},
		{"callback_return", `fn run()->Fn[()->i32] {}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "main.ki")
			mustWrite(t, path, prelude+tc.source)
			_, err := Load(dir)
			if !tc.missing {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic *runtimekit.Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != "KIRO_MISSING_RETURN" {
				t.Fatalf("expected missing return, got %v", err)
			}
			if diagnostic.File != path || diagnostic.Symbol != "main.run" || diagnostic.LocationKind != "declaration" || diagnostic.Range == nil {
				t.Fatalf("incomplete context: %+v", diagnostic)
			}
		})
	}
}

func TestStructuredFeedbackAndSyntaxOnlyDiscovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ki")
	source := "mod main\n\n/// purpose\nfn run()->i32 {return \"bad\"}\n"
	mustWrite(t, path, source)
	_, err := Load(dir)
	var diagnostic *runtimekit.Diagnostic
	if !errors.As(err, &diagnostic) {
		t.Fatal(err)
	}
	if diagnostic.Code != "KIRO_TYPE_MISMATCH" || diagnostic.Expected != "i32" || diagnostic.Actual != "str" || diagnostic.Range.Start.Line != 4 || diagnostic.Range.Start.Column != 4 {
		t.Fatalf("%+v", diagnostic)
	}
	p, err := Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	symbols := p.Symbols()
	if len(symbols) != 1 || symbols[0].ID != "main.run" || symbols[0].ReturnType != "i32" || strings.Join(symbols[0].Doc, "") != "purpose" {
		t.Fatalf("%+v", symbols)
	}
	rev := p.Revision()
	other := t.TempDir()
	mustWrite(t, filepath.Join(other, "main.ki"), source)
	clone, err := Parse(other)
	if err != nil {
		t.Fatal(err)
	}
	if clone.Revision() != rev {
		t.Fatal("revision depends on absolute directory")
	}
	mustWrite(t, path, source+"\n")
	p, err = Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Revision() == rev {
		t.Fatal("revision ignored source change")
	}
}

func TestOverlayChecksDoNotWriteFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ki")
	original := "mod main\nfn main()->i32 {return 0}\n"
	mustWrite(t, path, original)
	_, err := LoadOverlay(dir, map[string]string{path: "mod main\nfn main()->i32 {}\n"})
	var diagnostic *runtimekit.Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "KIRO_MISSING_RETURN" {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != original {
		t.Fatalf("disk changed: %s %v", actual, err)
	}
	added := filepath.Join(dir, "helper.ki")
	p, err := LoadOverlay(dir, map[string]string{added: "mod helper\nfn value()->i32 {return 1}\n"})
	if err != nil || len(p.Files) != 2 {
		t.Fatalf("new overlay: %v", err)
	}
	if _, err := os.Stat(added); !os.IsNotExist(err) {
		t.Fatal("virtual file was written")
	}
}
