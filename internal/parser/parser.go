package parser

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/lexer"
)

type Parser struct {
	tokens     []lexer.Token
	pos        int
	pendingDoc []string
}

func Parse(src string) (*ast.File, error) {
	toks, err := lexer.Lex(src)
	if err != nil {
		return nil, err
	}
	p := &Parser{tokens: toks}
	return p.parseFile()
}

func (p *Parser) parseFile() (*ast.File, error) {
	file := &ast.File{}
	if err := p.expectKeyword("mod"); err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return nil, err
	}
	file.Module = name.Text

	for !p.isEOF() {
		p.skipNewlines()
		if p.isEOF() {
			break
		}
		t := p.peek()
		if t.Kind == lexer.TokenDocComment {
			p.pendingDoc = append(p.pendingDoc, t.Text)
			p.next()
			continue
		}
		if t.Kind != lexer.TokenKeyword {
			return nil, fmt.Errorf("unexpected token %q at %d:%d", t.Text, t.Line, t.Column)
		}
		switch t.Text {
		case "import":
			if len(p.pendingDoc) > 0 {
				return nil, fmt.Errorf("doc comments may only appear on declarations at %d:%d", t.Line, t.Column)
			}
			p.next()
			imp, err := p.parseImportPath()
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, imp)
		case "const":
			d, err := p.parseConstDecl()
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, d)
		case "type":
			d, err := p.parseTypeDecl()
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, d)
		case "fn":
			d, err := p.parseFuncDecl()
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, d)
		default:
			return nil, fmt.Errorf("unsupported keyword %q at %d:%d", t.Text, t.Line, t.Column)
		}
	}
	if len(p.pendingDoc) > 0 {
		return nil, fmt.Errorf("doc comment not attached to declaration")
	}
	return file, nil
}

func (p *Parser) parseConstDecl() (ast.Decl, error) {
	p.next()
	name, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectText("="); err != nil {
		return nil, err
	}
	val, err := p.expectConstValue()
	if err != nil {
		return nil, err
	}
	return ast.ConstDecl{Doc: p.takePendingDoc(), Name: name.Text, Value: val.Text, ValueKind: string(val.Kind)}, nil
}

func (p *Parser) expectConstValue() (lexer.Token, error) {
	t := p.next()
	if t.Kind != lexer.TokenString && t.Kind != lexer.TokenInt && t.Kind != lexer.TokenIdent {
		return lexer.Token{}, fmt.Errorf("expected const value at %d:%d", t.Line, t.Column)
	}
	return t, nil
}

func (p *Parser) parseImportPath() (string, error) {
	seg, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return "", err
	}
	parts := []string{seg.Text}
	for p.peekText("/") {
		p.next()
		nextSeg, err := p.expect(lexer.TokenIdent)
		if err != nil {
			return "", err
		}
		parts = append(parts, nextSeg.Text)
	}
	return strings.Join(parts, "/"), nil
}

