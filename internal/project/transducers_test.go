package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTransducerChecking(t *testing.T) {
	const prelude = `mod main
import list
fn increment(value:i32)->i32 {return value + 1}
fn text(value:i32)->str {return "${value}"}
fn keep(value:i32)->bool {return value > 0}
fn sum(state:i32,value:i32)->i32 {return state + value}
fn bad_state(state:i32,value:i32)->str {return "wrong"}
fn emit(value:i32)->nil !io {println(value:value)
return nil}
fn role(item:i32)->i32 {return item}
`
	for _, tc := range []struct{ name, source, failure string }{
		{"typed_factory", `fn pipeline()->Transform[i32,str] {return list.compose(first:list.filtering(predicate:keep),then:list.mapping(transform:text))}`, ""},
		{"typed_parameter", `fn run(through:Transform[i32,str])->List[str] {return list.collect(items:[1],through:through)}`, ""},
		{"aliases", `fn run()->List[i32] {let mapping = list.mapping
let compose = list.compose
let xf = compose(first:mapping(transform:increment),then:mapping(transform:increment))
return list.collect(items:[1],through:xf)}`, ""},
		{"transduce", `fn run()->i32 {return list.transduce(items:[1,2],through:list.mapping(transform:increment),initial:0,reduce:sum)}`, ""},
		{"partial_transduce", `fn run()->i32 {let run = partial list.transduce(through:list.mapping(transform:increment),initial:0,reduce:sum)
return run(items:[1,2])}`, ""},
		{"reduce_name", `fn run()->i32 {return list.reduce(items:[],initial:0,reduce:sum)}`, ""},
		{"fold_migration", `fn run()->i32 {return list.fold(items:[],initial:0,reduce:sum)}`, "renamed to list.reduce"},
		{"fold_alias_migration", `fn run()->nil {let old = list.fold
return nil}`, "renamed to list.reduce"},
		{"mapping_role", `fn run()->nil {list.mapping(transform:role)
return nil}`, "exactly the value key"},
		{"mapping_effect", `fn run()->nil !io {list.mapping(transform:emit)
return nil}`, "pure callbacks"},
		{"filter_result", `fn run()->nil {list.filtering(predicate:increment)
return nil}`, "return bool"},
		{"stage_types", `fn run()->nil {list.compose(first:list.mapping(transform:text),then:list.mapping(transform:increment))
return nil}`, "cannot pass str output to i32 input"},
		{"bad_through", `fn run()->List[i32] {return list.collect(items:[],through:increment)}`, "must be a Transform"},
		{"bad_first", `fn run()->nil {list.compose(first:1,then:list.mapping(transform:text))
return nil}`, "must be Transform values"},
		{"bad_input", `fn run()->List[i32] {return list.collect(items:["wrong"],through:list.mapping(transform:increment))}`, "input expects i32"},
		{"bad_reducer_input_empty", `fn run()->i32 {return list.transduce(items:[],through:list.mapping(transform:text),initial:0,reduce:sum)}`, "cannot accept str output"},
		{"bad_reducer_state_empty", `fn run()->nil {list.transduce(items:[],through:list.mapping(transform:increment),initial:0,reduce:bad_state)
return nil}`, "match state type"},
		{"bad_initial_empty", `fn run()->i32 {return list.transduce(items:[],through:list.mapping(transform:increment),initial:"bad",reduce:sum)}`, "initial"},
		{"bad_transform_return", `fn pipeline()->Transform[i32,str] {return list.mapping(transform:increment)}`, "return type"},
		{"bad_transform_arity", `fn pipeline(x:Transform[i32])->nil {return nil}`, "requires two types"},
		{"bad_transform_type", `fn pipeline(x:Transform[i32,Fn[bad]])->nil {return nil}`, "expected"},
		{"missing_through", `fn run()->nil {list.collect(items:[])
return nil}`, "missing argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, "main.ki"), prelude+tc.source+"\n")
			_, err := Load(dir)
			if tc.failure == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("want %q, got %v", tc.failure, err)
			}
		})
	}
}
