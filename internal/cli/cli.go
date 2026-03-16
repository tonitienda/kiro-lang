package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiro-lang/kiro/internal/codegen"
	"github.com/kiro-lang/kiro/internal/format"
	"github.com/kiro-lang/kiro/internal/project"
)

func Run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: kiro <fmt|check|inspect|new|build|run|test> ...")
	}
	switch args[0] {
	case "fmt":
		return runFmt(args[1:])
	case "check":
		return runCheck(args[1:])
	case "inspect":
		return runInspect(args[1:])
	case "new":
		return runNew(args[1:])
	case "build", "run", "test":
		return fmt.Errorf("%s is not implemented in this frontend-focused slice", args[0])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runCheck(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: kiro check <entry-or-path>")
	}
	proj, err := project.Load(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("check ok: %d files\n", len(proj.Files))
	return nil
}

func runInspect(args []string) error {
	if len(args) < 2 || args[0] != "go" {
		return errors.New("usage: kiro inspect go <entry-or-path> [--out-dir <dir>]")
	}
	outDir := ".kiro-gen"
	entry := args[1]
	for i := 2; i < len(args); i++ {
		if args[i] == "--out-dir" && i+1 < len(args) {
			outDir = args[i+1]
			i++
		}
	}
	proj, err := project.Load(entry)
	if err != nil {
		return err
	}
	if err := codegen.EmitProjectGo(proj, outDir); err != nil {
		return err
	}
	fmt.Printf("generated Go for %d files in %s\n", len(proj.Files), outDir)
	return nil
}

func runNew(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: kiro new <hello|service>")
	}
	switch args[0] {
	case "hello":
		return scaffoldHello()
	case "service":
		return scaffoldService()
	default:
		return errors.New("unknown template: use hello or service")
	}
}

func scaffoldHello() error {
	if err := os.MkdirAll("hello", 0o755); err != nil {
		return err
	}
	return os.WriteFile("hello/main.ki", []byte(`mod main

fn main() -> i32 {
  println("hello")
  return 0
}
`), 0o644)
}

func scaffoldService() error {
	if err := os.MkdirAll("service", 0o755); err != nil {
		return err
	}
	main := `mod main

import http
import log

fn app(req:httpReq) -> Resp {
  when req.path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.text(404, "not found"))
    }
}

fn main() -> i32 {
  log.info("starting :8080")
  http.serve(":8080", app)?
  return 0
}
`
	if err := os.WriteFile("service/main.ki", []byte(main), 0o644); err != nil {
		return err
	}
	return os.WriteFile("service/README.md", []byte("Run with `kiro run service`.\n"), 0o644)
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
