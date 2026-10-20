package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

type SymbolField struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
}
type Symbol struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Module     string                  `json:"module"`
	Kind       string                  `json:"kind"`
	File       string                  `json:"file"`
	Range      *runtimekit.SourceRange `json:"range"`
	Signature  string                  `json:"signature"`
	Doc        []string                `json:"doc"`
	Parameters []SymbolField           `json:"parameters,omitempty"`
	ReturnType string                  `json:"return_type,omitempty"`
	Effects    []string                `json:"effects,omitempty"`
	Fields     []SymbolField           `json:"fields,omitempty"`
}

// Revision is a root-independent hash of sorted relative paths and exact source.
func (p *Project) Revision() string {
	records := [][2]string{}
	for _, f := range p.Files {
		records = append(records, [2]string{f.Rel, f.Src})
	}
	sort.Slice(records, func(i, j int) bool { return records[i][0] < records[j][0] })
	raw, _ := json.Marshal(records)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (p *Project) Symbols() []Symbol {
	out := []Symbol{}
	for _, file := range p.Files {
		for _, decl := range file.AST.Decls {
			s := Symbol{Module: file.AST.Module, File: file.Path, Doc: append([]string{}, decl.GetDoc()...)}
			line, col := 0, 0
			switch d := decl.(type) {
			case ast.FuncDecl:
				s.Name = d.Name
				s.Kind = "function"
				s.Signature = d.Signature()
				s.ReturnType = d.ReturnType
				s.Effects = d.CanonicalEffectNames()
				line, col = d.Line, d.Column
				if d.Receiver != nil {
					s.Kind = "method"
					s.Parameters = append(s.Parameters, SymbolField{Name: d.Receiver.Name, Type: d.Receiver.Type})
				}
				for _, param := range d.Params {
					s.Parameters = append(s.Parameters, SymbolField{Name: param.Name, Type: param.Type, Default: param.Default})
				}
				s.ID = file.AST.Module + "." + s.Name
				if d.Receiver != nil {
					receiver := d.Receiver.Type
					if !strings.Contains(receiver, ".") {
						receiver = file.AST.Module + "." + receiver
					}
					s.ID = receiver + "." + s.Name
				}
			case ast.TypeDecl:
				s.Name = d.Name
				s.Kind = "record"
				if len(d.Variants) > 0 {
					s.Kind = "variant_type"
				}
				s.Signature = "type " + d.Name
				line, col = d.Line, d.Column
				for _, field := range d.Fields {
					s.Fields = append(s.Fields, SymbolField{Name: field.Name, Type: field.Type})
				}
				for _, v := range d.Variants {
					variant := Symbol{ID: file.AST.Module + "." + d.Name + "." + v.Name, Name: v.Name, Module: file.AST.Module, File: file.Path, Kind: "constructor", ReturnType: file.AST.Module + "." + d.Name, Doc: []string{}, Range: runtimekit.DeclarationRange(v.Line, v.Column, len([]rune(v.Name)))}
					parts := []string{}
					for _, field := range v.Fields {
						variant.Parameters = append(variant.Parameters, SymbolField{Name: field.Name, Type: field.Type})
						parts = append(parts, field.Name+":"+field.Type)
					}
					variant.Signature = variant.ID + "(" + strings.Join(parts, ",") + ")->" + variant.ReturnType
					out = append(out, variant)
				}
			case ast.ConstDecl:
				s.Name = d.Name
				s.Kind = "constant"
				s.Signature = "const " + d.Name
				line, col = d.Line, d.Column
			}
			if s.ID == "" {
				s.ID = file.AST.Module + "." + s.Name
			}
			s.Range = runtimekit.DeclarationRange(line, col, len([]rune(s.Name)))
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Range.Start.Line < out[j].Range.Start.Line
	})
	return out
}
