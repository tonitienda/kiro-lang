package buildsys

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/project"
	"github.com/kiro-lang/kiro/internal/toolchain"
)

//go:embed runtimekit/*.go
var runtimeKit embed.FS

type BuildMode string

const (
	ModeBuild BuildMode = "build"
	ModeRun   BuildMode = "run"
	ModeTest  BuildMode = "test"
)

type Options struct {
	Entry   string
	Out     string
	KeepGen bool
	Mode    BuildMode
	Args    []string
	WorkDir string
	Stdout  *os.File
	Stderr  *os.File
}

type Result struct {
	Project     *project.Project
	Binary      string
	WorkDir     string
	GoBinary    string
	GoSource    string
	ProgramSpec ProgramSpec
}

type ProgramSpec struct {
	Mode        string       `json:"mode"`
	EntryModule string       `json:"entry_module"`
	Modules     []ModuleSpec `json:"modules"`
}

type ModuleSpec struct {
	Name   string      `json:"name"`
	Consts []ConstSpec `json:"consts"`
	Types  []TypeSpec  `json:"types"`
	Funcs  []FuncSpec  `json:"funcs"`
}

type ConstSpec struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	ValueKind string `json:"value_kind"`
}

type TypeSpec struct {
	Name   string      `json:"name"`
	Fields []FieldSpec `json:"fields"`
}

type FieldSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type FuncSpec struct {
	Module       string      `json:"module"`
	Name         string      `json:"name"`
	ReceiverType string      `json:"receiver_type,omitempty"`
	Params       []ParamSpec `json:"params"`
	ReturnType   string      `json:"return_type"`
	BlockBody    bool        `json:"block_body"`
	Body         string      `json:"body"`
}

type ParamSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func Build(opts Options) (*Result, error) {
	proj, err := project.Load(opts.Entry)
	if err != nil {
		return nil, err
	}
	goLoc, err := toolchain.Locate()
	if err != nil {
		return nil, err
	}
	workDir := opts.WorkDir
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "kiro-build-*")
		if err != nil {
			return nil, err
		}
	}
	out := opts.Out
	if out != "" && !filepath.IsAbs(out) {
		out = filepath.Join(mustGetwd(), out)
	}
	if out == "" {
		name := filepath.Base(proj.Root)
		if name == "." || name == string(filepath.Separator) || name == "" {
			name = "kiro-app"
		}
		out = filepath.Join(mustGetwd(), name)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(out, ".exe") {
		out += ".exe"
	}
	mode := string(opts.Mode)
	if mode == "" {
		mode = string(ModeBuild)
	}
	spec := projectToSpec(proj, mode)
	if err := emitWorkDir(workDir, spec); err != nil {
		return nil, err
	}
	cmd := exec.Command(goLoc.GoBinary, "build", "-o", out, "./cmd/kiro_program")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &Result{Project: proj, Binary: out, WorkDir: workDir, GoBinary: goLoc.GoBinary, GoSource: goLoc.Source, ProgramSpec: spec}, nil
}

func projectToSpec(proj *project.Project, mode string) ProgramSpec {
	byModule := map[string]*ModuleSpec{}
	for _, file := range proj.Files {
		mod := byModule[file.AST.Module]
		if mod == nil {
			mod = &ModuleSpec{Name: file.AST.Module}
			byModule[file.AST.Module] = mod
		}
		for _, decl := range file.AST.Decls {
			switch d := decl.(type) {
			case ast.ConstDecl:
				mod.Consts = append(mod.Consts, ConstSpec{Name: d.Name, Value: d.Value, ValueKind: d.ValueKind})
			case ast.TypeDecl:
				fields := make([]FieldSpec, 0, len(d.Fields))
				for _, f := range d.Fields {
					fields = append(fields, FieldSpec{Name: f.Name, Type: f.Type})
				}
				mod.Types = append(mod.Types, TypeSpec{Name: d.Name, Fields: fields})
			case ast.FuncDecl:
				params := make([]ParamSpec, 0, len(d.Params))
				for _, p := range d.Params {
					params = append(params, ParamSpec{Name: p.Name, Type: p.Type})
				}
				receiverType := ""
				if d.Receiver != nil {
					receiverType = d.Receiver.Type
					params = append([]ParamSpec{{Name: d.Receiver.Name, Type: d.Receiver.Type}}, params...)
				}
				mod.Funcs = append(mod.Funcs, FuncSpec{Module: file.AST.Module, Name: d.Name, ReceiverType: receiverType, Params: params, ReturnType: d.ReturnType, BlockBody: d.BlockBody, Body: d.Body})
			}
		}
	}
	moduleNames := make([]string, 0, len(byModule))
	for name := range byModule {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)
	mods := make([]ModuleSpec, 0, len(moduleNames))
	for _, name := range moduleNames {
		mods = append(mods, *byModule[name])
	}
	entryModule := proj.Files[0].AST.Module
	for _, file := range proj.Files {
		if filepath.Clean(file.Path) == filepath.Clean(proj.Entry) {
			entryModule = file.AST.Module
			break
		}
	}
	return ProgramSpec{Mode: mode, EntryModule: entryModule, Modules: mods}
}

func emitWorkDir(workDir string, spec ProgramSpec) error {
	for _, dir := range []string{filepath.Join(workDir, "kiro_runtime"), filepath.Join(workDir, "cmd", "kiro_program")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module generated\n\ngo 1.22\n"), 0o644); err != nil {
		return err
	}
	entries, err := runtimeKit.ReadDir("runtimekit")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		data, err := runtimeKit.ReadFile(filepath.Join("runtimekit", entry.Name()))
		if err != nil {
			return err
		}
		content := strings.ReplaceAll(string(data), "package runtimekit", "package kiro_runtime")
		if err := os.WriteFile(filepath.Join(workDir, "kiro_runtime", entry.Name()), []byte(content), 0o644); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	mainSrc := fmt.Sprintf("package main\n\nimport (\n\t\"os\"\n\tkiro_runtime \"generated/kiro_runtime\"\n)\n\nconst rawSpec = %q\n\nfunc main() {\n\tos.Exit(kiro_runtime.Main(rawSpec))\n}\n", string(raw))
	return os.WriteFile(filepath.Join(workDir, "cmd", "kiro_program", "main.go"), []byte(mainSrc), 0o644)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
