package lsp

import (
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/parser"
	"github.com/kiro-lang/kiro/internal/project"
)

func utf16Column(text string, column int) int {
	runes := []rune(text)
	index := max(0, min(column, len(runes)))
	return len(utf16.Encode(runes[:index]))
}
func lspPosition(text string, p runtimekit.SourcePosition) Position {
	lines := strings.Split(text, "\n")
	line := max(0, p.Line-1)
	if line >= len(lines) {
		return Position{Line: max(0, len(lines)-1), Character: 0}
	}
	return Position{Line: line, Character: utf16Column(lines[line], p.Column-1)}
}
func diagnosticFromSource(err error, text string) Diagnostic {
	d := runtimekit.DiagnosticOf(err, "KIRO_CHECK_FAILED")
	span := Range{Start: Position{}, End: Position{Character: 1}}
	if d.Range != nil {
		span = Range{Start: lspPosition(text, d.Range.Start), End: lspPosition(text, d.Range.End)}
	}
	return Diagnostic{Range: span, Severity: 1, Source: "kiro", Code: d.Code, Message: d.Message, Data: d}
}
func (s *Server) entryFor(path string) string {
	if s.root != "" {
		relative, err := filepath.Rel(s.root, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			chosen := ""
			for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
				if info, err := os.Stat(filepath.Join(dir, "main.ki")); err == nil && !info.IsDir() {
					chosen = dir
				}
				if dir == s.root || filepath.Dir(dir) == dir {
					break
				}
			}
			if chosen != "" {
				return chosen
			}
		}
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}
func (s *Server) projectDiagnostics() []Response {
	overlays := map[string]string{}
	uris := []string{}
	for uri, doc := range s.docs {
		uris = append(uris, uri)
		if strings.HasPrefix(uri, "file://") {
			overlays[PathFromURI(uri)] = doc.Text
		}
	}
	sort.Strings(uris)
	diagnostics := map[string][]Diagnostic{}
	checked := map[string]bool{}
	for _, uri := range uris {
		doc := s.docs[uri]
		path := PathFromURI(uri)
		entry := ""
		if strings.HasPrefix(uri, "file://") {
			entry = s.entryFor(path)
		}
		var err error
		if entry != "" {
			key := entry
			if info, e := os.Stat(entry); e == nil && !info.IsDir() {
				key = filepath.Dir(entry)
			}
			if checked[key] {
				continue
			}
			checked[key] = true
			_, err = project.LoadOverlay(entry, overlays)
		} else {
			file, e := parser.Parse(doc.Text)
			fallback := "KIRO_SYNTAX"
			err = e
			if err == nil {
				fallback = "KIRO_CHECK_FAILED"
				p := &project.Project{Root: filepath.Dir(path), Entry: path, Files: []project.File{{Path: path, Rel: filepath.Base(path), Src: doc.Text, AST: file}}}
				err = p.Validate()
			}
			if err != nil {
				d := runtimekit.DiagnosticOf(err, fallback)
				if d.File == "" {
					err = project.SourceDiagnostic(path, doc.Text, err, fallback)
				}
			}
		}
		if err == nil {
			continue
		}
		d := runtimekit.DiagnosticOf(err, "KIRO_PROJECT")
		target, text := uri, doc.Text
		if d.File != "" && d.File != path {
			if source, exists := overlays[d.File]; exists {
				text = source
			} else if raw, e := os.ReadFile(d.File); e == nil {
				text = string(raw)
			}
			target = (&url.URL{Scheme: "file", Path: d.File}).String()
		}
		diagnostics[target] = append(diagnostics[target], diagnosticFromSource(err, text))
	}
	publish := map[string]bool{}
	for uri := range s.published {
		publish[uri] = true
	}
	for _, uri := range uris {
		publish[uri] = true
	}
	for uri := range diagnostics {
		publish[uri] = true
	}
	names := []string{}
	for uri := range publish {
		names = append(names, uri)
	}
	sort.Strings(names)
	responses := []Response{}
	s.published = map[string]bool{}
	for _, uri := range names {
		list := diagnostics[uri]
		if list == nil {
			list = []Diagnostic{}
		} else {
			s.published[uri] = true
		}
		responses = append(responses, Response{JSONRPC: "2.0", Method: "textDocument/publishDiagnostics", Params: map[string]any{"uri": uri, "diagnostics": list}})
	}
	return responses
}
