package ast

type File struct {
	Module  string
	Imports []string
	Decls   []Decl
}

type Decl interface {
	declNode()
}

type ConstDecl struct {
	Name      string
	Value     string
	ValueKind string
}

func (ConstDecl) declNode() {}

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
	Receiver   *Param
	Params     []Param
	ReturnType string
	BlockBody  bool
	Body       string
}

func (FuncDecl) declNode() {}

type Param struct {
	Name string
	Type string
}
