package runtimekit

import (
	"fmt"
	"strconv"
)

type stmt interface{}
type expr interface{}

type letStmt struct {
	Name  string
	Value expr
}

type assignStmt struct {
	Name  string
	Value expr
}

type returnStmt struct{ Value expr }
type exprStmt struct{ Value expr }
type groupStmt struct{ Body []stmt }
type whenStmt struct {
	Value expr
	Cases []whenCase
}

type whenCase struct {
	Wildcard bool
	Pattern  expr
	Expr     expr
	Body     []stmt
}

type intExpr struct{ Value int }
type stringExpr struct{ Value string }
type nilExpr struct{}
type identExpr struct{ Name string }
type selectorExpr struct {
	Left expr
	Name string
}
type callExpr struct {
	Callee   expr
	TypeArgs []string
	Args     []expr
}
type binaryExpr struct {
	Left  expr
	Op    string
	Right expr
}
type structExpr struct {
	TypeName string
	Fields   []structFieldExpr
}
type structFieldExpr struct {
	Name  string
	Value expr
}
type unwrapExpr struct{ Value expr }
type whenExpr struct {
	Value expr
	Cases []whenCase
}
type sequenceExpr struct{ Items []expr }
type spawnExpr struct{ Value expr }
type awaitExpr struct{ Value expr }
type listExpr struct{ Items []expr }

type bodyParser struct {
	tokens []Token
	pos    int
}

func newBodyParser(src string) (*bodyParser, error) {
	toks, err := Lex(src)
	if err != nil {
		return nil, err
	}
	return &bodyParser{tokens: toks}, nil
}

func (p *bodyParser) parseStatementsUntil(stopText string) ([]stmt, error) {
	var stmts []stmt
	for p.peek().Kind != TokenEOF {
		p.skipNewlines()
		if p.peekText(stopText) || p.peek().Kind == TokenEOF {
			break
		}
		st, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, st)
		p.skipNewlines()
	}
	return stmts, nil
}

func (p *bodyParser) parseStatement() (stmt, error) {
	if p.peekIsKeyword("let") || p.peekIsKeyword("mut") {
		p.next()
		name, err := p.expect(TokenIdent, "")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenSymbol, "="); err != nil {
			return nil, err
		}
		value, err := p.parseSequenceExprUntil(TokenNewline, TokenEOF)
		if err != nil {
			return nil, err
		}
		return &letStmt{Name: name.Text, Value: value}, nil
	}
	if p.peekIsKeyword("return") {
		p.next()
		value, err := p.parseSequenceExprUntil(TokenNewline, TokenEOF)
		if err != nil {
			return nil, err
		}
		return &returnStmt{Value: value}, nil
	}
	if p.peekIsKeyword("group") {
		p.next()
		if _, err := p.expect(TokenSymbol, "{"); err != nil {
			return nil, err
		}
		body, err := p.parseStatementsUntil("}")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenSymbol, "}"); err != nil {
			return nil, err
		}
		return &groupStmt{Body: body}, nil
	}
	if p.peekIsKeyword("when") {
		w, err := p.parseWhen()
		if err != nil {
			return nil, err
		}
		return &whenStmt{Value: w.Value, Cases: w.Cases}, nil
	}
	if p.peek().Kind == TokenIdent && p.peekNext().Text == "=" {
		name := p.next().Text
		p.next()
		value, err := p.parseSequenceExprUntil(TokenNewline, TokenEOF)
		if err != nil {
			return nil, err
		}
		return &assignStmt{Name: name, Value: value}, nil
	}
	value, err := p.parseSequenceExprUntil(TokenNewline, TokenEOF)
	if err != nil {
		return nil, err
	}
	return &exprStmt{Value: value}, nil
}

func (p *bodyParser) parseWhen() (*whenExpr, error) {
	if _, err := p.expect(TokenKeyword, "when"); err != nil {
		return nil, err
	}
	value, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	p.skipNewlines()
	var cases []whenCase
	for !p.peekText("}") && p.peek().Kind != TokenEOF {
		if p.peek().Kind == TokenNewline {
			p.next()
			continue
		}
		c, err := p.parseWhenCase()
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
		p.skipNewlines()
	}
	return &whenExpr{Value: value, Cases: cases}, nil
}

func (p *bodyParser) parseWhenCase() (whenCase, error) {
	wildcard := false
	var pattern expr
	if p.peekText("_") {
		wildcard = true
		p.next()
	} else {
		var err error
		pattern, err = p.parseExpr(0)
		if err != nil {
			return whenCase{}, err
		}
	}
	if _, err := p.expect(TokenSymbol, "="); err != nil {
		return whenCase{}, err
	}
	if _, err := p.expect(TokenSymbol, ">"); err != nil {
		return whenCase{}, err
	}
	if p.peekText("{") {
		p.next()
		body, err := p.parseStatementsUntil("}")
		if err != nil {
			return whenCase{}, err
		}
		if _, err := p.expect(TokenSymbol, "}"); err != nil {
			return whenCase{}, err
		}
		return whenCase{Wildcard: wildcard, Pattern: pattern, Body: body}, nil
	}
	ex, err := p.parseSequenceExprUntil(TokenNewline, TokenEOF)
	if err != nil {
		return whenCase{}, err
	}
	return whenCase{Wildcard: wildcard, Pattern: pattern, Expr: ex}, nil
}

