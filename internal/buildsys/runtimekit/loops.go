package runtimekit

import "fmt"

type forStmt struct {
	Name  string
	Items expr
	Body  []stmt
}

func copyLocals(src map[string]value) map[string]value {
	out := map[string]value{}
	for k, v := range src {
		out[k] = v
	}
	return out
}

func (ctx *execContext) execFor(loop *forStmt) (value, bool, error) {
	v, err := ctx.eval(loop.Items)
	if err != nil {
		return nil, false, err
	}
	items, ok := v.([]value)
	if !ok {
		return nil, false, fmt.Errorf("for requires a List, got %T", v)
	}
	for _, item := range items {
		if ctx.rt.coverage != nil {
			ctx.rt.coverage.hit(loop, "body")
		}
		child := *ctx
		child.locals = copyLocals(ctx.locals)
		child.mutable = copyMutable(ctx.mutable)
		child.shadowed = map[string]bool{loop.Name: true}
		child.locals[loop.Name] = item
		child.mutable[loop.Name] = false
		result, returned, err := child.execStatements(loop.Body)
		for k := range ctx.locals {
			if ctx.mutable[k] && !child.shadowed[k] {
				ctx.locals[k] = child.locals[k]
			}
		}
		if err != nil || returned {
			return result, returned, err
		}
	}
	if ctx.rt.coverage != nil {
		ctx.rt.coverage.hit(loop, "exit")
	}
	return nil, false, nil
}
