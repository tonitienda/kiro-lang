package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/parser"
)

type File struct {
	Path string
	Rel  string
	AST  *ast.File
	Src  string
}

var stdlibModules = map[string]struct{}{
	"list": {}, "fs": {}, "http": {}, "json": {}, "cli": {}, "env": {}, "log": {}, "ctx": {}, "parse": {}, "test": {}, "maybe": {}, "time": {},
}

type Project struct {
	Root  string
	Entry string
	Files []File
}

func Load(entry string) (*Project, error) { return LoadOverlay(entry, nil) }
func LoadOverlay(entry string, overlay map[string]string) (*Project, error) {
	p, err := ParseOverlay(entry, overlay)
	if err != nil {
		d := runtimekit.DiagnosticOf(err, "KIRO_PROJECT")
		return nil, &d
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}
func Parse(entry string) (*Project, error) { return ParseOverlay(entry, nil) }
func (proj *Project) Validate() error {
	if err := proj.resolveImports(); err != nil {
		return err
	}
	if err := validateEffects(proj); err != nil {
		return err
	}
	if err := validateCalls(proj); err != nil {
		return err
	}
	return validateFunctionalContracts(proj)
}

// Overlay keys are absolute paths; contents are checked without writing to disk.
func ParseOverlay(entry string, overlay map[string]string) (*Project, error) {
	entryPath, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(entryPath)
	if err != nil {
		if _, exists := overlay[entryPath]; !exists {
			return nil, err
		}
	}

	proj := &Project{}
	if info != nil && info.IsDir() {
		proj.Root = entryPath
		proj.Entry = filepath.Join(entryPath, "main.ki")
		if _, err := os.Stat(proj.Entry); err != nil && !hasOverlay(overlay, proj.Entry) {
			return nil, fmt.Errorf("entry directory %s must contain main.ki", entry)
		}
	} else {
		if !strings.HasSuffix(entryPath, ".ki") {
			return nil, errors.New("entry must be a .ki file or a directory")
		}
		proj.Entry = entryPath
		proj.Root = filepath.Dir(entryPath)
	}

	paths := map[string]bool{}
	if err := filepath.WalkDir(proj.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != proj.Root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".ki") {
			paths[path] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for path := range overlay {
		rel, err := filepath.Rel(proj.Root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) || !strings.HasSuffix(path, ".ki") {
			continue
		}
		hidden := false
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			hidden = hidden || strings.HasPrefix(part, ".")
		}
		if !hidden {
			paths[path] = true
		}
	}
	names := []string{}
	for path := range paths {
		names = append(names, path)
	}
	sort.Strings(names)
	for _, path := range names {
		src, exists := overlay[path]
		if !exists {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			src = string(data)
		}
		f, err := parser.Parse(src)
		if err != nil {
			return nil, SourceDiagnostic(path, src, err, "KIRO_SYNTAX")
		}
		rel, _ := filepath.Rel(proj.Root, path)
		proj.Files = append(proj.Files, File{Path: path, Rel: filepath.ToSlash(rel), AST: f, Src: src})
	}
	if len(proj.Files) == 0 {
		return nil, errors.New("project contains no .ki source files")
	}

	return proj, nil
}

func (p *Project) resolveImports() error {
	mods := map[string]string{}
	for _, f := range p.Files {
		mods[f.AST.Module] = f.Rel
	}

	for _, f := range p.Files {
		for _, imp := range f.AST.Imports {
			if _, ok := stdlibModules[imp]; ok {
				continue
			}
			if _, ok := mods[imp]; ok {
				continue
			}
			if hasModuleByPath(p.Files, imp) {
				continue
			}
			d := runtimekit.NewDiagnostic("KIRO_PROJECT", fmt.Sprintf("%s: unresolved import %q\nhint: import a stdlib module (%s) or a project module whose `mod` name/path matches the import", f.Rel, imp, strings.Join(sortedStdlibModules(), ", ")))
			d.File = f.Path
			d.Target = imp
			d.Hint = "import a supported standard-library or declared project module"
			return d
		}
	}
	return nil
}

func sortedStdlibModules() []string {
	names := make([]string, 0, len(stdlibModules))
	for name := range stdlibModules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func hasModuleByPath(files []File, imp string) bool {
	for _, f := range files {
		if strings.TrimSuffix(f.Rel, ".ki") == imp {
			return true
		}
		if strings.TrimSuffix(f.Rel, "/main.ki") == imp {
			return true
		}
	}
	return false
}

func hasOverlay(overlay map[string]string, path string) bool { _, ok := overlay[path]; return ok }
