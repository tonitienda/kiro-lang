package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

func captureTool(t *testing.T, args ...string) (toolReport, error) {
	t.Helper()
	old := os.Stdout
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = file
	defer func() { os.Stdout = old; file.Close() }()
	callErr := Run(args)
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	var report toolReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("not one JSON document: %q: %v", raw, err)
	}
	if report.Schema != toolSchema {
		t.Fatalf("schema=%s", report.Schema)
	}
	if !report.OK {
		var status *ExitError
		if !errors.As(callErr, &status) || !status.Reported || status.Code != 1 {
			t.Fatalf("status=%v", callErr)
		}
	}
	return report, callErr
}
func TestAgentJSONCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ki")
	source := "mod main\nfn main()->i32 {return \"bad\"}\n"
	write(t, path, source)
	report, err := captureTool(t, "check", "--json", dir)
	if err == nil || report.OK || len(report.Diagnostics) != 1 {
		t.Fatalf("%+v %v", report, err)
	}
	d := report.Diagnostics[0]
	if d.Code != "KIRO_TYPE_MISMATCH" || d.Symbol != "main.main" || d.Expected != "i32" || d.Actual != "str" || d.File != path {
		t.Fatalf("%+v", d)
	}
	report, err = captureTool(t, "symbols", dir, "--symbol", "main.main", "--json")
	if err != nil || !report.OK {
		t.Fatalf("symbols %v", err)
	}
	data := report.Data.(map[string]any)
	if data["validation"] != "syntax_only" || data["revision"] == "" {
		t.Fatal(data)
	}
	report, err = captureTool(t, "describe", "json.decode", "--json")
	if err != nil || !report.OK {
		t.Fatal(err)
	}
	data = report.Data.(map[string]any)
	if data["name"] != "json.decode" {
		t.Fatal(data)
	}
	report, err = captureTool(t, "describe", "--json")
	if err != nil || !report.OK {
		t.Fatal(err)
	}
	data = report.Data.(map[string]any)
	if len(data["builtins"].(map[string]any)) != len(runtimekit.BuiltinSignatures()) {
		t.Fatal("builtin discovery drift")
	}
	for _, args := range [][]string{{"check", "--json"}, {"check", dir, "--typo", "--json"}, {"symbols", dir, "--symbol", "--json"}, {"describe", "not.real", "--json"}, {"fmt", path, "--json"}} {
		report, err = captureTool(t, args...)
		if err == nil || report.OK || len(report.Diagnostics) == 0 {
			t.Fatalf("%v: %+v %v", args, report, err)
		}
	}
}
func TestFormatterCheckAndPreflight(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ki")
	source := "mod main\nfn main()->i32{return 0}\n"
	write(t, path, source)
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	report, err := captureTool(t, "fmt", dir, "--check", "--json")
	if err == nil || report.Diagnostics[0].Code != "KIRO_FORMAT_REQUIRED" {
		t.Fatalf("%+v %v", report, err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != source {
		t.Fatal("--check wrote source")
	}
	report, err = captureTool(t, "lint", dir, "--json")
	if err == nil || report.Diagnostics[0].Code != "KIRO_FORMAT_REQUIRED" {
		t.Fatalf("lint: %+v %v", report, err)
	}
	bad := filepath.Join(dir, "z.ki")
	write(t, bad, "mod main\nfn broken(\n")
	if err := Run([]string{"fmt", dir}); err == nil {
		t.Fatal("invalid input accepted")
	}
	raw, _ = os.ReadFile(path)
	if string(raw) != source {
		t.Fatal("formatting wrote before preflight completed")
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"fmt", dir}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("file mode changed")
	}
	report, err = captureTool(t, "lint", dir, "--json")
	if err != nil || !report.OK {
		t.Fatalf("%+v %v", report, err)
	}
	report, err = captureTool(t, "fmt", path, "--check", "--json")
	if err != nil || !report.OK {
		t.Fatal(err)
	}
	hidden := filepath.Join(dir, ".kiro")
	if err := os.MkdirAll(hidden, 0755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(hidden, "bad.ki"), "broken")
	if err := Run([]string{"fmt", dir}); err != nil {
		t.Fatal("hidden guide should not be scanned", err)
	}
}
func TestMachineMissingVariantPayload(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "main.ki"), `mod main
type Error {variant A {} variant B {}}
fn run(error:Error)->nil {when (error) {Error.A() => {return nil}}}
`)
	report, err := captureTool(t, "check", dir, "--json")
	if err == nil {
		t.Fatal("missing variant accepted")
	}
	d := report.Diagnostics[0]
	if d.Code != "KIRO_NON_EXHAUSTIVE" || strings.Join(d.Missing, ",") != "B" || d.Symbol != "main.run" || d.Hint == "" {
		t.Fatalf("%+v", d)
	}
}
