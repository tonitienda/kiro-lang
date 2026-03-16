package parser

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/lexer"
)

type Parser struct {
	tokens []lexer.Token
	pos    int
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
		if t.Kind != lexer.TokenKeyword {
			return nil, fmt.Errorf("unexpected token %q at %d:%d", t.Text, t.Line, t.Column)
		}
		switch t.Text {
		case "import":
			p.next()
			imp, err := p.expect(lexer.TokenIdent)
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, imp.Text)
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
	return file, nil
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
	decl := ast.TypeDecl{Name: name.Text}
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
		typ, err := p.expect(lexer.TokenIdent)
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
		paramType, err := p.expect(lexer.TokenIdent)
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
	ret, err := p.expect(lexer.TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectText("="); err != nil {
		return nil, err
	}
	body := p.collectBody()
	return ast.FuncDecl{Name: name.Text, Params: params, ReturnType: ret.Text, Body: body}, nil
}

func (p *Parser) collectBody() string {
	var parts []string
	for !p.isEOF() {
		t := p.peek()
		if t.Kind == lexer.TokenKeyword && t.Column == 1 && (t.Text == "fn" || t.Text == "type" || t.Text == "import") {
			break
		}
		p.next()
		if t.Kind == lexer.TokenEOF {
			break
		}
		if t.Kind == lexer.TokenNewline {
			parts = append(parts, "\n")
			continue
		}
		if t.Kind == lexer.TokenString {
			parts = append(parts, fmt.Sprintf("\"%s\"", t.Text))
			continue
		}
		parts = append(parts, t.Text)
	}
	body := strings.TrimSpace(strings.Join(parts, " "))
	body = strings.ReplaceAll(body, "\n ", "\n")
	return body
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
