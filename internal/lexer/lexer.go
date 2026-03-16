package lexer

import (
	"fmt"
	"unicode"
)

type TokenKind string

const (
	TokenEOF     TokenKind = "EOF"
	TokenNewline TokenKind = "NEWLINE"
	TokenIdent   TokenKind = "IDENT"
	TokenInt     TokenKind = "INT"
	TokenString  TokenKind = "STRING"
	TokenKeyword TokenKind = "KEYWORD"
	TokenSymbol  TokenKind = "SYMBOL"
)

type Token struct {
	Kind   TokenKind
	Text   string
	Line   int
	Column int
}

type Lexer struct {
	src  []rune
	pos  int
	line int
	col  int
	toks []Token
}

func Lex(src string) ([]Token, error) {
	l := &Lexer{src: []rune(src), line: 1, col: 1}
	for {
		if l.eof() {
			l.emit(TokenEOF, "")
			return l.toks, nil
		}
		ch := l.peek()
		switch {
		case ch == ' ' || ch == '\t' || ch == '\r':
			l.advance()
		case ch == '\n':
			l.emit(TokenNewline, "\n")
			l.advanceLine()
		case unicode.IsLetter(ch) || ch == '_':
			l.lexIdent()
		case unicode.IsDigit(ch):
			l.lexInt()
		case ch == '"':
			if err := l.lexString(); err != nil {
				return nil, err
			}
		default:
			l.emit(TokenSymbol, string(ch))
			l.advance()
		}
	}
}

func (l *Lexer) lexIdent() {
	start := l.mark()
	for !l.eof() {
		ch := l.peek()
		if !(unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_') {
			break
		}
		l.advance()
	}
	text := string(l.src[start:l.pos])
	kind := TokenIdent
	if isKeyword(text) {
		kind = TokenKeyword
	}
	l.emitAt(kind, text, l.line, l.col-(l.pos-start))
}

func (l *Lexer) lexInt() {
	start := l.mark()
	for !l.eof() && unicode.IsDigit(l.peek()) {
		l.advance()
	}
	text := string(l.src[start:l.pos])
	l.emitAt(TokenInt, text, l.line, l.col-(l.pos-start))
}

func (l *Lexer) lexString() error {
	line, col := l.line, l.col
	l.advance()
	start := l.mark()
	for !l.eof() && l.peek() != '"' {
		if l.peek() == '\n' {
			return fmt.Errorf("unterminated string at %d:%d", line, col)
		}
		l.advance()
	}
	if l.eof() {
		return fmt.Errorf("unterminated string at %d:%d", line, col)
	}
	text := string(l.src[start:l.pos])
	l.advance()
	l.emitAt(TokenString, text, line, col)
	return nil
}

func (l *Lexer) emit(kind TokenKind, text string) {
	l.toks = append(l.toks, Token{Kind: kind, Text: text, Line: l.line, Column: l.col})
}

func (l *Lexer) emitAt(kind TokenKind, text string, line, col int) {
	l.toks = append(l.toks, Token{Kind: kind, Text: text, Line: line, Column: col})
}

func (l *Lexer) advance() {
	l.pos++
	l.col++
}

func (l *Lexer) advanceLine() {
	l.pos++
	l.line++
	l.col = 1
}

func (l *Lexer) peek() rune { return l.src[l.pos] }
func (l *Lexer) mark() int  { return l.pos }
func (l *Lexer) eof() bool  { return l.pos >= len(l.src) }

func isKeyword(s string) bool {
	switch s {
	case "mod", "import", "type", "fn", "let", "mut", "if", "else", "when", "for", "in", "while", "break", "continue", "defer", "return":
		return true
	default:
		return false
	}
}
