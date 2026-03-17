package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiro-lang/kiro/internal/codegen"
	"github.com/kiro-lang/kiro/internal/compat"
	"github.com/kiro-lang/kiro/internal/format"
	"github.com/kiro-lang/kiro/internal/lsp"
	"github.com/kiro-lang/kiro/internal/project"
)

const usageText = `usage: kiro <command> [args]

Core commands:
  fmt <paths...>                          Format .ki files deterministically
  check <entry-or-path>                   Parse and type-check a module/project
  inspect go <entry-or-path> [--out-dir]  Emit generated Go for inspection
  new <hello|service>                     Scaffold a starter project
  lsp                                     Run language server over stdio
  compat [root] [--mode fmt,check,inspect]
                                          Run compatibility fixture checks

Placeholders in this repo slice:
  build <entry>
  run <entry>
  test <path>

Use 'kiro help' to print this message.`

func Run(args []string) error {
	if len(args) == 0 {
		return errors.New(usageText)
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Println(usageText)
		return nil
	case "fmt":
		return runFmt(args[1:])
	case "check":
		return runCheck(args[1:])
	case "compat":
		return runCompat(args[1:])
	case "inspect":
		return runInspect(args[1:])
	case "new":
		return runNew(args[1:])
	case "lsp":
		return lsp.NewServer().Serve(os.Stdin, os.Stdout)
	case "build", "run", "test":
		return fmt.Errorf("%s is not implemented in this frontend-focused slice", args[0])
	default:
		return fmt.Errorf("unknown command: %s\n\n%s", args[0], usageText)
	}
}

func runCompat(args []string) error {
	root := "compat"
	modes := map[string]bool{"fmt": true, "check": true, "inspect": true}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			if i+1 >= len(args) {
				return errors.New("usage: kiro compat [root] [--mode fmt,check,inspect]")
			}
			modes = map[string]bool{}
			for _, m := range strings.Split(args[i+1], ",") {
				modes[strings.TrimSpace(m)] = true
			}
			i++
		default:
			root = args[i]
		}
	}
	return compat.Run(compat.RunOptions{Root: root, Modes: modes})
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
	if err := os.MkdirAll("service/app", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("service/internal/config", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("service/test", 0o755); err != nil {
		return err
	}

	main := `mod main

import app
import internal/config
import http
import log

fn main() -> i32 {
  let cfg = config.load()?
  log.info("starting ${cfg.port}")
  http.serve(cfg.port, app.handler)?
  return 0
}
`
	app := `mod app

import http

fn handler(req:httpReq) -> Resp {
  when req.path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.not_found())
    }
}
`
	config := `mod config

import env

type AppConfig {
  port:str
  env:str
}

fn load() -> R[AppConfig, str] {
  let port = env.get_or("PORT", ":8080")
  let app_env = env.get_or("APP_ENV", "dev")
  return Ok(AppConfig{port:port env:app_env})
}
`
	testFile := `mod health_test

import app
import http
import test

fn test_health_handler() -> nil {
  let req = http.test_req("GET", "/health", "")
  let res = app.handler(req)?
  test.eq(res.code, 200)
}
`
	readme := `# Kiro service template

This template shows the Phase 7 service shape:

- ` + "`internal/config`" + ` for explicit config loading
- ` + "`app`" + ` module for handler composition
- handler-level test via ` + "`http.test_req`" + ` style helpers

Check and inspect generated Go:

` + "```bash" + `
kiro check .
kiro inspect go . --out-dir .kiro-gen
` + "```" + `
`

	if err := os.WriteFile("service/main.ki", []byte(main), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("service/app/main.ki", []byte(app), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("service/internal/config/main.ki", []byte(config), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("service/test/health.ki", []byte(testFile), 0o644); err != nil {
		return err
	}
	return os.WriteFile("service/README.md", []byte(readme), 0o644)
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
