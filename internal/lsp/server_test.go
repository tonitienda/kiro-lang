package lsp

import "testing"

const sample = `mod main

/// add docs
fn add(a:i32, b:i32) -> i32 {
  return a + b
}

fn main() -> i32 {
  return add(1, 2)
}
`

func TestDiagnostics(t *testing.T) {
	_, diags := buildDocState("file:///x.ki", "mod main\nfn main( -> i32 {}")
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestHoverDefinitionAndSymbols(t *testing.T) {
	s := NewServer()
	s.upsertDocument("file:///main.ki", sample)
	hover := s.hover("file:///main.ki", Position{Line: 8, Character: 10})
	if hover == nil {
		t.Fatalf("expected hover result")
	}
	def := s.definition("file:///main.ki", Position{Line: 8, Character: 10})
	locs, ok := def.([]Location)
	if !ok || len(locs) == 0 {
		t.Fatalf("expected definition location")
	}
	doc := s.docs["file:///main.ki"]
	if len(doc.SymbolItems) < 2 {
		t.Fatalf("expected document symbols")
	}
}

func TestFormattingAndCompletion(t *testing.T) {
	s := NewServer()
	s.upsertDocument("file:///main.ki", "mod main\nfn main()->i32{return 0}\n")
	edits := s.formatting("file:///main.ki").([]map[string]any)
	if len(edits) == 0 {
		t.Fatalf("expected formatting edit")
	}
	items := s.completion("file:///main.ki").([]CompletionItem)
	if len(items) == 0 {
		t.Fatalf("expected completion items")
	}
}
