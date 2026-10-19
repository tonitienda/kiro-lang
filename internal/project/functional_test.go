package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctionalChecking(t *testing.T) {
	const prelude = `mod main
import list
import json
type Box { value:i32 }
fn add(value:i32,amount:i32 = 1)->i32 { return value + amount }
fn apply(value:i32,transform:Fn[(value:i32)->i32])->i32 { return transform(value:value) }
fn print_number(value:i32)->nil !io { println(value:value)
return nil }
fn label(item:i32)->i32 {return item}
fn wrong(value:i32)->str {return "wrong"}
`
	for _, tc := range []struct{ name, source, failure string }{
		{"partial_factory", `fn offset(amount:i32)->Fn[(value:i32)->i32] {return partial add(amount:amount)}`, ""},
		{"latent_effect", `fn output(value:i32)->Fn[()->nil!io] {return partial print_number(value:value)}`, ""},
		{"default", `fn run()->i32 {return add(value:2,amount:default)}`, ""},
		{"negative_default", `fn number(value:i32 = -2)->i32 {return value}`, ""},
		{"nullable_default", `fn label_or_nil(value:?str = nil)->?str {return value}`, ""},
		{"bool_default", `fn enabled(value:bool = true)->bool {return value}`, ""},
		{"generic_partial", `fn decoder(text:str)->Fn[()->R[Box,str]] {return partial json.decode[Box](text:text)}`, ""},
		{"missing_key", `fn run()->i32 {return add(value:1)}`, "missing argument"},
		{"unknown_partial", `fn run()->nil {let f = partial add(other:1)
return nil}`, "unknown argument"},
		{"empty_partial", `fn run()->nil {let f = partial add()
return nil}`, "at least one"},
		{"nonliteral_default", `fn run(value:i32 = add(value:1,amount:1))->i32 {return value}`, "scalar literals"},
		{"wrong_default_type", `fn run(value:i32 = "wrong")->i32 {return value}`, "expected i32"},
		{"interpolated_default", `fn run(value:str = "${add(value:1,amount:1)}")->str {return value}`, "without interpolation"},
		{"no_default", `fn run()->i32 {return add(value:default,amount:1)}`, "no declared default"},
		{"default_expression", `fn run()->nil {let n = default
return nil}`, "only valid"},
		{"argument_label", `fn run()->i32 {return apply(value:1,transform:label)}`, "parameter value:i32"},
		{"argument_return", `fn run()->i32 {return apply(value:1,transform:wrong)}`, "return type"},
		{"argument_effect", `fn accepts(f:Fn[(value:i32)->nil])->nil {return nil}
fn run()->nil {return accepts(f:print_number)}`, "effect !io"},
		{"return_effect", `fn output(value:i32)->Fn[()->nil] {return partial print_number(value:value)}`, "effect !io"},
		{"effectful_capture", `fn run()->nil !io {let f = partial add(amount:print_number(value:1))
return nil}`, "effect !io"},
		{"typed_alias_effect", `fn run(f:Fn[(value:i32)->nil!io])->nil {let alias = f
return alias(value:1)}`, "effect !io"},
		{"partial_alias_keys", `fn run()->i32 {let f = partial add(amount:1)
return f(other:2)}`, "unknown argument"},
		{"pure_map", `fn run()->nil !io {list.map(items:[],transform:print_number)
return nil}`, "pure callbacks"},
		{"predicate", `fn run()->nil {list.filter(items:[],predicate:partial add(amount:1))
return nil}`, "return bool"},
		{"callback_label", `fn run()->nil {list.map(items:[],transform:label)
return nil}`, "unknown argument"},
		{"immutable", `fn run()->nil {let n = 1
n = 2
return nil}`, "immutable"},
		{"mut_type", `fn run()->nil {mut n = 1
n = "bad"
return nil}`, "changes type"},
		{"known_return", `fn run()->i32 {return "bad"}`, "return type"},
		{"duplicate_type_key", `fn run(f:Fn[(value:i32,value:i32)->i32])->nil {return nil}`, "duplicate"},
		{"nested_bad_type", `fn run(f:List[Fn[wrong]])->nil {return nil}`, "expected"},
		{"invalid_type_effect", `fn run(f:Fn[()->nil!invented])->nil {return nil}`, "invalid"},
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
				t.Fatalf("want %q; got %v", tc.failure, err)
			}
		})
	}
}

func TestFunctionFieldContracts(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.ki"), `mod main
import list
type Config { transform:Fn[(value:i32)->i32] }
fn apply(config:Config)->List[i32] {
return list.map(items:[1,2],transform:config.transform)
}
fn subtract(value:i32,amount:i32)->i32 {return value - amount}
fn call(f:Fn[(amount:i32,value:i32)->i32])->i32 {return f(value:7,amount:2)}
fn main()->i32 {return call(f:subtract)}
`)
	if _, err := Load(dir); err != nil {
		t.Fatal(err)
	}
}
