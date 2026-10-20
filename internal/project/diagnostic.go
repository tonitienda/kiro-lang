package project

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

var locPattern = regexp.MustCompile(`(\d+):(\d+)`)

// Legacy prose locations are identified as legacy, never advertised as token spans.
func SourceDiagnostic(path, src string, err error, fallback string) error {
	d := runtimekit.DiagnosticOf(err, fallback)
	d.Message = err.Error()
	if path != "" {
		absolute, e := filepath.Abs(path)
		if e == nil {
			path = absolute
		}
		d.File = path
	}
	if d.Range == nil {
		match := locPattern.FindStringSubmatch(err.Error())
		if len(match) == 3 {
			line, _ := strconv.Atoi(match[1])
			col, _ := strconv.Atoi(match[2])
			d.Range = runtimekit.DeclarationRange(line, col, 1)
			d.LocationKind = "legacy"
		}
	}
	if d.Range == nil {
		if path != "" {
			d.Message = path + ": " + d.Message
		}
		return &d
	}
	line, col := d.Range.Start.Line, d.Range.Start.Column
	lines := strings.Split(src, "\n")
	if line > 0 && line <= len(lines) {
		d.Message = fmt.Sprintf("%s:%d:%d: %s\n  %s\n  %s^", path, line, col, err.Error(), lines[line-1], strings.Repeat(" ", max(col-1, 0)))
	}
	return &d
}
func withSourceLocation(path, src string, err error) error {
	return SourceDiagnostic(path, src, err, "KIRO_CHECK_FAILED")
}
