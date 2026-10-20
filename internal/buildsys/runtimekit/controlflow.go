package runtimekit

import "strings"

// A successful path must explicitly return unless nil is admitted by the type.
// Loops may be empty; ? may succeed. Neither establishes a definite return.
func needsReturn(typ string) bool {
	return typ != "nil" && typ != "Any" && !strings.HasPrefix(typ, "?")
}
func statementsReturn(stmts []stmt) bool {
	for _, st := range stmts {
		switch n := st.(type) {
		case *returnStmt:
			return true
		case *letStmt:
			if expressionReturns(n.Value) {
				return true
			}
		case *assignStmt:
			if expressionReturns(n.Value) {
				return true
			}
		case *exprStmt:
			if expressionReturns(n.Value) {
				return true
			}
		case *groupStmt:
			if statementsReturn(n.Body) {
				return true
			}
		case *whenStmt:
			if expressionReturns(n.Value) || whenReturns(n.Cases) {
				return true
			}
		case *forStmt:
			if expressionReturns(n.Items) {
				return true
			}
		}
	}
	return false
}
func expressionReturns(e expr) bool {
	switch n := e.(type) {
	case *partialExpr:
		return expressionReturns(n.Call)
	case *whenExpr:
		return expressionReturns(n.Value) || whenReturns(n.Cases)
	case *sequenceExpr:
		for _, e := range n.Items {
			if expressionReturns(e) {
				return true
			}
		}
	case *binaryExpr:
		return expressionReturns(n.Left) || (n.Op != "&&" && n.Op != "||" && expressionReturns(n.Right))
	case *callExpr:
		if expressionReturns(n.Callee) {
			return true
		}
		for _, arg := range n.Args {
			if expressionReturns(arg.Value) {
				return true
			}
		}
	case *structExpr:
		for _, f := range n.Fields {
			if expressionReturns(f.Value) {
				return true
			}
		}
	case *listExpr:
		for _, item := range n.Items {
			if expressionReturns(item) {
				return true
			}
		}
	case *unwrapExpr:
		return expressionReturns(n.Value)
	case *awaitExpr:
		return expressionReturns(n.Value)
	case *selectorExpr:
		return expressionReturns(n.Left)
	}
	return false
}
func whenReturns(arms []whenCase) bool {
	if !whenCovers(arms) {
		return false
	}
	for _, arm := range arms {
		if arm.Expr != nil {
			if !expressionReturns(arm.Expr) {
				return false
			}
		} else if !statementsReturn(arm.Body) {
			return false
		}
	}
	return true
}

// The semantic checker has already verified constructor coverage and pattern
// types. Primitive dispatch needs a catch-all, binding, or both boolean literals.
func whenCovers(arms []whenCase) bool {
	constructors, yes, no := false, false, false
	for _, arm := range arms {
		if arm.Wildcard {
			return true
		}
		switch p := arm.Pattern.(type) {
		case *bindingExpr:
			return true
		case *identExpr:
			if p.Name == "_" {
				return true
			}
			yes = yes || p.Name == "true"
			no = no || p.Name == "false"
		case *callExpr:
			constructors = true
		}
	}
	return constructors || (yes && no)
}