func (p *bodyParser) parseSequenceExprUntil(stops ...TokenKind) (expr, error) {
	items := []expr{}
	for {
		p.skipNewlines()
		if p.peek().Kind == TokenEOF || p.peekKindIn(stops...) || p.peekText("}") {
			break
		}
		ex, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		items = append(items, ex)
		if p.peek().Kind == TokenNewline || p.peekKindIn(stops...) || p.peekText("}") || p.peek().Kind == TokenEOF {
			break
		}
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &sequenceExpr{Items: items}, nil
}

func (p *bodyParser) parseExpr(minPrec int) (expr, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		if tok.Kind == TokenNewline || tok.Kind == TokenEOF || tok.Text == ")" || tok.Text == "}" || tok.Text == "]" || tok.Text == "," || tok.Text == ":" {
			break
		}
		prec := precedence(tok)
		if prec < minPrec {
			break
		}
		if tok.Text == "?" {
			p.next()
			left = &unwrapExpr{Value: left}
			continue
		}
		if tok.Text == "." {
			p.next()
			name, err := p.expect(TokenIdent, "")
			if err != nil {
				return nil, err
			}
			left = &selectorExpr{Left: left, Name: name.Text}
			continue
		}
		if tok.Text == "(" || tok.Text == "[" {
			left, err = p.finishCall(left)
			if err != nil {
				return nil, err
			}
			continue
		}
		if tok.Text == "{" {
			ident, ok := left.(*identExpr)
			if !ok {
				break
			}
			left, err = p.finishStructLiteral(ident.Name)
			if err != nil {
				return nil, err
			}
			continue
		}
		op := p.next().Text
		right, err := p.parseExpr(prec + 1)
		if err != nil {
			return nil, err
		}
		left = &binaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *bodyParser) parsePrefix() (expr, error) {
	tok := p.next()
	switch tok.Kind {
	case TokenInt:
		n, err := strconv.Atoi(tok.Text)
		if err != nil {
			return nil, err
		}
		return &intExpr{Value: n}, nil
	case TokenString:
		return &stringExpr{Value: tok.Text}, nil
	case TokenIdent:
		return &identExpr{Name: tok.Text}, nil
	case TokenKeyword:
		switch tok.Text {
		case "nil":
			return &nilExpr{}, nil
		case "when":
			p.pos--
			return p.parseWhen()
		case "spawn":
			ex, err := p.parseExpr(100)
			if err != nil {
				return nil, err
			}
			return &spawnExpr{Value: ex}, nil
		case "await":
			ex, err := p.parseExpr(100)
			if err != nil {
				return nil, err
			}
			return &awaitExpr{Value: ex}, nil
		default:
			return &identExpr{Name: tok.Text}, nil
		}
	case TokenSymbol:
		switch tok.Text {
		case "(":
			ex, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenSymbol, ")"); err != nil {
				return nil, err
			}
			return ex, nil
		case "[":
			var items []expr
			for !p.peekText("]") {
				item, err := p.parseExpr(0)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
				if p.peekText(",") {
					p.next()
				}
			}
			if _, err := p.expect(TokenSymbol, "]"); err != nil {
				return nil, err
			}
			return &listExpr{Items: items}, nil
		}
	}
	return nil, fmt.Errorf("unexpected token %q", tok.Text)
}

func (p *bodyParser) finishCall(callee expr) (expr, error) {
	var typeArgs []string
	if p.peekText("[") {
		p.next()
		for !p.peekText("]") {
			arg, err := p.expect(TokenIdent, "")
			if err != nil {
				return nil, err
			}
			typeArgs = append(typeArgs, arg.Text)
			if p.peekText(",") {
				p.next()
			}
		}
		if _, err := p.expect(TokenSymbol, "]"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(TokenSymbol, "("); err != nil {
		return nil, err
	}
	var args []expr
	for !p.peekText(")") {
		arg, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peekText(",") {
			p.next()
		}
	}
	if _, err := p.expect(TokenSymbol, ")"); err != nil {
		return nil, err
	}
	return &callExpr{Callee: callee, TypeArgs: typeArgs, Args: args}, nil
}

func (p *bodyParser) finishStructLiteral(typeName string) (expr, error) {
	if _, err := p.expect(TokenSymbol, "{"); err != nil {
		return nil, err
	}
	var fields []structFieldExpr
	for !p.peekText("}") {
		name, err := p.expect(TokenIdent, "")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenSymbol, ":"); err != nil {
			return nil, err
		}
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		fields = append(fields, structFieldExpr{Name: name.Text, Value: val})
		if p.peekText(",") {
			p.next()
		}
	}
	if _, err := p.expect(TokenSymbol, "}"); err != nil {
		return nil, err
	}
	return &structExpr{TypeName: typeName, Fields: fields}, nil
}

func precedence(tok Token) int {
	switch tok.Text {
	case "?", ".", "(", "[", "{":
		return 90
	case "+":
		return 10
	default:
		return -1
	}
}

func (p *bodyParser) skipNewlines() {
	for p.peek().Kind == TokenNewline {
		p.next()
	}
}

func (p *bodyParser) peek() Token { return p.tokens[p.pos] }

func (p *bodyParser) peekNext() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Kind: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *bodyParser) next() Token {
	tok := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return tok
}

func (p *bodyParser) expect(kind TokenKind, text string) (Token, error) {
	tok := p.next()
	if tok.Kind != kind {
		return Token{}, fmt.Errorf("expected %s, got %s", kind, tok.Kind)
	}
	if text != "" && tok.Text != text {
		return Token{}, fmt.Errorf("expected %q, got %q", text, tok.Text)
	}
	return tok, nil
}

func (p *bodyParser) peekText(text string) bool { return p.peek().Text == text }
func (p *bodyParser) peekIsKeyword(text string) bool {
	return p.peek().Kind == TokenKeyword && p.peek().Text == text
}
func (p *bodyParser) peekKindIn(kinds ...TokenKind) bool {
	for _, kind := range kinds {
		if p.peek().Kind == kind {
			return true
		}
	}
	return false
}
