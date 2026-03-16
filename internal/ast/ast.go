package ast

type File struct {
	Module  string
	Imports []string
	Decls   []Decl
}

type Decl interface {
	declNode()
}

type TypeDecl struct {
	Name   string
	Fields []Field
}

func (TypeDecl) declNode() {}

type Field struct {
	Name string
	Type string
}

type FuncDecl struct {
	Name       string
	Params     []Param
	ReturnType string
	Body       string
}

func (FuncDecl) declNode() {}

type Param struct {
	Name string
	Type string
}
