package project

import (
	"github.com/kiro-lang/kiro/internal/ast"
	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"sort"
)

// Keep semantic checking independent of generated Go and the build pipeline.
func validateFunctionalContracts(p *Project) error {
	spec := runtimekit.ProgramSpec{EntryModule: p.Files[0].AST.Module}
	modules := map[string]*runtimekit.ModuleSpec{}
	for _, file := range p.Files {
		mod := modules[file.AST.Module]
		if mod == nil {
			mod = &runtimekit.ModuleSpec{Name: file.AST.Module}
			modules[file.AST.Module] = mod
		}
		for _, declaration := range file.AST.Decls {
			switch d := declaration.(type) {
			case ast.FuncDecl:
				fn := runtimekit.FuncSpec{Module: mod.Name, Name: d.Name, ReturnType: d.ReturnType, Body: d.Body, BlockBody: d.BlockBody, Effects: d.CanonicalEffectNames()}
				if d.Receiver != nil {
					fn.ReceiverType = d.Receiver.Type
					fn.Params = append(fn.Params, runtimekit.ParamSpec{Name: d.Receiver.Name, Type: d.Receiver.Type})
				}
				for _, param := range d.Params {
					fn.Params = append(fn.Params, runtimekit.ParamSpec{Name: param.Name, Type: param.Type, Default: param.Default})
				}
				mod.Funcs = append(mod.Funcs, fn)
			case ast.TypeDecl:
				typ := runtimekit.TypeSpec{Name: d.Name}
				for _, f := range d.Fields {
					typ.Fields = append(typ.Fields, runtimekit.FieldSpec{Name: f.Name, Type: f.Type})
				}
				for _, v := range d.Variants {
					variant := runtimekit.VariantSpec{Name: v.Name}
					for _, f := range v.Fields {
						variant.Fields = append(variant.Fields, runtimekit.FieldSpec{Name: f.Name, Type: f.Type})
					}
					typ.Variants = append(typ.Variants, variant)
				}
				mod.Types = append(mod.Types, typ)
			case ast.ConstDecl:
				mod.Consts = append(mod.Consts, runtimekit.ConstSpec{Name: d.Name, Value: d.Value, ValueKind: d.ValueKind})
			}
		}

	}
	names := []string{}
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		spec.Modules = append(spec.Modules, *modules[name])
	}
	return runtimekit.ValidateFunctionalContracts(spec)
}
