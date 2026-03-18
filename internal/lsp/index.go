package lsp

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/lexer"
	"github.com/kiro-lang/kiro/internal/parser"
)

type Symbol struct {
	Name  string
	Kind  int
	Line  int
	Col   int
	Sig   string
	Doc   string
	Range Range
}

type DocState struct {
	URI         string
	Text        string
	Symbols     map[string]Symbol
	SymbolItems []DocumentSymbol
}

func buildDocState(uri, text string) (*DocState, []Diagnostic) {
	d := &DocState{URI: uri, Text: text, Symbols: map[string]Symbol{}}
	f, err := parser.Parse(text)
	if err != nil {
		return d, []Diagnostic{diagnosticFromError(err)}
	}
	toks, err := lexer.Lex(text)
	if err != nil {
		return d, []Diagnostic{diagnosticFromError(err)}
	}
	pos := declPositions(toks)
	for _, decl := range f.Decls {
		s := symbolFromDecl(decl, pos)
		if s.Name == "" {
			continue
		}
		d.Symbols[s.Name] = s
		d.SymbolItems = append(d.SymbolItems, DocumentSymbol{
			Name:           s.Name,
			Kind:           s.Kind,
			Range:          s.Range,
			SelectionRange: s.Range,
		})
	}
	moduleLine, moduleCol := modulePosition(toks)
	if moduleLine > 0 {
		d.SymbolItems = append([]DocumentSymbol{{
			Name:           "module " + f.Module,
			Kind:           2,
			Range:          toRange(moduleLine, moduleCol, len(f.Module)+4),
			SelectionRange: toRange(moduleLine, moduleCol, len(f.Module)+4),
		}}, d.SymbolItems...)
	}
	return d, nil
}

func symbolFromDecl(d ast.Decl, pos map[string]Position) Symbol {
	s := Symbol{}
	switch v := d.(type) {
	case ast.ConstDecl:
		s.Name = v.Name
		s.Kind = 14
		s.Sig = fmt.Sprintf("const %s = %s", v.Name, v.Value)
		s.Doc = strings.Join(v.Doc, "\n")
	case ast.TypeDecl:
		s.Name = v.Name
		s.Kind = 5
		s.Sig = "type " + v.Name
		s.Doc = strings.Join(v.Doc, "\n")
	case ast.FuncDecl:
		s.Name = v.Name
		s.Kind = 6
		s.Sig = v.Signature()
		s.Doc = strings.Join(v.Doc, "\n")
	}
	if p, ok := pos[s.Name]; ok {
		s.Line, s.Col = p.Line, p.Character
		s.Range = Range{Start: p, End: Position{Line: p.Line, Character: p.Character + len(s.Name)}}
	}
	return s
}

func modulePosition(toks []lexer.Token) (int, int) {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind == lexer.TokenKeyword && toks[i].Text == "mod" && toks[i+1].Kind == lexer.TokenIdent {
			return toks[i].Line - 1, toks[i].Column - 1
		}
	}
	return 0, 0
}

func declPositions(toks []lexer.Token) map[string]Position {
	out := map[string]Position{}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.Kind != lexer.TokenKeyword {
			continue
		}
		switch t.Text {
		case "const", "type":
			if i+1 < len(toks) && toks[i+1].Kind == lexer.TokenIdent {
				n := toks[i+1]
				out[n.Text] = Position{Line: n.Line - 1, Character: n.Column - 1}
			}
		case "fn":
			j := i + 1
			if j < len(toks) && toks[j].Text == "(" {
				for j < len(toks) && toks[j].Text != ")" {
					j++
				}
				j++
			}
			if j < len(toks) && toks[j].Kind == lexer.TokenIdent {
				n := toks[j]
				out[n.Text] = Position{Line: n.Line - 1, Character: n.Column - 1}
			}
		}
	}
	return out
}

func toRange(line, col, width int) Range {
	return Range{Start: Position{Line: line, Character: col}, End: Position{Line: line, Character: col + width}}
}

func diagnosticFromError(err error) Diagnostic {
	msg := err.Error()
	line, col := 0, 0
	_, _ = fmt.Sscanf(msg, "%*[^0-9]%d:%d", &line, &col)
	if line > 0 {
		line--
		col--
	}
	return Diagnostic{
		Range:    Range{Start: Position{Line: line, Character: max(col, 0)}, End: Position{Line: line, Character: max(col+1, 1)}},
		Severity: 1,
		Source:   "kiro",
		Message:  msg,
	}
}
