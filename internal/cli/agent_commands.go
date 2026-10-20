package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	llmassets "github.com/kiro-lang/kiro/docs/llm"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/format"
	"github.com/kiro-lang/kiro/internal/project"
	"github.com/kiro-lang/kiro/internal/version"
)

const toolSchema = "kiro.tools.v1"

type toolReport struct {
	Schema      string                  `json:"schema"`
	Command     string                  `json:"command"`
	OK          bool                    `json:"ok"`
	Diagnostics []runtimekit.Diagnostic `json:"diagnostics"`
	Data        any                     `json:"data,omitempty"`
}

func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}
func outputReport(command string, data any, diags []runtimekit.Diagnostic, machine bool) error {
	if diags == nil {
		diags = []runtimekit.Diagnostic{}
	}
	report := toolReport{Schema: toolSchema, Command: command, OK: len(diags) == 0, Diagnostics: diags, Data: data}
	if machine {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
		if !report.OK {
			return &ExitError{Code: 1, Reported: true}
		}
		return nil
	}
	if !report.OK {
		messages := []string{}
		for _, d := range diags {
			messages = append(messages, d.Code+": "+d.Message)
		}
		return errors.New(strings.Join(messages, "\n"))
	}
	switch command {
	case "check":
		fmt.Printf("check ok: %d files\n", data.(map[string]any)["files"])
	case "lint":
		fmt.Println("lint ok: semantic contracts and canonical formatting")
	case "fmt":
		fmt.Println("format check ok")
	default:
		raw, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(raw))
	}
	return nil
}
func commandFailure(command string, args []string, err error, code string) error {
	return outputReport(command, nil, []runtimekit.Diagnostic{runtimekit.DiagnosticOf(err, code)}, jsonRequested(args))
}
func usageFailure(command string, args []string, usage string) error {
	return commandFailure(command, args, errors.New(usage), "KIRO_USAGE")
}
func parseSinglePath(args []string) (string, error) {
	entry := ""
	for _, arg := range args {
		if arg == "--json" {
			continue
		}
		if strings.HasPrefix(arg, "-") || entry != "" {
			return "", errors.New("expected one project path and optional --json")
		}
		entry = arg
	}
	if entry == "" {
		return "", errors.New("missing project path")
	}
	return entry, nil
}
func runAnalysis(args []string, lint bool) error {
	command := "check"
	if lint {
		command = "lint"
	}
	entry, err := parseSinglePath(args)
	if err != nil {
		return usageFailure(command, args, "usage: kiro "+command+" <entry-or-path> [--json]")
	}
	p, err := project.Load(entry)
	if err != nil {
		return commandFailure(command, args, err, "KIRO_CHECK_FAILED")
	}
	data := map[string]any{"files": len(p.Files), "revision": p.Revision(), "validation": "semantic", "diagnostic_mode": "first_error"}
	if lint {
		paths := []string{}
		for _, f := range p.Files {
			paths = append(paths, f.Path)
		}
		_, diags := prepareFormatting(paths)
		return outputReport(command, data, diags, jsonRequested(args))
	}
	return outputReport(command, data, nil, jsonRequested(args))
}

type formatChange struct {
	path   string
	output []byte
	mode   fs.FileMode
}

