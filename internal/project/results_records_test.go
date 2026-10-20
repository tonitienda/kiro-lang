package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResultsAndRecordChecking(t *testing.T) {
	const prelude = `mod main
import json
import parse
type Child { quantity:i32 }
type Input { child:Child note:?str }
fn leaf()->R[i32,str] {return Err(error:"domain")}
`
	for _, tc := range []struct{ name, source, failure string }{
		{"propagate", `fn parent()->R[str,str] {let n = leaf()?
return Ok(value:"${n}")}`, ""},
		{"invalid_boundary", `fn run()->i32 {return leaf()?}`, "requires an R[T,E] return type"},
		{"invalid_operand", `fn run()->R[i32,str] {let n = 1?
return Ok(value:n)}`, "invalid ? on non-result"},
		{"error_type", `fn run()->R[i32,nil] {let n = leaf()?
return Ok(value:n)}`, "error type str is incompatible"},
		{"nil_error", `fn run()->R[i32,nil] {return Err(error:nil)}`, ""},
		{"bad_nil_error", `fn run()->R[i32,str] {return Err(error:nil)}`, "return type"},
		{"spawn_local_boundary", `fn run()->R[Any,str] {return Ok(value:spawn leaf()? )}`, "requires an R[T,E] return type"},
		{"callback_result", `fn call(f:Fn[()->R[i32,str]])->R[i32,str] {let n = f()?
return Ok(value:n)}`, ""},
		{"interpolated_boundary", `fn run()->str {return "${leaf()?}"}`, "requires an R[T,E] return type"},
		{"interpolated_effect", `fn noisy()->nil !io {println(value:1)
return nil}
fn run()->str {return "${noisy()}"}`, "effect !io"},
		{"record_valid", `fn run()->Input {return Input{child:Child{quantity:2} note:nil}}`, ""},
		{"record_missing", `fn run()->Child {return Child{}}`, "missing required field"},
		{"record_optional_explicit", `fn run()->Input {return Input{child:Child{quantity:2}}}`, "missing required field"},
		{"record_unknown", `fn run()->Child {return Child{quantity:1 other:2}}`, "unknown field"},
		{"record_duplicate", `fn run()->Child {return Child{quantity:1 quantity:2}}`, "duplicate field"},
		{"record_wrong", `fn run()->Child {return Child{quantity:"2"}}`, "expected i32, got str"},
		{"record_nested_wrong", `fn run()->Input {return Input{child:1 note:nil}}`, "expected main.Child"},
		{"duplicate_schema", `type Bad { value:i32 value:str }`, "duplicate field"},
		{"unknown_schema", `fn run()->R[Any,str] {return json.decode[Missing](text:"{}")}`, "unknown record type"},
		{"unsupported_schema", `fn run()->R[Any,str] {return json.decode[Fn[()->nil]](text:"{}")}`, "not a JSON schema type"},
		{"missing_schema", `fn run()->R[Any,str] {return json.decode(text:"{}")}`, "requires one explicit schema"},
		{"generic_schema", `fn run()->R[List[Child],str] {return json.decode[List[Child]](text:"[]")}`, ""},
		{"schema_mismatch", `fn run()->R[i32,str] {return json.decode[Child](text:"{}")}`, "return type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, "main.ki"), prelude+tc.source+"\n")
			_, err := Load(dir)
			if tc.failure == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("want %q, got %v", tc.failure, err)
			}
		})
	}
}
