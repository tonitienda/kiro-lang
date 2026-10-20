package lsp

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

func fileURI(path string) string { return (&url.URL{Scheme: "file", Path: path}).String() }
func reportedDiagnostics(responses []Response) map[string][]Diagnostic {
	out := map[string][]Diagnostic{}
	for _, response := range responses {
		params := response.Params.(map[string]any)
		out[params["uri"].(string)] = params["diagnostics"].([]Diagnostic)
	}
	return out
}
func TestWorkspaceUnsavedSemanticFeedback(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ki")
	helper := filepath.Join(dir, "helper.ki")
	mainSource := "mod main\nimport helper\nfn main()->i32 {return helper.value()}\n"
	helperSource := "mod helper\nfn value()->i32 {return 3}\n"
	for path, source := range map[string]string{entry: mainSource, helper: helperSource} {
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer()
	raw, _ := json.Marshal(map[string]any{"rootUri": fileURI(dir)})
	server.handle(Request{Method: "initialize", ID: 1, Params: raw})
	server.upsertDocument(fileURI(entry), mainSource)
	responses := server.upsertDocument(fileURI(helper), "mod helper\nfn value()->i32 {return \"bad\"}\n")
	diags := reportedDiagnostics(responses)[fileURI(helper)]
	if len(diags) != 1 || diags[0].Code != "KIRO_TYPE_MISMATCH" || diags[0].Range.Start.Line != 1 {
		t.Fatalf("%+v", diags)
	}
	data := diags[0].Data.(runtimekit.Diagnostic)
	if data.Symbol != "helper.value" || data.Expected != "i32" || data.Actual != "str" {
		t.Fatalf("%+v", data)
	}
	disk, _ := os.ReadFile(helper)
	if string(disk) != helperSource {
		t.Fatal("unsaved buffer wrote disk")
	}
	for uri, list := range reportedDiagnostics(server.upsertDocument(fileURI(helper), helperSource)) {
		if len(list) > 0 {
			t.Fatalf("stale diagnostics at %s: %+v", uri, list)
		}
	}
	server.upsertDocument(fileURI(helper), "mod helper\nfn value()->i32 {}\n")
	raw, _ = json.Marshal(map[string]any{"textDocument": map[string]string{"uri": fileURI(helper)}})
	for uri, list := range reportedDiagnostics(server.handle(Request{Method: "textDocument/didClose", Params: raw})) {
		if len(list) > 0 {
			t.Fatalf("close did not restore disk at %s: %+v", uri, list)
		}
	}
}
func TestLSPUnicodeCoordinatesAndCompletionContracts(t *testing.T) {
	server := NewServer()
	text := "mod main\nfn bad(value:str = \"😀\", wrong i32)->i32 {return 0}\n"
	diags := reportedDiagnostics(server.upsertDocument("file:///virtual/unicode.ki", text))["file:///virtual/unicode.ki"]
	line := strings.Split(text, "\n")[1]
	prefix := line[:strings.Index(line, "i32)")]
	expected := len(utf16.Encode([]rune(prefix)))
	if len(diags) != 1 || diags[0].Code != "KIRO_SYNTAX" || diags[0].Range.Start.Character != expected {
		t.Fatalf("want UTF16 column %d, got %+v", expected, diags)
	}
	source := "mod main\ntype Failure {variant Missing {key:str}}\nfn 足す()->i32 {return 0}\n"
	server.upsertDocument("file:///virtual/valid.ki", source)
	items := server.completion("file:///virtual/valid.ki").([]CompletionItem)
	seen := map[string]bool{}
	for i, item := range items {
		if i > 0 && items[i-1].Label > item.Label {
			t.Fatal("completion order is unstable")
		}
		seen[item.Label] = true
	}
	for _, name := range []string{"list.reduce", "partial", "variant", "Failure.Missing"} {
		if !seen[name] {
			t.Fatal("missing completion", name)
		}
	}
	for _, name := range []string{"if", "while", "defer"} {
		if seen[name] {
			t.Fatal("suggested unsupported syntax", name)
		}
	}
	if wordAt("😀 足す", Position{Line: 0, Character: 3}) != "足す" {
		t.Fatal("word lookup did not honor UTF16")
	}
}