func formatInputs(paths []string) ([]string, error) {
	set := map[string]bool{}
	for _, path := range paths {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if !strings.HasSuffix(absolute, ".ki") {
				return nil, fmt.Errorf("expected .ki source: %s", path)
			}
			set[absolute] = true
			continue
		}
		err = filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if p != absolute && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".ki") {
				set[p] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	out := []string{}
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, errors.New("no .ki files selected")
	}
	return out, nil
}
func prepareFormatting(paths []string) ([]formatChange, []runtimekit.Diagnostic) {
	changes := []formatChange{}
	diags := []runtimekit.Diagnostic{}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			diags = append(diags, runtimekit.DiagnosticOf(err, "KIRO_PROJECT"))
			continue
		}
		out, err := format.Source(string(source))
		if err != nil {
			diags = append(diags, runtimekit.DiagnosticOf(project.SourceDiagnostic(path, string(source), err, "KIRO_SYNTAX"), "KIRO_SYNTAX"))
			continue
		}
		if string(source) == out {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			diags = append(diags, runtimekit.DiagnosticOf(err, "KIRO_PROJECT"))
			continue
		}
		changes = append(changes, formatChange{path, []byte(out), info.Mode().Perm()})
		d := runtimekit.NewDiagnostic("KIRO_FORMAT_REQUIRED", "source differs from canonical formatting")
		d.File = path
		d.Hint = "run kiro fmt on this file"
		diags = append(diags, *d)
	}
	return changes, diags
}
func runAgentFmt(args []string) error {
	paths := []string{}
	check, machine := false, jsonRequested(args)
	for _, arg := range args {
		switch arg {
		case "--check":
			check = true
		case "--json":
		default:
			if strings.HasPrefix(arg, "-") {
				return usageFailure("fmt", args, "usage: kiro fmt <paths...> [--check [--json]]")
			}
			paths = append(paths, arg)
		}
	}
	if len(paths) == 0 || (machine && !check) {
		return usageFailure("fmt", args, "usage: kiro fmt <paths...> [--check [--json]]")
	}
	files, err := formatInputs(paths)
	if err != nil {
		return commandFailure("fmt", args, err, "KIRO_PROJECT")
	}
	changes, diags := prepareFormatting(files)
	if check {
		return outputReport("fmt", map[string]any{"files": len(files), "changed": len(changes), "writes": false}, diags, machine)
	}
	for _, d := range diags {
		if d.Code != "KIRO_FORMAT_REQUIRED" {
			return outputReport("fmt", nil, diags, false)
		}
	}
	// Preflight every source before the first write. I/O failures during writes are
	// still possible; this is not a transactional filesystem operation.
	for _, change := range changes {
		if err := os.WriteFile(change.path, change.output, change.mode); err != nil {
			return err
		}
	}
	return nil
}
func runDescribe(args []string) error {
	name := ""
	for _, arg := range args {
		if arg == "--json" {
			continue
		}
		if strings.HasPrefix(arg, "-") || name != "" {
			return usageFailure("describe", args, "usage: kiro describe [builtin] [--json]")
		}
		name = arg
	}
	raw, err := llmassets.Files.ReadFile("references/kiro.json")
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	builtins := map[string]any{}
	for key, signature := range runtimekit.BuiltinContracts() {
		builtins[key] = map[string]any{"signature": signature.String(), "parameters": signature.Params, "return_type": signature.ReturnType, "effects": append([]string{}, signature.Effects...), "contract_level": "base_signature"}
	}
	if name != "" {
		value, ok := builtins[name]
		if !ok {
			d := runtimekit.NewDiagnostic("KIRO_UNKNOWN_SYMBOL", "unknown builtin "+name)
			d.Target = name
			return commandFailure("describe", args, d, d.Code)
		}
		return outputReport("describe", map[string]any{"name": name, "contract": value}, nil, jsonRequested(args))
	}
	return outputReport("describe", map[string]any{"compiler_version": version.KiroVersion, "guide_version": manifest["version"], "stable_core": manifest["stable_core"], "unsupported_syntax": manifest["unsupported_syntax"], "commands": []string{"check <path> --json", "lint <path> --json", "fmt <paths...> --check --json", "describe [builtin] --json", "symbols <path> [--symbol module.name] --json", "analyze <operation> <path> [options] --json", "lsp"}, "optional_analysis": manifest["agent_tools"].(map[string]any)["optional_analysis"], "diagnostic_codes": runtimekit.DiagnosticCodes(), "builtins": builtins, "limits": []string{"base builtin signatures use Any where specialization depends on arguments, schemas or callbacks", "check reports the first compiler error; lint checks formatting after semantic success", "symbols describe declarations, not resolved dynamic callers"}}, nil, jsonRequested(args))
}
func runSymbols(args []string) error {
	entry, filter := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
		case "--symbol":
			if i+1 >= len(args) || filter != "" || strings.HasPrefix(args[i+1], "-") {
				return usageFailure("symbols", args, "usage: kiro symbols <path> [--symbol module.name] [--json]")
			}
			i++
			filter = args[i]
		default:
			if strings.HasPrefix(args[i], "-") || entry != "" {
				return usageFailure("symbols", args, "usage: kiro symbols <path> [--symbol module.name] [--json]")
			}
			entry = args[i]
		}
	}
	if entry == "" {
		return usageFailure("symbols", args, "usage: kiro symbols <path> [--symbol module.name] [--json]")
	}
	p, err := project.Parse(entry)
	if err != nil {
		return commandFailure("symbols", args, err, "KIRO_PROJECT")
	}
	symbols := p.Symbols()
	if filter != "" {
		selected := []project.Symbol{}
		for _, s := range symbols {
			if s.ID == filter {
				selected = append(selected, s)
			}
		}
		symbols = selected
		if len(symbols) == 0 {
			d := runtimekit.NewDiagnostic("KIRO_UNKNOWN_SYMBOL", "no declaration named "+filter)
			d.Target = filter
			return commandFailure("symbols", args, d, d.Code)
		}
	}
	return outputReport("symbols", map[string]any{"revision": p.Revision(), "validation": "syntax_only", "coordinate_system": "one_based_unicode_codepoints_end_exclusive", "symbols": symbols}, nil, jsonRequested(args))
}
