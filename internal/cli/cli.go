package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiro-lang/kiro/internal/format"
)

func Run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: kiro <fmt|build|run|test> ...")
	}
	switch args[0] {
	case "fmt":
		return runFmt(args[1:])
	case "build", "run", "test":
		return fmt.Errorf("%s is not implemented in milestone 1", args[0])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runFmt(paths []string) error {
	if len(paths) == 0 {
		return errors.New("usage: kiro fmt <paths...>")
	}
	for _, path := range paths {
		if err := formatPath(path); err != nil {
			return err
		}
	}
	return nil
}

func formatPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".ki") {
				return nil
			}
			return formatFile(p)
		})
	}
	if strings.HasSuffix(path, ".ki") {
		return formatFile(path)
	}
	return nil
}

func formatFile(path string) error {
	in, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := format.Source(string(in))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, []byte(out), 0o644)
}
