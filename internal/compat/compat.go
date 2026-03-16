package compat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/codegen"
	"github.com/kiro-lang/kiro/internal/format"
	"github.com/kiro-lang/kiro/internal/project"
)

type FixtureMeta struct {
	Modes           []string `json:"modes"`
	ExpectSuccess   *bool    `json:"expect_success"`
	ErrorContains   string   `json:"error_contains"`
	InspectGo       bool     `json:"inspect_go"`
	Entry           string   `json:"entry"`
	SkipFmt         bool     `json:"skip_fmt"`
	ExpectedModules []string `json:"expected_modules"`
}

type RunOptions struct {
	Root  string
	Modes map[string]bool
}

func Run(opts RunOptions) error {
	if opts.Root == "" {
		opts.Root = "compat"
	}
	if len(opts.Modes) == 0 {
		opts.Modes = map[string]bool{"fmt": true, "check": true, "inspect": true}
	}
	fixtures, err := discoverFixtures(opts.Root)
	if err != nil {
		return err
	}
	if len(fixtures) == 0 {
		return fmt.Errorf("no fixtures found under %s", opts.Root)
	}
	for _, fx := range fixtures {
		if err := runFixture(fx, opts.Modes); err != nil {
			return fmt.Errorf("fixture %s: %w", fx.Name, err)
		}
	}
	fmt.Printf("compat ok: %d fixtures\n", len(fixtures))
	return nil
}

type fixture struct {
	Name string
	Path string
	Meta FixtureMeta
}

func discoverFixtures(root string) ([]fixture, error) {
	var out []fixture
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		mainFile := filepath.Join(path, "main.ki")
		if _, err := os.Stat(mainFile); err != nil {
			return nil
		}
		meta, err := loadMeta(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out = append(out, fixture{Name: filepath.ToSlash(rel), Path: path, Meta: meta})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func loadMeta(path string) (FixtureMeta, error) {
	metaPath := filepath.Join(path, "fixture.json")
	var m FixtureMeta
	b, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse fixture.json: %w", err)
	}
	return m, nil
}

func runFixture(fx fixture, globalModes map[string]bool) error {
	modes := modesForFixture(fx.Meta, globalModes)
	if modes["fmt"] && !fx.Meta.SkipFmt {
		if err := checkFmtIdempotent(fx.Path); err != nil {
			return err
		}
	}
	entry := fx.Path
	if fx.Meta.Entry != "" {
		entry = filepath.Join(fx.Path, fx.Meta.Entry)
	}
	var proj *project.Project
	if modes["check"] || modes["inspect"] {
		p, err := project.Load(entry)
		if !expectSuccess(fx.Meta) {
			if err == nil {
				return fmt.Errorf("expected failure, got success")
			}
			if fx.Meta.ErrorContains != "" && !strings.Contains(err.Error(), fx.Meta.ErrorContains) {
				return fmt.Errorf("error %q does not contain %q", err.Error(), fx.Meta.ErrorContains)
			}
			return nil
		}
		if err != nil {
			return err
		}
		proj = p
	}
	if modes["inspect"] && expectSuccess(fx.Meta) && (fx.Meta.InspectGo || len(fx.Meta.ExpectedModules) > 0) {
		out, err := os.MkdirTemp("", "kiro-compat-gen-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(out)
		if err := codegen.EmitProjectGo(proj, out); err != nil {
			return err
		}
		for _, m := range fx.Meta.ExpectedModules {
			modFile := filepath.Join(out, "src", m+".go")
			if _, err := os.Stat(modFile); err != nil {
				return fmt.Errorf("missing generated module %s: %w", m, err)
			}
		}
	}
	return nil
}

func expectSuccess(meta FixtureMeta) bool {
	if meta.ExpectSuccess == nil {
		return true
	}
	return *meta.ExpectSuccess
}

func modesForFixture(meta FixtureMeta, global map[string]bool) map[string]bool {
	if len(meta.Modes) == 0 {
		return global
	}
	m := map[string]bool{}
	for _, mode := range meta.Modes {
		m[mode] = global[mode]
	}
	return m
}

func checkFmtIdempotent(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".ki") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		once, err := format.Source(string(b))
		if err != nil {
			return fmt.Errorf("fmt pass 1 for %s: %w", path, err)
		}
		twice, err := format.Source(once)
		if err != nil {
			return fmt.Errorf("fmt pass 2 for %s: %w", path, err)
		}
		if once != twice {
			return fmt.Errorf("non-idempotent formatting for %s", path)
		}
		return nil
	})
}
