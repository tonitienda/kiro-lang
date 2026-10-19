package runtimekit

import (
	"fmt"
	"strings"
)

// BuiltinParameters is shared by project checking and executable call binding.
// Parameter names are part of the source-level API.
func BuiltinParameters(name string) ([]string, bool) {
	parameters, ok := builtinParameters[name]
	return append([]string{}, parameters...), ok
}

func BuiltinSignatures() map[string][]string {
	result := make(map[string][]string, len(builtinParameters))
	for name, parameters := range builtinParameters {
		result[name] = append([]string{}, parameters...)
	}
	return result
}

var builtinParameters = map[string][]string{
	"print": {"value"}, "println": {"value"},
	"Ok": {"value"}, "Err": {"error"},
	"env.get_or":       {"key", "fallback"},
	"log.info":         {"message"},
	"test.eq":          {"actual", "expected"},
	"assert.equal_i32": {"actual", "expected"},
	"parse.i32":        {"text"}, "cli.args": {},
	"fs.read_file": {"path"}, "fs.write_file": {"path", "contents"},
	"http.text": {"code", "body"}, "http.json": {"code", "body"},
	"http.not_found": {}, "http.with_header": {"resp", "key", "value"},
	"http.test_req": {"method", "path", "body"}, "http.query": {"req", "key"},
	"http.serve":  {"addr", "handler"},
	"json.encode": {"value"}, "json.decode": {"text"},
	"ctx.background": {}, "ctx.with_timeout_ms": {"parent", "ms"},
	"list.map": {"items", "transform"}, "list.filter": {"items", "predicate"}, "list.fold": {"items", "initial", "reduce"},
}

// ArgumentOrder validates a complete keyed call before any argument is evaluated.
// The returned indices bind source-order arguments to declaration-order parameters.
func ArgumentOrder(parameters, keys []string) ([]int, error) {
	expected := map[string]bool{}
	for _, parameter := range parameters {
		if expected[parameter] {
			return nil, fmt.Errorf("duplicate parameter %q in callable signature", parameter)
		}
		expected[parameter] = true
	}
	positions := map[string]int{}
	for i, key := range keys {
		if _, duplicate := positions[key]; duplicate {
			return nil, fmt.Errorf("duplicate argument %q\nhint: supply each argument exactly once", key)
		}
		if !expected[key] {
			return nil, fmt.Errorf("unknown argument %q\nexpected: %s\nhint: use the declared parameter names", key, strings.Join(parameters, ", "))
		}
		positions[key] = i
	}
	order := make([]int, len(parameters))
	for i, parameter := range parameters {
		position, ok := positions[parameter]
		if !ok {
			return nil, fmt.Errorf("missing argument %q\nhint: supply %s: expression; optional values still require an explicit argument", parameter, parameter)
		}
		order[i] = position
	}
	return order, nil
}

type CallSite struct {
	Target  string
	Keys    []string
	Line    int
	Column  int
	Partial bool
}

// InspectCalls checks keyed-call syntax without pretending to parse every body
// construct. This lets the frontend reject bad calls even in its wider grammar.
func InspectCalls(body string) ([]CallSite, error) {
	tokens, err := Lex(body)
	if err != nil {
		return nil, err
	}
	var sites []CallSite
	for i, token := range tokens {
		if token.Kind != TokenIdent || token.Text == "if" || token.Text == "while" || (i > 0 && tokens[i-1].Text == ".") {
			continue
		}
		j := i + 1
		parts := []string{token.Text}
		for j+1 < len(tokens) && tokens[j].Text == "." && tokens[j+1].Kind == TokenIdent {
			parts = append(parts, tokens[j+1].Text)
			j += 2
		}
		if j < len(tokens) && tokens[j].Text == "[" {
			depth := 1
			j++
			for j < len(tokens) && depth > 0 {
				if tokens[j].Text == "[" {
					depth++
				}
				if tokens[j].Text == "]" {
					depth--
				}
				j++
			}
		}
		if j >= len(tokens) || tokens[j].Text != "(" {
			continue
		}
		keys, err := inspectArgumentKeys(tokens, j)
		if err != nil {
			return nil, fmt.Errorf("%d:%d: call %s: %w", token.Line, token.Column, strings.Join(parts, "."), err)
		}
		sites = append(sites, CallSite{Target: strings.Join(parts, "."), Keys: keys, Line: token.Line, Column: token.Column, Partial: i > 0 && tokens[i-1].Text == "partial"})
	}
	return sites, nil
}

func inspectArgumentKeys(tokens []Token, open int) ([]string, error) {
	var keys []string
	seen := map[string]bool{}
	i := open + 1
	skipLines := func() {
		for i < len(tokens) && tokens[i].Kind == TokenNewline {
			i++
		}
	}
	for {
		skipLines()
		if i >= len(tokens) || tokens[i].Kind == TokenEOF {
			return nil, fmt.Errorf("unterminated call")
		}
		if tokens[i].Text == ")" {
			return keys, nil
		}
		if tokens[i].Kind != TokenIdent || i+1 >= len(tokens) || tokens[i+1].Text != ":" {
			return nil, fmt.Errorf("named arguments are required\nhint: write name: expression for every value argument")
		}
		key := tokens[i].Text
		if seen[key] {
			return nil, fmt.Errorf("duplicate argument %q", key)
		}
		seen[key] = true
		keys = append(keys, key)
		i += 2
		skipLines()
		start := i
		var stack []string
		for i < len(tokens) && tokens[i].Kind != TokenEOF {
			token := tokens[i]
			if token.Kind == TokenSymbol {
				if len(stack) == 0 && (token.Text == "," || token.Text == ")") {
					break
				}
				switch token.Text {
				case "(":
					stack = append(stack, ")")
				case "[":
					stack = append(stack, "]")
				case "{":
					stack = append(stack, "}")
				case ")", "]", "}":
					if len(stack) == 0 || stack[len(stack)-1] != token.Text {
						return nil, fmt.Errorf("unbalanced argument expression")
					}
					stack = stack[:len(stack)-1]
				}
			}
			if len(stack) == 0 && i > start && token.Kind == TokenIdent && i+1 < len(tokens) && tokens[i+1].Text == ":" {
				return nil, fmt.Errorf("expected comma between named arguments")
			}
			i++
		}
		if i == start {
			return nil, fmt.Errorf("missing expression for argument %q", key)
		}
		if i >= len(tokens) || tokens[i].Kind == TokenEOF {
			return nil, fmt.Errorf("unterminated call")
		}
		if tokens[i].Text == "," {
			i++
			continue
		}
		return keys, nil
	}
}

func CheckArgumentKeys(parameters, keys []string, partial bool) ([]int, error) {
	if !partial {
		return ArgumentOrder(parameters, keys)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("partial must bind at least one argument")
	}
	allowed := map[string]bool{}
	for _, name := range parameters {
		allowed[name] = true
	}
	for _, name := range keys {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown argument %q in partial application", name)
		}
	}
	return ArgumentOrder(keys, keys)
}
