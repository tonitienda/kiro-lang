package format

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
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
		switch decl := d.(type) {
		case ast.TypeDecl:
			fmt.Fprintf(&b, "type %s {\n", decl.Name)
			for _, f := range decl.Fields {
				fmt.Fprintf(&b, "  %s:%s\n", f.Name, f.Type)
			}
			b.WriteString("}")
		case ast.FuncDecl:
			fmt.Fprintf(&b, "fn %s(", decl.Name)
			for pi, p := range decl.Params {
				if pi > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%s:%s", p.Name, p.Type)
			}
			fmt.Fprintf(&b, ") -> %s =\n", decl.ReturnType)
			for _, line := range normalizeBody(decl.Body) {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
		if i < len(file.Decls)-1 {
			b.WriteString("\n\n")
		}
	}
	out := strings.TrimRight(b.String(), "\n") + "\n"
	return out
}

func normalizeBody(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return []string{""}
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
