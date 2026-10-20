package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVariantContracts(t *testing.T) {
	const prelude = `mod main
import json
import test
type Failure {
 variant Missing { key:str }
 variant Invalid { amount:i32 note:?str }
 variant Unavailable {}
}
fn leaf()->R[i32,Failure] {return Err(error:Failure.Missing(key:"item"))}
`
	for _, tc := range []struct{ name, source, failure string }{
		{"constructor", `fn run()->Failure {return Failure.Invalid(note:nil,amount:2)}`, ""},
		{"constructor_missing", `fn run()->Failure {return Failure.Invalid(amount:2)}`, "missing argument"},
		{"constructor_unknown_key", `fn run()->Failure {return Failure.Missing(other:"x")}`, "unknown argument"},
		{"constructor_wrong_type", `fn run()->Failure {return Failure.Missing(key:2)}`, "expected str"},
		{"constructor_unknown_tag", `fn run()->Failure {return Failure.Typo()}`, "unknown variant"},
		{"record_constructor", `fn run()->Failure {return Failure{}}`, "requires a named constructor"},
		{"duplicate_tag", `type Bad {variant A {} variant A {}}`, "duplicate variant"},
		{"duplicate_payload", `type Bad {variant A {value:i32 value:str}}`, "duplicate field"},
		{"mixed_record", `type Bad {value:i32 variant A {}}`, "cannot mix"},
		{"exhaustive", `fn run(error:Failure)->str {return when (error) {
 Failure.Missing(key:let key) => key
 Failure.Invalid(amount:_,note:_) => "invalid"
 Failure.Unavailable() => "offline"
}}`, ""},
		{"missing_arm", `fn run(error:Failure)->nil {when (error) {Failure.Missing(key:_) => {return nil}}}`, "missing Invalid, Unavailable"},
		{"wildcard", `fn run(error:Failure)->nil {when (error) {_ => {return nil}}}`, "require explicit exhaustive arms"},
		{"partial_payload_coverage", `fn run(error:Failure)->nil {when (error) {
 Failure.Missing(key:"only") => {return nil}
 Failure.Invalid(amount:_,note:_) => {return nil}
 Failure.Unavailable() => {return nil}
}}`, "missing Missing"},
		{"duplicate_arm", `fn run(error:Failure)->nil {when (error) {
 Failure.Missing(key:_) => {return nil}
 Failure.Missing(key:_) => {return nil}
}}`, "unreachable duplicate variant"},
		{"unknown_pattern_key", `fn run(error:Failure)->nil {when (error) {Failure.Missing(nope:_) => {return nil}}}`, "unknown argument"},
		{"duplicate_binder", `fn run(error:Failure)->nil {when (error) {Failure.Invalid(amount:let x,note:let x) => {return nil}}}`, "duplicate pattern binding"},
		{"immutable_binder", `fn run(error:Failure)->nil {when (error) {Failure.Missing(key:let key) => {key = "new"}}}`, "immutable"},
		{"binder_type", `fn run(error:Failure)->nil {when (error) {Failure.Missing(key:let key) => {test.eq(actual:1,expected:1)
return key}}}`, "return type"},
		{"bare_binder", `fn run(error:Failure)->nil {when (error) {Failure.Missing(key:key) => {return nil}}}`, "use let key"},
		{"binding_outside_pattern", `fn run()->str {return let key}`, "only valid in when patterns"},
		{"no_implicit_conversion", `fn run()->R[i32,str] {return Ok(value:leaf()? )}`, "incompatible with enclosing error type"},
		{"result_complete", `fn run()->i32 {return when (leaf()) {Ok(value:let n) => n
Err(error:_) => 0}}`, ""},
		{"result_missing", `fn run()->nil {when (leaf()) {Ok(value:_) => {return nil}}}`, "missing Err"},
		{"result_wrong_pattern", `fn run()->nil {when (1) {Ok(value:_) => {return nil}}}`, "requires a known R"},
		{"foreign_variant", `type Another {variant Missing {key:str}}
fn run(error:Failure)->nil {when (error) {Another.Missing(key:_) => {return nil}}}`, "belongs to"},
		{"block_return_expression", `fn run(error:Failure)->str {return when (error) {
Failure.Missing(key:let key) => {return key}
Failure.Invalid(amount:_,note:_) => {return "bad"}
Failure.Unavailable() => {return "offline"}
}}`, ""},
		{"wrong_literal", `fn run()->nil {when (leaf()) {Ok(value:"bad") => {return nil}}}`, "pattern type str"},
		{"arm_types", `fn run()->i32 {return when (leaf()) {Ok(value:_) => 1
Err(error:_) => "bad"}}`, "arm types differ"},
		{"pattern_call_effect", `fn run()->nil {when (1) {test.eq(actual:1,expected:1) => {return nil}}}`, "unknown variant pattern"},
		{"binder_scope", `fn run()->str {when (leaf()) {Ok(value:let n) => {test.eq(actual:n,expected:1)}
Err(error:_) => {return "bad"}}
return n}`, "unknown identifier"},
		{"loop_list", `fn run()->i32 {mut n = 0
for value in [1,2,3] {n = n + value}
return n}`, ""},
		{"loop_nonlist", `fn run()->nil {for value in 3 {return nil}}`, "requires a known List"},
		{"loop_element_type", `fn run()->nil {for value in [1] {test.eq(actual:1,expected:1)
let e = Failure.Missing(key:value)}
return nil}`, "expected str"},
		{"loop_binding_immutable", `fn run()->nil {for value in [1] {value = 2}
return nil}`, "immutable"},
		{"unsupported_body", `fn run()->nil {while true {return nil}}`, "body:"},
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

func TestNewVariantInvalidatesEachHandler(t *testing.T) {
	const base = `mod main
type Failure {variant Missing {key:str}}
fn first(error:Failure)->str {return when (error) {Failure.Missing(key:let key) => key}}
fn second(error:Failure)->i32 {return when (error) {Failure.Missing(key:_) => 404}}
`
	for _, handler := range []string{"first", "second"} {
		t.Run(handler, func(t *testing.T) {
			dir := t.TempDir()
			source := base
			if handler == "first" {
				source = strings.Split(source, "fn second")[0]
			} else {
				start := strings.Index(source, "fn first")
				end := strings.Index(source, "fn second")
				source = source[:start] + source[end:]
			}
			file := filepath.Join(dir, "main.ki")
			mustWrite(t, file, source)
			if _, err := Load(dir); err != nil {
				t.Fatal(err)
			}
			source = strings.Replace(source, "variant Missing {key:str}", "variant Missing {key:str} variant Timeout {}", 1)
			mustWrite(t, file, source)
			_, err := Load(dir)
			if err == nil || !strings.Contains(err.Error(), "main."+handler) || !strings.Contains(err.Error(), "missing Timeout") {
				t.Fatalf("expected new-variant diagnostic at %s, got %v", handler, err)
			}
		})
	}
}