func (p *Parser) parseTypeDecl() (ast.Decl, error) {
	p.next()
	name, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectText("{"); err != nil {
		return nil, err
	}
	decl := ast.TypeDecl{Doc: p.takePendingDoc(), Name: name.Text}
	for {
		p.skipNewlines()
		if p.peekText("}") {
			p.next()
			break
		}
		fname, err := p.expect(lexer.TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expectText(":"); err != nil {
			return nil, err
		}
		typ, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		decl.Fields = append(decl.Fields, ast.Field{Name: fname.Text, Type: typ.Text})
		p.skipNewlines()
	}
	return decl, nil
}

func (p *Parser) parseFuncDecl() (ast.Decl, error) {
	p.next()
	var receiver *ast.Param
	if p.peekText("(") {
		r, err := p.parseReceiver()
		if err != nil {
			return nil, err
		}
		receiver = &r
	}
	name, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectText("("); err != nil {
		return nil, err
	}
	var params []ast.Param
	for !p.peekText(")") {
		paramName, err := p.expect(lexer.TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expectText(":"); err != nil {
			return nil, err
		}
		paramType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		params = append(params, ast.Param{Name: paramName.Text, Type: paramType.Text})
		if p.peekText(",") {
			p.next()
		}
	}
	p.next()
	if _, err := p.expectText("-"); err != nil {
		return nil, err
	}
	if _, err := p.expectText(">"); err != nil {
		return nil, err
	}
	ret, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	effects, err := p.parseEffects()
	if err != nil {
		return nil, err
	}

	decl := ast.FuncDecl{
		Doc:        p.takePendingDoc(),
		Name:       name.Text,
		Receiver:   receiver,
		Params:     params,
		ReturnType: ret.Text,
		Effects:    effects,
		Line:       name.Line,
		Column:     name.Column,
	}
	if p.peekText("=") {
		eq := p.next()
		return nil, fmt.Errorf("%d:%d: expression-bodied functions were removed\nhint: replace \"=\" with \"{\" and use an explicit return", eq.Line, eq.Column)
	}
	if p.peekText("{") {
		p.next()
		body, err := p.collectBlockBody()
		if err != nil {
			return nil, err
		}
		decl.BlockBody = true
		decl.Body = body
		return decl, nil
	}
	return nil, fmt.Errorf("expected \"{\" to start function body at %d:%d", p.peek().Line, p.peek().Column)
}

func (p *Parser) parseEffects() ([]ast.EffectDecl, error) {
	var effects []ast.EffectDecl
	for p.peekText("!") {
		bang := p.next()
		name, err := p.expect(lexer.TokenIdent)
		if err != nil {
			return nil, err
		}
		effects = append(effects, ast.EffectDecl{Name: name.Text, Line: bang.Line, Column: bang.Column})
	}
	return effects, nil
}

func (p *Parser) parseReceiver() (ast.Param, error) {
	if _, err := p.expectText("("); err != nil {
		return ast.Param{}, err
	}
	name, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return ast.Param{}, err
	}
	if _, err := p.expectText(":"); err != nil {
		return ast.Param{}, err
	}
	typ, err := p.parseTypeRef()
	if err != nil {
		return ast.Param{}, err
	}
	if _, err := p.expectText(")"); err != nil {
		return ast.Param{}, err
	}
	return ast.Param{Name: name.Text, Type: typ.Text}, nil
}

func (p *Parser) collectBlockBody() (string, error) {
	depth := 1
	var lines []string
	var current []string
	for !p.isEOF() {
		t := p.next()
		if t.Kind == lexer.TokenSymbol {
			if t.Text == "{" {
				depth++
			}
			if t.Text == "}" {
				depth--
				if depth == 0 {
					if len(current) > 0 {
						lines = append(lines, strings.Join(current, " "))
					}
					return strings.TrimSpace(strings.Join(lines, "\n")), nil
				}
			}
		}
		if t.Kind == lexer.TokenNewline {
			line := strings.TrimSpace(strings.Join(current, " "))
			if line != "" {
				lines = append(lines, line)
			}
			current = nil
			continue
		}
		current = append(current, renderToken(t))
	}
	return "", fmt.Errorf("unterminated block body")
}

func (p *Parser) parseTypeRef() (lexer.Token, error) {
	optional := ""
	startCol := 0
	if p.peekText("?") {
		q := p.next()
		optional = q.Text
		startCol = q.Column
	}
	base, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return lexer.Token{}, err
	}
	parts := []string{base.Text}
	for p.peekText(".") {
		p.next()
		part, err := p.expect(lexer.TokenIdent)
		if err != nil {
			return lexer.Token{}, err
		}
		parts = append(parts, part.Text)
	}
	text := strings.Join(parts, ".")
	if p.peekText("[") {
		parts := []string{text, "["}
		p.next()
		depth := 1
		for depth > 0 {
			t := p.next()
			if t.Kind == lexer.TokenEOF {
				return lexer.Token{}, fmt.Errorf("unterminated type reference at %d:%d", base.Line, base.Column)
			}
			parts = append(parts, renderToken(t))
			if t.Kind == lexer.TokenSymbol {
				if t.Text == "[" {
					depth++
				} else if t.Text == "]" {
					depth--
				}
			}
		}
		text = strings.Join(parts, "")
	}
	base.Text = optional + text
	if optional != "" {
		base.Column = startCol
	}
	return base, nil
}

func renderToken(t lexer.Token) string {
	if t.Kind == lexer.TokenString {
		return fmt.Sprintf("\"%s\"", t.Text)
	}
	return t.Text
}

func (p *Parser) takePendingDoc() []string {
	if len(p.pendingDoc) == 0 {
		return nil
	}
	out := make([]string, len(p.pendingDoc))
	copy(out, p.pendingDoc)
	p.pendingDoc = nil
	return out
}

func (p *Parser) expectKeyword(k string) error {
	t := p.next()
	if t.Kind != lexer.TokenKeyword || t.Text != k {
		return fmt.Errorf("expected %q at %d:%d", k, t.Line, t.Column)
	}
	return nil
}

func (p *Parser) expect(kind lexer.TokenKind) (lexer.Token, error) {
	t := p.next()
	if t.Kind != kind {
		return lexer.Token{}, fmt.Errorf("expected %s at %d:%d", kind, t.Line, t.Column)
	}
	return t, nil
}

func (p *Parser) expectText(s string) (lexer.Token, error) {
	t := p.next()
	if t.Text != s {
		return lexer.Token{}, fmt.Errorf("expected %q at %d:%d", s, t.Line, t.Column)
	}
	return t, nil
}

func (p *Parser) skipNewlines() {
	for p.peek().Kind == lexer.TokenNewline {
		p.next()
	}
}

func (p *Parser) peekText(s string) bool { return p.peek().Text == s }
func (p *Parser) isEOF() bool            { return p.peek().Kind == lexer.TokenEOF }
func (p *Parser) peek() lexer.Token      { return p.tokens[p.pos] }

func (p *Parser) next() lexer.Token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}
