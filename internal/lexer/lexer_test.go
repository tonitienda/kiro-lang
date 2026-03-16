package lexer

import "testing"

func TestLexBasic(t *testing.T) {
	src := "mod main\nfn main() -> i32 = 0\n"
	toks, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}
	if len(toks) == 0 {
		t.Fatal("expected tokens")
	}
	if toks[0].Text != "mod" || toks[1].Text != "main" {
		t.Fatalf("unexpected first tokens: %#v", toks[:2])
	}
}

func TestLexPhase3Keywords(t *testing.T) {
	src := "while break continue defer"
	toks, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}
	for i, tok := range toks[:4] {
		if tok.Kind != TokenKeyword {
			t.Fatalf("token[%d] kind = %s", i, tok.Kind)
		}
	}
}
