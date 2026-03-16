package project

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var locPattern = regexp.MustCompile(`(\d+):(\d+)`)

func withSourceLocation(path, src string, err error) error {
	msg := err.Error()
	match := locPattern.FindStringSubmatch(msg)
	if len(match) != 3 {
		return fmt.Errorf("%s: %w", path, err)
	}
	line, _ := strconv.Atoi(match[1])
	col, _ := strconv.Atoi(match[2])
	lines := strings.Split(src, "\n")
	if line <= 0 || line > len(lines) {
		return fmt.Errorf("%s:%d:%d: %s", path, line, col, msg)
	}
	snippet := lines[line-1]
	caret := strings.Repeat(" ", max(col-1, 0)) + "^"
	return fmt.Errorf("%s:%d:%d: %s\n  %s\n  %s", path, line, col, msg, snippet, caret)
}
