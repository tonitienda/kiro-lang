package ast

type File struct {
	Module  string
	Imports []string
	Decls   []Decl
}

type Decl interface {
	declNode()
	GetDoc() []string
}

type ConstDecl struct {
	Doc       []string
	Name      string
	Value     string
	ValueKind string
}

func (d ConstDecl) GetDoc() []string { return d.Doc }
func (ConstDecl) declNode()          {}

type TypeDecl struct {
	Doc    []string
	Name   string
	Fields []Field
}

func (d TypeDecl) GetDoc() []string { return d.Doc }
func (TypeDecl) declNode()          {}

type Field struct {
	Name string
	Type string
}

type EffectDecl struct {
	Name   string
	Line   int
	Column int
}

type FuncDecl struct {
	Doc        []string
	Name       string
	Receiver   *Param
	Params     []Param
	ReturnType string
	Effects    []EffectDecl
	BlockBody  bool
	Body       string
	Line       int
	Column     int
}

func (d FuncDecl) GetDoc() []string { return d.Doc }
func (FuncDecl) declNode()          {}

type Param struct {
	Name string
	Type string
}
