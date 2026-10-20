# Kiro language reference

Reference schema: `kiro.language-reference.v1`. Guide revision: `phase19-language-reference`.
This document specifies the current canonical source language, including its implementation
limits. It is the primary language reference shipped with the compiler, not a proposal for
future syntax. Read `../SKILL.md` first for a short working guide; use `kiro.json` for compact
capabilities and exact builtin keys. Historical phase notes describe changes, not competing
syntax. No training-time knowledge of Kiro is assumed.

If the reference and a particular compiler disagree, record the compiler commit, guide
revision, source and diagnostic as a conformance defect. Do not silently invent syntax or
treat compiler acceptance as proof of intended behavior. Checking is incremental; this is
an executable experimental contract, not a formal soundness/completeness proof.

## Contents

1. [Source and project model](#source-and-project-model)
2. [Declarations and types](#declarations-and-types)
3. [Bindings, scope and evaluation](#bindings-scope-and-evaluation)
4. [Calls, defaults and function values](#calls-defaults-and-function-values)
5. [Results and optional values](#results-and-optional-values)
6. [Dispatch and patterns](#dispatch-and-patterns)
7. [Finite loops](#finite-loops)
8. [Collections and transforms](#collections-and-transforms)
9. [Effects and concurrency](#effects-and-concurrency)
10. [JSON and external data](#json-and-external-data)
11. [Builtin API](#builtin-api)
12. [Execution, tools and unsupported forms](#execution-tools-and-unsupported-forms)
13. [Conformance examples](#conformance-examples)

## Source and project model

Files are UTF-8 text with extension `.ki`. The first declaration is `mod name` with a
single identifier, followed by imports and declarations. Use letters/underscore followed
by letters/digits/underscore for identifiers; prefer ASCII identifiers for portability.
Names are case-sensitive. `//` introduces a line comment; `///` documents the next top-level
const/type/function declaration. Do not use block comments or semicolon terminators.

Write one statement per line, braces around blocks, commas between call arguments and list
items, and commas between record-literal fields. Declaration fields use whitespace/newlines,
not commas. A colon separates each name from its type/value. Format with `kiro fmt`; spacing
in this reference is for readability and does not replace the formatter.

A directory entry must contain `main.ki`. A file entry makes its containing directory the
project root. The loader reads all `.ki` files recursively beneath the root, excluding hidden
directories; it does not load only reachable imports. Unused source files must still check.
Keep unrelated projects outside that root. The scaffold's hidden `.kiro/skill/` examples are
therefore documentation, not application modules.

`import name` resolves a builtin or a declared project module. Slash paths such as
`import internal/pricing` also resolve matching file paths or directories containing
`main.ki`. Calls and qualified types use the declared module name, e.g. `pricing.quote`,
not the slash path. Prefer unique module names and unique declarations within a module;
overloading, package privacy and import aliases are not a supported design surface.
The current runtime can resolve some globally unique names without an import: do not rely
on that fallback. Write imports and qualify cross-module symbols explicitly.

Preferred service layout: startup in `main.ki`, handlers in `app/`, configuration in
`internal/config/`, and `test_*` functions in `test/`. This is a convention, not a grammar rule.

## Declarations and types

The following EBNF describes canonical forms, omitting whitespace and expression precedence.
It is a guide to the supported surface, not a standalone parser implementation. `ident` is
an identifier, `qualified` is dot-separated identifiers, braces in quotes are source tokens,
and `{ X }` outside quotes means repetition. Newlines separate statements/arms.

```ebnf
file        = "mod" ident { "import" import-path | declaration } ;
import-path = ident { "/" ident } ;
declaration = constant | record | variant | function ;
constant    = "const" ident "=" (string | unsigned-integer) ;
record      = "type" ident "{" { ident ":" type } "}" ;
variant     = "type" ident "{" variant-arm { variant-arm } "}" ;
variant-arm = "variant" ident "{" { ident ":" type } "}" ;
function    = "fn" [ "(" ident ":" type ")" ] ident
              "(" [ parameter { "," parameter } ] ")" "->" type
              { "!" effect } block ;
parameter   = ident ":" type [ "=" scalar-literal ] ;
type        = "?" type | qualified | "List[" type "]" | "R[" type "," type "]"
            | "Transform[" type "," type "]"
            | "Fn[(" [ fn-parameter { "," fn-parameter } ] ")->" type { "!" effect } "]" ;
fn-parameter = ident ":" type ;
block       = "{" { statement } "}" ;
statement   = ("let" | "mut") ident "=" expression | ident "=" expression
            | "return" expression | expression | "group" block
            | "for" ident "in" expression block | dispatch ;
dispatch    = "when" "(" expression ")" "{" { pattern "=>" (block | expression) } "}" ;
call        = expression [ "[" type { "," type } "]" ]
              "(" [ ident ":" expression { "," ident ":" expression } ] ")" ;
```

`const` is intentionally narrow. The frontend also accepts an identifier initializer, but
it is stored as identifier text, not resolved as an alias; use literal strings/unsigned
integers. General constant expressions, negative const initializers and user-defined
generic declarations are not part of the canonical surface. Generic syntax in calls is
for supported builtin specialization, notably `json.decode[T]`, not user-defined generics.

| Type | Contract |
| --- | --- |
| `i32` | Signed 32-bit integer, from -2147483648 to 2147483647 |
| `str` | String; quoted literals support Go-style escapes and `${expression}` interpolation |
| `bool` | `true` or `false` |
| `nil` | The nil value; also used as the current void-like function return type |
| `?T` | A value of T or nil; absence is separate from failure |
| `List[T]` | Finite list, literal `[a, b]`; an empty literal is `[]` |
| `R[T,E]` | Tagged success `Ok(value:t)` or failure `Err(error:e)` |
| Record/variant name | Nominal declared type; qualify ambiguous/cross-module names |
| `Fn[(key:T)->U!effect]` | Key-preserving callable contract; zero or more latent effects |
| `Transform[T,U]` | Pure, reusable mapping/filtering plan |
| `Any` | Dynamic escape hatch; offers no concrete data-schema guarantee |
| `http.Req`, `http.Resp`, `ctx.Ctx` | Builtin runtime values; see the API and limitations below |

There are no implicit string/integer conversions. Records contain all declared fields,
including explicit nil for absent optional fields. Read a field with `value.field`.
The current language has no general field/index assignment; rebuild a record and rebind a
`mut` local. Parameters and return types are explicit; local binding types are inferred.
Record and variant declarations cannot mix fields and variants. Empty records are allowed.

Methods use `fn (self:Type) name(key:T) -> U { ... }` and are called as
`receiver.name(key:value)`; do not pass the receiver again as a keyed argument.

```ki example=records expect=pass
mod main
import test
const Version = "1"
type User { name:str email:?str }
fn (self:User) label() -> str {
  return "${self.name}@${Version}"
}
fn main() -> i32 { return 0 }
fn test_record() -> nil {
  let user = User{name:"Ada", email:nil}
  test.eq(actual:user.label(), expected:"Ada@1")
  return nil
}
```

## Bindings, scope and evaluation

`let name = expression` introduces an immutable binding. `mut name = expression` introduces
one that can be rebound with `name = expression`. Parameters, pattern binders and loop items
are immutable. Loop and dispatch-arm locals do not escape their blocks; outer mutable updates
do. Shadowing inside an arm or loop is local. `group` currently executes in the surrounding
local scope; it does not create a new isolation or ownership boundary.

Use expressions built from literals, names, selection, record/list construction, calls,
parentheses, dispatch, `partial`, `spawn`, `await`, result unwrapping and the operators below.
A negative integer literal is supported; general unary negation of an expression is not
(use `0 - value`). There is no source floating-point literal, logical `!`, `&&`/`||`, modulo,
indexing or slicing. Express boolean decisions with `when` and list traversal with `for`.

| Precedence, highest first | Operators/forms | Evaluation |
| --- | --- | --- |
| Postfix | selection `.`, call `(...)`, typed call `[...] (...)`, unwrap `?` | Chain left to right; parenthesize compound operands |
| Multiplicative | `*`, `/` | Left associative integer arithmetic |
| Additive | `+`, `-` | Left associative; string + string concatenates |
| Ordered comparison | `<`, `<=`, `>`, `>=` | Numeric operands, boolean result |
| Equality | `==`, `!=` | Structural value comparison, boolean result |

Binary operands evaluate left then right. Lists and record-literal field expressions evaluate
in source order. Arithmetic detects i32 overflow and division by zero; division truncates
toward zero. These are execution failures, not domain `Err` values. Use `test.eq` for structural
assertions; do not depend on implicit coercion or runtime handle identity.

Quoted strings cannot contain literal newlines; use `\n`, `\"`, `\\`, or Unicode escapes.
Interpolation evaluates original source placeholders once, in order; inserted text containing
`${...}` is data and is never re-evaluated. There is no raw-string or multiline-string form.

## Calls, defaults and function values

Every value argument uses the declaration's exact key. All parameters must appear once,
including optional ones. Keys may appear in any order; expressions execute once in source
order, then values bind to parameters by key. Key-set validation occurs before argument
execution. Parameterless calls use `f()`. Generic type arguments remain positional.
No omitted keys, positional arguments, shorthand keys, implicit currying or argument spreading.

A declaration default is a scalar literal: signed integer, string without interpolation,
boolean, or compatible nil. Select it explicitly with `key:default`; omission is still an
error. A default marker is valid only as an entire keyed argument. Defaults cannot depend on
other parameters, function calls or configuration.

Functions are first-class named values. `Fn` parameter labels and types are part of their
contract; label order may differ, but sets/types must agree. Pure functions may satisfy an
effectful bound; effectful functions cannot satisfy a pure bound. A `Fn` type does not promise
that a default exists, so callers through a typed parameter must provide concrete arguments.
There are no anonymous functions or nested function declarations.

`partial f(key:value)` binds at least one key and returns a function with the remaining keys.
Binding every key creates a thunk, called with `f()`. Partials can be specialized again.
The callee and capture expressions must be pure and evaluate once at construction; records,
lists, partials and method receivers are snapshotted. Later rebinding does not change them.
Each invocation gets fresh captured data. Futures/unsupported runtime handles cannot be
captured. The target's latent effects are required when invoked, not when constructing it.

```ki example=functions expect=pass
mod main
import test
fn add(value:i32, amount:i32 = 1) -> i32 { return value + amount }
fn apply(value:i32, transform:Fn[(value:i32)->i32]) -> i32 {
  return transform(value:value)
}
fn main() -> i32 { return 0 }
fn test_functions() -> nil {
  mut amount = 3
  let offset = partial add(amount:amount)
  amount = 20
  test.eq(actual:apply(value:4, transform:offset), expected:7)
  test.eq(actual:add(amount:default, value:4), expected:5)
  return nil
}
```

## Results and optional values

`R[T,E]` has explicit success/failure tags, including `Ok(value:nil)` versus `Err(error:nil)`.
A postfix `?` requires a result operand and an enclosing result-returning function with a
compatible error type. Ok yields its payload. Err returns from that function immediately,
without executing its remaining expressions/statements. The success types can differ.
Propagation works inside arguments, interpolation, loops and nested blocks; it stops at
that user-function boundary. It is not an exception handler for assertions, type errors or
arithmetic failures. Convert unrelated error types with an explicit typed function/match.

`?T` means optional T. It is not a result and postfix `?` does not unwrap it. Match nil then
use a wildcard arm for the present value; the checker narrows the matched optional binding
in that arm. Nil/optional/Any functions can fall through to nil in this implementation;
write explicit returns anyway. Other return contracts require an explicit return on every
successful path. A potentially empty loop and a possibly successful `?` do not prove return.

```ki example=results expect=pass
mod main
import test
import parse
fn read(raw:str) -> R[i32,str] { return parse.i32(text:raw) }
fn label(raw:str) -> R[str,str] {
  let number = read(raw:raw)?
  return Ok(value:"id:${number}")
}
fn email(value:?str) -> str {
  return when (value) {
    nil => "absent"
    _ => value
  }
}
fn main() -> i32 { return 0 }
fn test_results() -> nil {
  test.eq(actual:label(raw:"42"), expected:Ok(value:"id:42"))
  test.eq(actual:email(value:nil), expected:"absent")
  test.eq(actual:email(value:"a"), expected:"a")
  test.eq(actual:Ok(value:nil) == Err(error:nil), expected:false)
  return nil
}
```

## Dispatch and patterns

Use `when (subject) { ... }`; the subject evaluates once and arms are tried in source order.
A matching block runs its statements; a matching expression yields its value. In an expression
context, arm value types must agree when known. `return` in a block arm returns from the
enclosing function, not just the dispatch. A primitive statement dispatch with no match does
nothing; a dispatch with no match yields nil, which may later fail a nonoptional type contract. Use a final `_` for total
primitive/value dispatch. The checker does not solve arbitrary primitive constraint coverage.

Patterns are literals, `_`, `let name`, or declared variant/result constructors. Constructor
patterns destructure; they do not invoke functions. Use each payload key and bind with
`key:let name` or ignore with `key:_`. Literal/nested payload patterns constrain values;
strings in patterns are literal text, with no executed interpolation. Do not use function
calls, computed arithmetic or guards as patterns; nest a `when` for extra conditions.

Every tag of a known declared variant needs a total arm (all payloads bound or ignored).
A wildcard catch-all is rejected for declared variants so adding a tag invalidates affected
matches. An arm after a total arm of the same tag is unreachable. Known result matches need
both total Ok/Err arms or a final wildcard. A nested payload pattern is not proof of total
coverage for its outer constructor. Pattern bindings are immutable and local to their arm.

```ki example=variants expect=pass
mod main
import test
type Failure { variant Missing { key:str } variant Offline {} }
fn explain(error:Failure) -> str {
  return when (error) {
    Failure.Missing(key:let key) => "missing:${key}"
    Failure.Offline() => "offline"
  }
}
fn recover(value:R[i32,Failure]) -> str {
  return when (value) {
    Ok(value:let number) => "${number}"
    Err(error:let failure) => explain(error:failure)
  }
}
fn main() -> i32 { return 0 }
fn test_variants() -> nil {
  test.eq(actual:recover(value:Err(error:Failure.Missing(key:"sku"))), expected:"missing:sku")
  test.eq(actual:recover(value:Err(error:Failure.Offline())), expected:"offline")
  test.eq(actual:recover(value:Ok(value:4)), expected:"4")
  return nil
}
```

## Finite loops

`for item in items { ... }` evaluates a finite list once and visits its items in order.
The item is immutable; body locals are scoped per iteration. Outer `mut` rebinding, nested
loops, explicit return and result propagation work. Empty lists execute no iterations.
There are no index/range loops, break/continue, while, implicit collection growth or iterator
protocols. Use an explicit accumulator or pure collection helpers.

```ki example=loops expect=pass
mod main
import test
fn total(items:List[i32]) -> i32 {
  mut sum = 0
  for item in items {
    let contribution = item * 2
    sum = sum + contribution
  }
  return sum
}
fn main() -> i32 { return 0 }
fn test_loops() -> nil {
  test.eq(actual:total(items:[1, 2, 3]), expected:12)
  test.eq(actual:total(items:[]), expected:0)
  return nil
}
```

## Collections and transforms

`list.map(items:xs, transform:f)` and `list.filter(items:xs, predicate:p)` execute eager,
separate full passes. Their pure callbacks use the label `value`; the predicate returns
bool. `list.reduce(items:xs, initial:zero, reduce:r)` is an ordered left fold, with pure
`state` and `value` callback parameters and the same state/return type. Empty reduction
returns its initial value. Contracts are checked even on empty lists. Callback result values
are ordinary values; neither a result mapper nor a reducer returning Err stops iteration.

For explicit fusion, construct pure snapshotted plans with `list.mapping`, `list.filtering`
and `list.compose(first:a, then:b)`. Construction executes no callbacks. `list.collect` or
`list.transduce` executes each input item through its stages in first-then order before
processing the next item. A rejected item skips remaining stages/reduction. A runtime failure
aborts at the first failing item/stage. There are no intermediate data lists; collect builds
its final list. Plan descriptors, input, callback values and accumulator still allocate.
This order can expose a different first error from separate eager passes; fusion is never
implicit. Only finite, pure, stateless mapping/filtering stages are supported.

```ki example=transforms expect=pass
mod main
import test
import list
fn positive(value:i32) -> bool { return value > 0 }
fn increase(value:i32) -> i32 { return value + 1 }
fn sum(state:i32, value:i32) -> i32 { return state + value }
fn plan() -> Transform[i32,i32] {
  return list.compose(first:list.filtering(predicate:positive), then:list.mapping(transform:increase))
}
fn main() -> i32 { return 0 }
fn test_transforms() -> nil {
  test.eq(actual:list.collect(items:[-1, 1, 2], through:plan()), expected:[2, 3])
  test.eq(actual:list.transduce(items:[-1, 1, 2], through:plan(), initial:0, reduce:sum), expected:5)
  test.eq(actual:list.map(items:[1, 2], transform:increase), expected:[2, 3])
  test.eq(actual:list.filter(items:[-1, 2], predicate:positive), expected:[2])
  test.eq(actual:list.reduce(items:[], initial:0, reduce:sum), expected:0)
  return nil
}
```

## Effects and concurrency

Effects use repeated suffixes, e.g. `-> nil !fs !io`. The known set is `env`, `fs`, `io`,
`log`, `net`, `panic`, `proc`, `time`. Unknown/duplicate effect names fail. Pure functions
have no suffix. Callers must allow effects required by callees/typed function values.
Parsing, JSON and assertions do not require pseudo-effects such as `!json` or `!test`.
An effect name being recognized does not imply a corresponding builtin exists: discover
actual operations in the API table. Effect checking is incomplete, especially through
some external callbacks; these declarations are not a security sandbox.

`spawn expression` creates a task, `await task` waits and returns its value or runtime failure,
and `group { ... }` makes the intended grouping visible. Use a named result-returning worker,
await its result and apply `?` in the awaiting function. Inline propagation inside a spawn
expression is rejected. Do not rely on an automatic join or cancellation: `group` currently
executes a block and does not enforce task lifetimes. Always explicitly await your tasks.
There is no ownership/race-freedom guarantee; avoid concurrent mutation/shared runtime handles.
`ctx` helpers are placeholder values and do not implement deadlines or cancellation.

```ki example=concurrency expect=pass
mod main
import test
fn work() -> R[i32,str] { return Ok(value:7) }
fn joined() -> R[i32,str] {
  group {
    let task = spawn work()
    let result = await task
    let value = result?
    return Ok(value:value + 1)
  }
}
fn main() -> i32 { return 0 }
fn test_joined() -> nil {
  test.eq(actual:joined(), expected:Ok(value:8))
  return nil
}
```

## JSON and external data

`json.decode[T](text:source)` requires exactly one explicit schema and returns `R[T,str]`.
Supported schemas include primitives, nil, optionals, lists, records, declared variants and results; not Fn,
Transform or ctx schemas. Unknown schemas are program failures; malformed
or nonconforming data returns Err with a path. JSON decoding rejects duplicate/unknown fields,
missing nonoptional fields, trailing values and wrong nested types. Omitted optional fields
become nil; this differs from source constructors, which require explicit optional keys.

For i32, JSON must use in-range integer number text: `1.0` and `1e0` are rejected. Inputs must
be UTF-8 and nesting is limited to 128 levels. `Any` retains untyped JSON data and exact number
text; do not use it to promise a validated record. JSON encoding hides record implementation
metadata. Results use exactly one `{"Ok":payload}` or `{"Err":payload}` key, even for null
payloads. Encoding runtime callables/transforms/handles fails rather than exposing them.
Declared variants encode as exactly one constructor tag mapping to its payload object,
e.g. `{"Missing":{"key":"sku"}}`; decoding checks that tag and its closed payload schema.

```ki example=json expect=pass
mod main
import test
import json
type Input { amount:i32 note:?str }
fn amount(text:str) -> R[i32,str] {
  let input = json.decode[Input](text:text)?
  return Ok(value:input.amount)
}
fn rejected(text:str) -> bool {
  return when (json.decode[Input](text:text)) {
    Ok(value:_) => false
    Err(error:_) => true
  }
}
fn main() -> i32 { return 0 }
fn test_json() -> nil {
  test.eq(actual:amount(text:"{\"amount\":4}"), expected:Ok(value:4))
  test.eq(actual:rejected(text:"{\"amount\":1.0}"), expected:true)
  test.eq(actual:rejected(text:"{\"amount\":1,\"amount\":2}"), expected:true)
  test.eq(actual:rejected(text:"{\"amount\":1,\"extra\":2}"), expected:true)
  return nil
}
```

## Builtin API

The table below lists the complete executable builtin registry at this revision. Base
contracts use Any for specialization; they do not override callback/schema/transform rules
above. `kiro describe builtin.name --json` gives the same base contract and parameter names.
Tests compare every table signature to the registry. Import module names explicitly in
application code; `print`, `println`, Ok/Err are global. `assert.equal_i32` is a legacy
executable assertion; new code should use `test.eq`.

<!-- builtin-contracts:start -->
| Builtin | Base callable contract |
| --- | --- |
| `Err` | `Fn[(error:Any)->R[Any,Any]]` |
| `Ok` | `Fn[(value:Any)->R[Any,Any]]` |
| `assert.equal_i32` | `Fn[(actual:Any,expected:Any)->nil]` |
| `cli.args` | `Fn[()->List[str]!proc]` |
| `ctx.background` | `Fn[()->ctx.Ctx]` |
| `ctx.with_timeout_ms` | `Fn[(parent:ctx.Ctx,ms:i32)->ctx.Ctx]` |
| `env.get_or` | `Fn[(key:str,fallback:str)->str!env]` |
| `fs.read_file` | `Fn[(path:str)->R[str,str]!fs]` |
| `fs.write_file` | `Fn[(path:str,contents:str)->R[nil,str]!fs]` |
| `http.json` | `Fn[(code:i32,body:str)->http.Resp]` |
| `http.not_found` | `Fn[()->http.Resp]` |
| `http.query` | `Fn[(req:http.Req,key:str)->?str]` |
| `http.serve` | `Fn[(addr:str,handler:Any)->R[nil,str]!net]` |
| `http.test_req` | `Fn[(method:str,path:str,body:str)->http.Req]` |
| `http.text` | `Fn[(code:i32,body:str)->http.Resp]` |
| `http.with_header` | `Fn[(resp:http.Resp,key:str,value:str)->http.Resp]` |
| `json.decode` | `Fn[(text:str)->R[Any,str]]` |
| `json.encode` | `Fn[(value:Any)->R[str,str]]` |
| `list.collect` | `Fn[(items:List[Any],through:Any)->List[Any]]` |
| `list.compose` | `Fn[(first:Any,then:Any)->Any]` |
| `list.filter` | `Fn[(items:List[Any],predicate:Any)->List[Any]]` |
| `list.filtering` | `Fn[(predicate:Any)->Any]` |
| `list.map` | `Fn[(items:List[Any],transform:Any)->List[Any]]` |
| `list.mapping` | `Fn[(transform:Any)->Any]` |
| `list.reduce` | `Fn[(items:List[Any],initial:Any,reduce:Any)->Any]` |
| `list.transduce` | `Fn[(items:List[Any],through:Any,initial:Any,reduce:Any)->Any]` |
| `log.info` | `Fn[(message:str)->nil!log]` |
| `parse.i32` | `Fn[(text:str)->R[i32,str]]` |
| `print` | `Fn[(value:Any)->nil!io]` |
| `println` | `Fn[(value:Any)->nil!io]` |
| `test.eq` | `Fn[(actual:Any,expected:Any)->nil]` |
<!-- builtin-contracts:end -->

Operational details: `print` omits a newline; `println` adds one. `env.get_or` reads an
existing variable or returns fallback. `cli.args` returns arguments passed to the generated
program. File helpers read/write UTF-8-style string contents, returning OS errors as strings;
write replaces the target with mode 0644 for a new file. Test assertions compare values and
fail execution on a mismatch. `parse.i32` parses signed base-10 text into the i32 range.

`http.Req` exposes method/path/body; `http.Resp` exposes code/body/headers. Create them with
HTTP helpers. `http.json` takes an already-encoded JSON string, not an arbitrary record.
`http.with_header` updates a response; do not assume general record immutability for this
host helper. `http.query` currently performs simple raw query-pair lookup, not full URL
percent-decoding. `http.serve` listens until stopped and invokes a handler with key `req`
returning `R[http.Resp,str]`; an unhandled Err/runtime failure becomes HTTP 500. Application
code must map domain failures to deliberate status codes. No outbound HTTP-client builtin,
real time/sleep API, general map API or process execution API is provided at this revision.

## Execution, tools and unsupported forms

`kiro run`/`build`/`test` produce native Go-hosted interpreter programs; this is not direct
typed Go lowering. `kiro inspect go` emits structural inspection output, not a faithful
executable translation. Main is a parameterless function in the entry module. An integer
main return sets the exit code, nil means success, and an outer result is unwrapped once.
An unhandled Err fails the process. Tests are parameterless `test_*` functions; assertions,
runtime errors and returned Err fail tests. Ordinary tests retain shared-runtime/fail-fast
behavior; optional coverage uses fresh interpreters and collects failures.

| Command | When to use it |
| --- | --- |
| `kiro describe --reference` | Read this exact bundled document (`--json` adds version/hash metadata) |
| `kiro describe --json` | Discover guide revision, language capabilities, builtins and diagnostics |
| `kiro symbols . --json` | Read declared signatures, schemas, defaults and source revision |
| `kiro fmt .` | Apply canonical formatting |
| `kiro check . --json` | Check syntax/imports/known contracts; repair the first reported error and rerun |
| `kiro lint . --json` | Semantic check followed by canonical-format check |
| `kiro fmt . --check --json` | Check formatting without writing |
| `kiro test .` | Execute the project's tests |
| `kiro analyze <operation> . --json` | Explicitly request maintenance evidence when useful |

Compiler acceptance is not a complete type/effect proof: unknown expression types can be
validated only at execution. Source coordinates are one-based Unicode code points; reports
mark exact tokens versus enclosing declarations. LSP uses zero-based UTF-16 ranges.
Analysis, coverage, mutation and architecture policy never run implicitly during checking,
compilation, lint or LSP updates. Users may explicitly add hooks. There is no human code-review
gate in the experiment.

Do not generate if/else, while, break/continue, defer, lambdas, nested functions, expression
function bodies, pipe operators, maps, list indexing, logical operators, arbitrary generic
functions, aliases for list.fold, lazy sequences, stateful transducers, tagged optionals or
a distinct unit type. Reserved words in a lexer or historical/proposal documents do not
establish executable support. Use the forms and runtime contracts in this reference.

## Conformance examples

Every `ki` fence in this reference is a complete single-file project. Metadata identifies
its stable example ID and expected outcome. `expect=pass` requires checking plus passing
Kiro tests; `expect=reject` requires a compiler rejection with the named diagnostic code;
`expect=runtime-error` requires checking but a failing Kiro test with the named message
fragment. These tests run in `go test ./docs/llm`. They establish the recorded examples,
not agreement for every possible program. Rejection blocks below are intentionally invalid.

A key is mandatory even when a declaration has a default:

```ki example=missing-key expect=reject diagnostic=KIRO_ARGUMENT_MISSING
mod main
fn add(value:i32, amount:i32 = 1) -> i32 { return value + amount }
fn main() -> i32 { return add(value:4) }
```

Positional calls and expression function bodies are removed:

```ki example=positional expect=reject diagnostic=KIRO_SYNTAX
mod main
import parse
fn main() -> i32 { return parse.i32("1") }
```

```ki example=expression-body expect=reject diagnostic=KIRO_SYNTAX
mod main
fn main() -> i32 = 0
```

A known type mismatch and an unknown identifier fail checking:

```ki example=wrong-type expect=reject diagnostic=KIRO_TYPE_MISMATCH
mod main
fn main() -> i32 { return "0" }
```

```ki example=unknown-name expect=reject diagnostic=KIRO_UNKNOWN_SYMBOL
mod main
fn main() -> i32 { return missing }
```

The caller must declare operational effects:

```ki example=missing-effect expect=reject diagnostic=KIRO_EFFECT_REQUIRED
mod main
fn main() -> i32 {
  println(value:"hello")
  return 0
}
```

Neither potentially empty iteration nor successful propagation guarantees a return:

```ki example=missing-return expect=reject diagnostic=KIRO_MISSING_RETURN
mod main
fn first(items:List[i32]) -> i32 {
  for item in items { return item }
}
fn main() -> i32 { return 0 }
```

Every declared tag needs a total arm:

```ki example=missing-variant expect=reject diagnostic=KIRO_NON_EXHAUSTIVE
mod main
type Failure { variant Missing {} variant Offline {} }
fn explain(error:Failure) -> str {
  return when (error) { Failure.Missing() => "missing" }
}
fn main() -> i32 { return 0 }
```

Optional source fields must still be supplied:

```ki example=missing-field expect=reject diagnostic=KIRO_CHECK_FAILED
mod main
type User { name:str email:?str }
fn main() -> i32 {
  let user = User{name:"Ada"}
  return 0
}
```

Local let bindings cannot be reassigned:

```ki example=immutable expect=reject diagnostic=KIRO_CHECK_FAILED
mod main
fn main() -> i32 {
  let number = 1
  number = 2
  return number
}
```

Unsupported control-flow syntax must not be inferred from other languages:

```ki example=unsupported-if expect=reject diagnostic=KIRO_SYNTAX
mod main
fn main() -> i32 {
  if true { return 1 }
  return 0
}
```

Arithmetic failure aborts execution rather than becoming a domain result:

```ki example=division-by-zero expect=runtime-error message=division
mod main
fn divide(value:i32) -> i32 { return 1 / value }
fn main() -> i32 { return 0 }
fn test_failure() -> nil {
  let value = divide(value:0)
  return nil
}
```
