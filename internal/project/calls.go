package project

import (
	"fmt"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/lexer"
)

func validateCalls(p *Project) error {
	modules := map[string]map[string]ast.FuncDecl{}
	var methods []ast.FuncDecl
	for _, file := range p.Files {
		if modules[file.AST.Module] == nil {
			modules[file.AST.Module] = map[string]ast.FuncDecl{}
		}
		for _, declaration := range file.AST.Decls {
			if fn, ok := declaration.(ast.FuncDecl); ok {
				modules[file.AST.Module][fn.Name] = fn
				if fn.Receiver != nil {
					methods = append(methods, fn)
				}
			}
		}
	}
	for _, file := range p.Files {
		for _, declaration := range file.AST.Decls {
			fn, ok := declaration.(ast.FuncDecl)
			if !ok {
				continue
			}
			calls, err := runtimekit.InspectCalls(fn.Body)
			if err != nil {
				return err
			}
			locals, types := localCallNames(fn)
			for _, call := range calls {
				var parameters []string
				known := false
				parts := strings.Split(call.Target, ".")
				if len(parts) == 1 && !locals[parts[0]] {
					if target, ok := modules[file.AST.Module][parts[0]]; ok {
						parameters = declarationParameters(target, true)
						known = true
					} else {
						parameters, known = runtimekit.BuiltinParameters(call.Target)
					}
				} else if len(parts) == 2 {
					if !locals[parts[0]] {
						if call.Target == "list.fold" {
							return withSourceLocation(file.Path, file.Src, fmt.Errorf("%d:%d: list.fold was renamed to list.reduce; keep items, initial and reduce keys", fn.Line, fn.Column))
						}
						parameters, known = runtimekit.BuiltinParameters(call.Target)
						if target, ok := modules[parts[0]][parts[1]]; ok && !known {
							parameters = declarationParameters(target, true)
							known = true
						}
						if _, standard := stdlibModules[parts[0]]; standard && !known {
							return withSourceLocation(file.Path, file.Src, fmt.Errorf("%d:%d: unknown callable %q\nhint: use a supported standard-library signature", fn.Line, fn.Column, call.Target))
						}
					}
					if !known && types[parts[0]] != "" {
						for _, method := range methods {
							if method.Name == parts[1] && method.Receiver.Type == types[parts[0]] {
								parameters = declarationParameters(method, false)
								known = true
							}
						}
					}
				}
				// Higher-order/otherwise unresolved callables are validated against
				// their actual signature by the runtime, before evaluating arguments.
				if !known {
					continue
				}
				if _, err := runtimekit.CheckArgumentKeys(parameters, call.Keys, call.Partial); err != nil {
					d := runtimekit.DiagnosticOf(err, "KIRO_CHECK_FAILED")
					d.Message = fmt.Sprintf("function %s, call %s: %v", fn.Name, call.Target, err)
					d.Symbol = file.AST.Module + "." + fn.Name
					d.Target = call.Target
					d.Range = runtimekit.DeclarationRange(fn.Line, fn.Column, len([]rune(fn.Name)))
					d.LocationKind = "declaration"
					return withSourceLocation(file.Path, file.Src, &d)
				}
			}
		}
	}
	return nil
}

func declarationParameters(fn ast.FuncDecl, includeReceiver bool) []string {
	var parameters []string
	if includeReceiver && fn.Receiver != nil {
		parameters = append(parameters, fn.Receiver.Name)
	}
	for _, parameter := range fn.Params {
		parameters = append(parameters, parameter.Name)
	}
	return parameters
}

func localCallNames(fn ast.FuncDecl) (map[string]bool, map[string]string) {
	locals := map[string]bool{}
	types := map[string]string{}
	for _, parameter := range fn.Params {
		locals[parameter.Name] = true
		types[parameter.Name] = parameter.Type
	}
	if fn.Receiver != nil {
		locals[fn.Receiver.Name] = true
		types[fn.Receiver.Name] = fn.Receiver.Type
	}
	tokens, _ := lexer.Lex(fn.Body)
	for i := 0; i+3 < len(tokens); i++ {
		if tokens[i].Text != "let" && tokens[i].Text != "mut" {
			continue
		}
		if tokens[i+1].Kind != lexer.TokenIdent || tokens[i+2].Text != "=" {
			continue
		}
		name := tokens[i+1].Text
		locals[name] = true
		if i+4 < len(tokens) && tokens[i+3].Kind == lexer.TokenIdent && tokens[i+4].Text == "{" {
			types[name] = tokens[i+3].Text
		}
	}
	return locals, types
}
