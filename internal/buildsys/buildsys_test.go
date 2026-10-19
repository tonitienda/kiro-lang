package buildsys

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildCreatesRunnableBinary(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn main() -> i32 !io {
  println(value : "hello from buildsys")
  return 0
}
`)
	out := filepath.Join(dir, binaryName("app"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("built binary failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "hello from buildsys") {
		t.Fatalf("built binary output = %q, want greeting", string(output))
	}
}

func TestBuildRunReportsInvalidResultUnwrap(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn value() -> i32 {
  return 1
}

fn main() -> i32 {
  let n = value()?
  return n
}
`)
	out := filepath.Join(dir, binaryName("bad-unwrap"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("expected runtime failure, got success: %s", output)
	}
	if !strings.Contains(string(output), "invalid ? on non-result value") {
		t.Fatalf("runtime output = %q", string(output))
	}
}

func TestBuildRunReportsAwaitHint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ki"), `mod main

fn value() -> i32 {
  return 1
}

fn main() -> i32 {
  let n = await value()
  return n
}
`)
	out := filepath.Join(dir, binaryName("bad-await"))
	result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	run := exec.Command(out)
	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("expected runtime failure, got success: %s", output)
	}
	if !strings.Contains(string(output), "await requires a spawned task") || !strings.Contains(string(output), "call spawn first") {
		t.Fatalf("runtime output = %q", string(output))
	}
}

func TestQuoteServiceExample(t *testing.T) {
	entry := filepath.Join("..", "..", "examples", "quote_service")
	out := filepath.Join(t.TempDir(), binaryName("quote-tests"))
	result, err := Build(Options{Entry: entry, Out: out, Mode: ModeTest})
	if err != nil {
		t.Fatalf("build quote service tests: %v", err)
	}
	defer os.RemoveAll(result.WorkDir)
	output, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("quote service tests failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "ok\t14 test(s)") {
		t.Fatalf("expected the complete quote service suite to execute:\n%s", output)
	}
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestFunctionalExampleNativeBinary(t *testing.T) {
	for _, tc := range []struct {
		mode BuildMode
		want string
	}{{ModeBuild, "3700\n"}, {ModeTest, "ok\t4 test(s)"}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			out := filepath.Join(t.TempDir(), binaryName("functional"))
			result, err := Build(Options{Entry: filepath.Join("..", "..", "examples", "functional"), Out: out, Mode: tc.mode})
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(result.WorkDir)
			output, err := exec.Command(out).CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("want %q, got %s", tc.want, output)
			}
		})
	}
}

func TestBuildTransducerFailureOrder(t *testing.T) {
	const declarations = `mod main
import list
fn predicate(value:i32)->bool {
let n = value + 2147483646
return n > 0
}
fn mapper(value:i32)->i32 {return 1 / (value - 1)}
fn sum(state:i32,value:i32)->i32 {return state + value}
`
	for _, tc := range []struct{ name, body, want string }{
		{"eager", `let selected = list.filter(items:[1,2],predicate:predicate)
list.map(items:selected,transform:mapper)`, "i32 arithmetic overflow"},
		{"fused", `let through = list.compose(first:list.filtering(predicate:predicate),then:list.mapping(transform:mapper))
list.transduce(items:[1,2],through:through,initial:0,reduce:sum)`, "item 0: stage 1: division by zero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "main.ki"), declarations+"fn main()->nil {\n"+tc.body+"\nreturn nil\n}\n")
			out := filepath.Join(dir, binaryName("failure-order"))
			result, err := Build(Options{Entry: dir, Out: out, Mode: ModeBuild})
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(result.WorkDir)
			output, err := exec.Command(out).CombinedOutput()
			if err == nil || !strings.Contains(string(output), tc.want) {
				t.Fatalf("want %s; got %v: %s", tc.want, err, output)
			}
		})
	}
}
