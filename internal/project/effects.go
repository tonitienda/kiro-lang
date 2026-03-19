package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/lexer"
)

var knownEffects = map[string]struct{}{
	"env": {}, "fs": {}, "io": {}, "log": {}, "net": {}, "panic": {}, "proc": {}, "time": {},
}

var stdlibCallEffects = map[string][]string{
	"cli.args":      {"proc"},
	"env.get":       {"env"},
	"env.get_or":    {"env"},
	"env.require":   {"env"},
	"fs.read_file":  {"fs"},
	"fs.write_file": {"fs"},
	"http.serve":    {"net"},
	"log.error":     {"log"},
	"log.info":      {"log"},
	"log.warn":      {"log"},
	"print":         {"io"},
	"println":       {"io"},
	"proc.exit":     {"proc"},
	"proc.run":      {"proc"},
	"time.now_unix": {"time"},
	"time.sleep_ms": {"time"},
}

type funcInfo struct {
	file    File
	decl    ast.FuncDecl
	effects map[string]struct{}
}

func validateEffects(p *Project) error {
	moduleFuncs := map[string]map[string]funcInfo{}
	nameFuncs := map[string][]funcInfo{}

	for _, file := range p.Files {
		for _, decl := range file.AST.Decls {
			fn, ok := decl.(ast.FuncDecl)
			if !ok {
				continue
			}
			effects, err := validateEffectSignature(file, fn)
			if err != nil {
				return err
			}
			if moduleFuncs[file.AST.Module] == nil {
				moduleFuncs[file.AST.Module] = map[string]funcInfo{}
			}
			info := funcInfo{file: file, decl: fn, effects: effects}
			moduleFuncs[file.AST.Module][fn.Name] = info
			nameFuncs[fn.Name] = append(nameFuncs[fn.Name], info)
		}
	}

	for _, file := range p.Files {
		for _, decl := range file.AST.Decls {
			fn, ok := decl.(ast.FuncDecl)
			if !ok {
				continue
			}
			for _, call := range collectCalls(fn.Body) {
				required := resolveCallEffects(file, fn, call, moduleFuncs, nameFuncs)
				if len(required) == 0 {
					continue
				}
				missing := missingEffects(fn, required)
				if len(missing) == 0 {
					continue
				}
				sort.Strings(missing)
				msg := fmt.Errorf("%d:%d: function %q calls %q which requires effect %q\nhint: add %q to the function signature", fn.Line, fn.Column, fn.Name, call, "!"+missing[0], "!"+missing[0])
				return withSourceLocation(file.Path, file.Src, msg)
			}
		}
	}

	return nil
}

func validateEffectSignature(file File, fn ast.FuncDecl) (map[string]struct{}, error) {
	effects := map[string]struct{}{}
	for _, effect := range fn.Effects {
		if _, ok := knownEffects[effect.Name]; !ok {
			known := sortedKnownEffects()
			msg := fmt.Errorf("%d:%d: unknown effect %q\nknown effects: %s", effect.Line, effect.Column, "!"+effect.Name, strings.Join(known, ", "))
			return nil, withSourceLocation(file.Path, file.Src, msg)
		}
		if _, ok := effects[effect.Name]; ok {
			msg := fmt.Errorf("%d:%d: duplicate effect %q in function signature", effect.Line, effect.Column, "!"+effect.Name)
			return nil, withSourceLocation(file.Path, file.Src, msg)
		}
		effects[effect.Name] = struct{}{}
	}
	return effects, nil
}

func sortedKnownEffects() []string {
	known := make([]string, 0, len(knownEffects))
	for effect := range knownEffects {
		known = append(known, "!"+effect)
	}
	sort.Strings(known)
	return known
}

func collectCalls(body string) []string {
	toks, err := lexer.Lex(body)
	if err != nil {
		return nil
	}
	var calls []string
	for i := 0; i < len(toks); i++ {
		if toks[i].Kind != lexer.TokenIdent {
			continue
		}
		parts := []string{toks[i].Text}
		j := i + 1
		for j+1 < len(toks) && toks[j].Text == "." && toks[j+1].Kind == lexer.TokenIdent {
			parts = append(parts, toks[j+1].Text)
			j += 2
		}
		if j < len(toks) && toks[j].Text == "(" {
			calls = append(calls, strings.Join(parts, "."))
			i = j - 1
		}
	}
	return calls
}

func resolveCallEffects(file File, fn ast.FuncDecl, call string, moduleFuncs map[string]map[string]funcInfo, nameFuncs map[string][]funcInfo) []string {
	if effects, ok := stdlibCallEffects[call]; ok {
		return effects
	}
	if strings.Contains(call, ".") {
		parts := strings.Split(call, ".")
		if len(parts) == 2 {
			if modFns, ok := moduleFuncs[parts[0]]; ok {
				if info, ok := modFns[parts[1]]; ok {
					return sortedEffectSet(info.effects)
				}
			}
			return unionNamedEffects(parts[1], nameFuncs)
		}
	}
	if modFns, ok := moduleFuncs[file.AST.Module]; ok {
		if info, ok := modFns[call]; ok {
			return sortedEffectSet(info.effects)
		}
	}
	return unionNamedEffects(call, nameFuncs)
}

func unionNamedEffects(name string, nameFuncs map[string][]funcInfo) []string {
	infos := nameFuncs[name]
	if len(infos) == 0 {
		return nil
	}
	merged := map[string]struct{}{}
	for _, info := range infos {
		for effect := range info.effects {
			merged[effect] = struct{}{}
		}
	}
	return sortedEffectSet(merged)
}

func sortedEffectSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for effect := range set {
		out = append(out, effect)
	}
	sort.Strings(out)
	return out
}

func missingEffects(fn ast.FuncDecl, required []string) []string {
	declared := map[string]struct{}{}
	for _, effect := range fn.Effects {
		declared[effect.Name] = struct{}{}
	}
	var missing []string
	for _, effect := range required {
		if _, ok := declared[effect]; !ok {
			missing = append(missing, effect)
		}
	}
	return missing
}

func importAlias(path string) string {
	base := filepath.Base(path)
	if base == "." || base == "/" {
		return path
	}
	return base
}
