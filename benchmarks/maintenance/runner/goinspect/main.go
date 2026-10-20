// goinspect parses source without executing candidate code. Run with go run main.go ROOT.
package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Node struct {
	ID      string   `json:"id"`
	Module  string   `json:"module"`
	Returns []string `json:"returns,omitempty"`
}
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

func importModule(path string) string {
	if strings.HasPrefix(path, "benchmark.local/checkout/internal/") {
		return strings.TrimPrefix(path, "benchmark.local/checkout/internal/")
	}
	if strings.HasPrefix(path, "benchmark.local/checkout/") {
		return strings.TrimPrefix(path, "benchmark.local/checkout/")
	}
	return filepath.Base(path)
}

func main() {
	root := os.Args[1]
	context := build.Default
	if len(os.Args) > 2 {
		context.GOOS = os.Args[2]
	}
	if len(os.Args) > 3 {
		context.GOARCH = os.Args[3]
	}
	nodes := []Node{}
	edges := []Edge{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		active, err := context.MatchFile(filepath.Dir(path), filepath.Base(path))
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, filepath.Dir(path))
		module := strings.TrimPrefix(filepath.ToSlash(relative), "internal/")
		if relative != "app" && !strings.HasPrefix(relative, "internal"+string(filepath.Separator)) {
			return nil
		}
		imports := map[string]string{}
		selectorNames := map[*ast.Ident]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				selectorNames[s.Sel] = true
			}
			return true
		})
		for _, imp := range f.Imports {
			value := strings.Trim(imp.Path.Value, "\"")
			name := filepath.Base(value)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imports[name] = value
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						nodes = append(nodes, Node{ID: module + "." + s.Name.Name, Module: module})
					case *ast.ValueSpec:
						for _, name := range s.Names {
							nodes = append(nodes, Node{ID: module + "." + name.Name, Module: module})
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv != nil {
					continue
				}
				from := module + "." + d.Name.Name
				returns := []string{}
				if d.Type.Results != nil {
					for _, result := range d.Type.Results.List {
						var out bytes.Buffer
						format.Node(&out, token.NewFileSet(), result.Type)
						count := len(result.Names)
						if count == 0 {
							count = 1
						}
						for i := 0; i < count; i++ {
							returns = append(returns, out.String())
						}
					}
				}
				nodes = append(nodes, Node{ID: from, Module: module, Returns: returns})
				ast.Inspect(d.Body, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.CallExpr:
						switch fun := x.Fun.(type) {
						case *ast.Ident:
							if fun.Obj == nil || fun.Obj.Kind == ast.Fun {
								edges = append(edges, Edge{from, module + "." + fun.Name, "call"})
							}
						case *ast.SelectorExpr:
							if ident, ok := fun.X.(*ast.Ident); ok && ident.Obj == nil {
								if path, ok := imports[ident.Name]; ok {
									edges = append(edges, Edge{from, importModule(path) + "." + fun.Sel.Name, "call"})
								}
							}
						}
					case *ast.Ident:
						if (x.Name == "ErrPaymentDeclined" || x.Name == "PaymentDeclined") && !selectorNames[x] && (x.Obj == nil || f.Scope.Objects[x.Name] == x.Obj) {
							edges = append(edges, Edge{from, module + "." + x.Name, "reference"})
						}
					}
					return true
				})
			}
		}
		// Imports count as dependencies even if no function calls them (e.g. embedded types).
		for _, path := range imports {
			if strings.HasPrefix(path, "benchmark.local/checkout/") {
				edges = append(edges, Edge{module + ".__imports", importModule(path) + ".__package", "dependency"})
				nodes = append(nodes, Node{ID: module + ".__imports", Module: module}, Node{ID: importModule(path) + ".__package", Module: importModule(path)})
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"nodes": nodes, "edges": edges})
}
