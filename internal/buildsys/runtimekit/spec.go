package runtimekit

import (
	"encoding/json"
	"fmt"
	"os"
)

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

func DecodeSpec(raw string) (ProgramSpec, error) {
	var spec ProgramSpec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return ProgramSpec{}, err
	}
	return spec, nil
}

func Main(raw string) int {
	spec, err := DecodeSpec(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode generated spec: %v\n", err)
		return 1
	}
	rt, err := newRuntime(spec, os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	code, err := rt.run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return code
}
