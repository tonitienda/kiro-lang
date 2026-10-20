package format

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/lexer"
	"github.com/kiro-lang/kiro/internal/parser"
)

func Source(src string) (string, error) {
	file, err := parser.Parse(src)
	if err != nil {
		return "", err
	}
	return Print(file), nil
}

func Print(file *ast.File) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mod %s\n", file.Module)
	if len(file.Imports) > 0 || len(file.Decls) > 0 {
		b.WriteString("\n")
	}
	for _, imp := range file.Imports {
		fmt.Fprintf(&b, "import %s\n", imp)
	}
	if len(file.Imports) > 0 && len(file.Decls) > 0 {
		b.WriteString("\n")
	}
	for i, d := range file.Decls {
		printDoc(&b, d.GetDoc())
		switch decl := d.(type) {
		case ast.ConstDecl:
			val := decl.Value
			if decl.ValueKind == "STRING" {
				val = fmt.Sprintf("\"%s\"", decl.Value)
			}
			fmt.Fprintf(&b, "const %s = %s", decl.Name, val)
		case ast.TypeDecl:
			fmt.Fprintf(&b, "type %s {\n", decl.Name)
			for _, f := range decl.Fields {
				fmt.Fprintf(&b, "  %s:%s\n", f.Name, f.Type)
			}
			for _, v := range decl.Variants {
				fmt.Fprintf(&b, "  variant %s {\n", v.Name)
				for _, f := range v.Fields {
					fmt.Fprintf(&b, "    %s:%s\n", f.Name, f.Type)
				}
				b.WriteString("  }\n")
			}
			b.WriteString("}")
		case ast.FuncDecl:
			b.WriteString(decl.Signature())
			b.WriteString(" {\n")
			for _, line := range normalizeBody(decl.Body) {
				fmt.Fprintf(&b, "  %s\n", line)
			}
			b.WriteString("}")
		}
		if i < len(file.Decls)-1 {
			b.WriteString("\n\n")
		}
	}
	out := strings.TrimRight(b.String(), "\n") + "\n"
	return out
}

func printDoc(b *strings.Builder, lines []string) {
	for _, line := range lines {
		fmt.Fprintf(b, "/// %s\n", strings.TrimSpace(line))
	}
}

func normalizeBody(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return []string{""}
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := normalizeBodyLine(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func normalizeBodyLine(line string) string {
	tokens, err := lexer.Lex(line)
	if err != nil {
		// Source() already lexed the input. Keep a manually constructed AST's
		// unknown text intact rather than rewriting its literal contents.
		return strings.TrimSpace(line)
	}
	var parts, plain []string
	flush := func() {
		if len(plain) == 0 {
			return
		}
		text := strings.Join(plain, " ")
		text = strings.ReplaceAll(text, "= =", "==")
		text = strings.ReplaceAll(text, "= >", "=>")
		text = strings.ReplaceAll(text, "- >", "->")
		text = strings.ReplaceAll(text, " . ", ".")
		parts = append(parts, text)
		plain = nil
	}
	for _, token := range tokens {
		switch token.Kind {
		case lexer.TokenEOF, lexer.TokenNewline:
			continue
		case lexer.TokenString:
			flush()
			parts = append(parts, "\""+token.Text+"\"")
		default:
			plain = append(plain, token.Text)
		}
	}
	flush()
	return strings.Join(parts, " ")
}
