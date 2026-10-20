package runtimekit

import (
	"errors"
	"strings"
)

// Coordinates use one-based Unicode code-point columns, with exclusive ends.
// Declaration locations are explicitly distinguished from exact token locations.
type SourcePosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}
type SourceRange struct {
	Start SourcePosition `json:"start"`
	End   SourcePosition `json:"end"`
}
type Diagnostic struct {
	Code         string       `json:"code"`
	Severity     string       `json:"severity"`
	Message      string       `json:"message"`
	File         string       `json:"file,omitempty"`
	Range        *SourceRange `json:"range,omitempty"`
	LocationKind string       `json:"location_kind,omitempty"`
	Symbol       string       `json:"symbol,omitempty"`
	Target       string       `json:"target,omitempty"`
	Expected     string       `json:"expected,omitempty"`
	Actual       string       `json:"actual,omitempty"`
	Missing      []string     `json:"missing,omitempty"`
	Hint         string       `json:"hint,omitempty"`
}

func (d *Diagnostic) Error() string { return d.Message }
func NewDiagnostic(code, message string) *Diagnostic {
	return &Diagnostic{Code: code, Severity: "error", Message: message}
}
func DiagnosticOf(err error, fallback string) Diagnostic {
	var d *Diagnostic
	if errors.As(err, &d) {
		copy := *d
		copy.Missing = append([]string(nil), d.Missing...)
		return copy
	}
	return *NewDiagnostic(fallback, err.Error())
}
func TypeMismatch(message, expected, actual string) error {
	d := NewDiagnostic("KIRO_TYPE_MISMATCH", message)
	d.Expected = expected
	d.Actual = actual
	return d
}
func FunctionDiagnostic(err error, fn FuncSpec, fallback string) error {
	d := DiagnosticOf(err, fallback)
	d.Message = fn.Module + "." + fn.Name + ": " + err.Error()
	d.Symbol = fn.Module + "." + fn.Name
	if fn.ReceiverType != "" {
		receiver := fn.ReceiverType
		if !strings.Contains(receiver, ".") {
			receiver = fn.Module + "." + receiver
		}
		d.Symbol = receiver + "." + fn.Name
	}
	d.File = fn.File
	if fn.Line > 0 {
		d.Range = DeclarationRange(fn.Line, fn.Column, len([]rune(fn.Name)))
		d.LocationKind = "declaration"
	}
	return &d
}
func DeclarationRange(line, column, width int) *SourceRange {
	return &SourceRange{Start: SourcePosition{line, column}, End: SourcePosition{line, column + width}}
}

// This catalog describes codes, not prose matching rules. Unconverted legacy
// errors retain a broad fallback code and no invented expected/actual fields.
func DiagnosticCodes() map[string]string {
	return map[string]string{
		"KIRO_SYNTAX":             "Source or executable-body syntax is invalid.",
		"KIRO_CHECK_FAILED":       "A legacy compiler contract failed; inspect its message.",
		"KIRO_PROJECT":            "Project input, filesystem or import resolution failed.",
		"KIRO_TYPE_MISMATCH":      "Known actual type differs from its expected contract.",
		"KIRO_MISSING_RETURN":     "A non-nil function can finish without returning.",
		"KIRO_NON_EXHAUSTIVE":     "A known result or variant match misses total constructor arms.",
		"KIRO_ARGUMENT_MISSING":   "A required keyed argument is absent.",
		"KIRO_ARGUMENT_UNKNOWN":   "An argument key is not declared by the callable.",
		"KIRO_ARGUMENT_DUPLICATE": "An argument key is repeated.",
		"KIRO_UNKNOWN_SYMBOL":     "An identifier cannot be resolved in its scope.",
		"KIRO_EFFECT_REQUIRED":    "A call requires an undeclared operational effect.",
		"KIRO_FORMAT_REQUIRED":    "Source differs from canonical formatting.",
		"KIRO_MUTATION_SURVIVOR":  "An explicitly requested mutation gate found surviving mutations.",
		"KIRO_ANALYSIS":           "Explicit analysis could not complete.",
		"KIRO_ANALYSIS_BUDGET":    "Explicit analysis exceeded its selected budget.",
		"KIRO_ARCHITECTURE":       "An explicitly requested architecture rule failed.",
		"KIRO_TEST_FAILED":        "Explicit coverage tests failed or could not finish.",
		"KIRO_USAGE":              "Command arguments are invalid.",
	}
}

func FunctionID(fn FuncSpec) string {
	if fn.ReceiverType != "" {
		receiver := fn.ReceiverType
		if !strings.Contains(receiver, ".") {
			receiver = fn.Module + "." + receiver
		}
		return receiver + "." + fn.Name
	}
	return fn.Module + "." + fn.Name
}
