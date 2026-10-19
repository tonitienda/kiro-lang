package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamedCallChecking(t *testing.T) {
	for _, tc := range []struct{ name, body, failure string }{
		{"reordered", `pair(right:"B",left:"A")`, ""},
		{"positional", `pair("A","B")`, "named arguments are required"},
		{"mixed", `pair(left:"A","B")`, "named arguments are required"},
		{"unknown", `pair(wrong:"A",right:"B")`, `unknown argument "wrong"`},
		{"missing", `pair(left:"A")`, `missing argument "right"`},
		{"empty", `pair()`, `missing argument "left"`},
		{"duplicate", `pair(left:"A",left:"B")`, `duplicate argument "left"`},
		{"missing_comma", `pair(left:"A" right:"B")`, "expected comma"},
		{"missing_expression", `pair(left:, right:"B")`, "missing expression"},
		{"builtin", `println(message:"wrong")`, `unknown argument "message"`},
		{"generic", `json.decode[Input](value:"{}")`, `unknown argument "value"`},
		{"module", `util.combine(right:"B",left:"A")`, ""},
		{"module_bad", `util.combine(left:"A")`, `missing argument "right"`},
		{"method", `let u = User{name:"Toni"}
u.greet(prefix:"Hi")`, ""},
		{"method_bad", `let u = User{name:"Toni"}
u.greet(wrong:"Hi")`, `unknown argument "wrong"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "util"), 0o755); err != nil {
				t.Fatal(err)
			}
			mustWrite(t, filepath.Join(dir, "util", "main.ki"), "mod util\nfn combine(left:str,right:str) -> str {\nreturn left + right\n}\n")
			mustWrite(t, filepath.Join(dir, "main.ki"), `mod main
import util
import json
type User { name:str }
type Input { name:str }
fn (u:User) greet(prefix:str) -> str { return prefix + u.name }
fn pair(left:str,right:str) -> str { return left + right }
fn main() -> i32 !io {
`+tc.body+"\nreturn 0\n}\n")
			_, err := Load(dir)
			if tc.failure == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("expected %q, got %v", tc.failure, err)
			}
		})
	}
}

func TestDuplicateParameterRejected(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), "mod main\nfn pair(value:str,value:str) -> str { return value }\n")
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), `duplicate parameter "value"`) {
		t.Fatalf("expected duplicate parameter, got %v", err)
	}
}
