package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/parser"
)

type File struct {
	Path string
	Rel  string
	AST  *ast.File
	Src  string
}

var stdlibModules = map[string]struct{}{
	"fs": {}, "http": {}, "json": {}, "cli": {}, "env": {}, "log": {}, "ctx": {}, "parse": {}, "test": {}, "maybe": {}, "time": {},
}

type Project struct {
	Root  string
	Entry string
	Files []File
}

func Load(entry string) (*Project, error) {
	entryPath, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(entryPath)
	if err != nil {
		return nil, err
	}

	proj := &Project{}
	if info.IsDir() {
		proj.Root = entryPath
		proj.Entry = filepath.Join(entryPath, "main.ki")
		if _, err := os.Stat(proj.Entry); err != nil {
			return nil, fmt.Errorf("entry directory %s must contain main.ki", entry)
		}
	} else {
		if !strings.HasSuffix(entryPath, ".ki") {
			return nil, errors.New("entry must be a .ki file or a directory")
		}
		proj.Entry = entryPath
		proj.Root = filepath.Dir(entryPath)
	}

	if err := filepath.WalkDir(proj.Root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != proj.Root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ki") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.Parse(string(src))
		if err != nil {
			return withSourceLocation(path, string(src), err)
		}
		rel, _ := filepath.Rel(proj.Root, path)
		proj.Files = append(proj.Files, File{Path: path, Rel: filepath.ToSlash(rel), AST: f, Src: string(src)})
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(proj.Files, func(i, j int) bool { return proj.Files[i].Rel < proj.Files[j].Rel })

	if err := proj.resolveImports(); err != nil {
		return nil, err
	}
	if err := validateEffects(proj); err != nil {
		return nil, err
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
			return fmt.Errorf("%s: unresolved import %q\nhint: import a stdlib module (%s) or a project module whose `mod` name/path matches the import", f.Rel, imp, strings.Join(sortedStdlibModules(), ", "))
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
