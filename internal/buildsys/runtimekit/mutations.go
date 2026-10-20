package runtimekit

import (
	"fmt"
	"strconv"
	"strings"
)

type Mutation struct {
	ID         string `json:"id"`
	Symbol     string `json:"symbol"`
	Operator   string `json:"operator"`
	Before     string `json:"before"`
	After      string `json:"after"`
	BodyLine   int    `json:"body_line"`
	BodyColumn int    `json:"body_column"`
	Body       string `json:"-"`
}

// GenerateMutations uses executable tokens, never string/comment text. Selection
// is source-ordered and bounded; semantic validation is a separate classification.
func GenerateMutations(spec ProgramSpec, symbol string, limit int) ([]Mutation, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("mutation limit must be positive")
	}
	var selected *FuncSpec
	for _, mod := range spec.Modules {
		for _, fn := range mod.Funcs {
			if FunctionID(fn) == symbol {
				copy := fn
				selected = &copy
			}
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("unknown mutation target %s", symbol)
	}
	if strings.HasPrefix(selected.Name, "test_") {
		return nil, fmt.Errorf("target a production function, not a test")
	}
	tokens, err := Lex(selected.Body)
	if err != nil {
		return nil, err
	}
	runes := []rune(selected.Body)
	offset := func(token Token) int {
		line, column := 1, 1
		for i, r := range runes {
			if line == token.Line && column == token.Column {
				return i
			}
			if r == '\n' {
				line++
				column = 1
			} else {
				column++
			}
		}
		return len(runes)
	}
	out := []Mutation{}
	for i, token := range tokens {
		if len(out) >= limit {
			break
		}
		if token.Kind != TokenSymbol && token.Kind != TokenIdent && token.Kind != TokenKeyword {
			continue
		}
		if token.Text == ">" && i > 0 && tokens[i-1].Text == "=" {
			continue
		}
		before, after := token.Text, ""
		end := offset(token) + len([]rune(token.Text))
		if (before == "=" || before == "!" || before == "<" || before == ">") && i+1 < len(tokens) && tokens[i+1].Text == "=" {
			before += "="
			end = offset(tokens[i+1]) + 1
			i++
		}
		switch before {
		case "==":
			after = "!="
		case "!=":
			after = "=="
		case "<":
			after = "<="
		case "<=":
			after = "<"
		case ">":
			after = ">="
		case ">=":
			after = ">"
		case "+":
			after = "-"
		case "-":
			if i > 0 && (tokens[i-1].Kind == TokenIdent || tokens[i-1].Kind == TokenInt || tokens[i-1].Text == ")") {
				after = "+"
			}
		case "true":
			after = "false"
		case "false":
			after = "true"
		}
		if after == "" {
			continue
		}
		start := offset(token)
		body := string(runes[:start]) + after + string(runes[end:])
		out = append(out, Mutation{ID: symbol + "/mutation/t" + strconv.Itoa(i), Symbol: symbol, Operator: before + " -> " + after, Before: before, After: after, BodyLine: token.Line, BodyColumn: token.Column, Body: body})
	}
	return out, nil
}
func MutatedSpec(spec ProgramSpec, mutation Mutation) ProgramSpec {
	result := spec
	result.Modules = append([]ModuleSpec{}, spec.Modules...)
	for i, mod := range result.Modules {
		result.Modules[i].Funcs = append([]FuncSpec{}, mod.Funcs...)
		for j, fn := range mod.Funcs {
			if FunctionID(fn) == mutation.Symbol {
				result.Modules[i].Funcs[j].Body = mutation.Body
			}
		}
	}
	return result
}
