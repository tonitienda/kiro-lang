# RFC: Add Effect Declarations to Kiro

- Status: Proposed
- Target phase: Next implementation phase after current `main`
- Author: Toni / Codex
- Scope: Parser, AST, formatter, sema, docs, compatibility fixtures
- Non-goal for this RFC: full effect inference, effect polymorphism, async/effect interaction redesign

## Summary

Add **effect declarations** to Kiro function signatures using a compact postfix syntax such as:

```ki
fn read_file(path:str) -> R[str, str] !fs
fn serve(addr:str) -> R[nil, str] !net
fn app(req:http.Req) -> R[http.Resp, str] !json
fn pure_add(a:i32, b:i32) -> i32
```

Effects are explicit annotations on function declarations that describe externally observable behavior such as file I/O, networking, environment access, logging, process interaction, time, JSON/serialization, or panic-like unsafe runtime behavior.

The first version should focus on:
1. syntax
2. AST representation
3. formatting
4. declaration checking
5. call-site checking with explicit effect propagation
6. docs and compatibility coverage

The first version should **not** try to solve full effect inference or higher-kinded effect systems.

## Motivation

Kiro is intentionally explicit and backend-oriented. Effects fit that style very well.

Benefits:
- function signatures become clearer
- service code is easier to read and audit
- compiler feedback improves when impure work leaks into places intended to stay pure
- generated code structure stays unchanged, but source-level semantics become stronger
- this aligns with Kiro’s philosophy of small explicit semantics (`R[T,E]`, `?`, `?T`, `group`, etc.)

Example:

```ki
fn load_config() -> R[AppConfig, str] !env
fn build_handler(cfg:AppConfig) -> Handler
fn main() -> i32 !env !log !net
```

This communicates intent immediately.

## Design goals

1. Keep syntax small and readable.
2. Require no effect annotation for pure functions.
3. Make effects explicit at function declarations.
4. Make effect checking simple and conservative.
5. Avoid broad inference or advanced effect algebra in v1.
6. Preserve Kiro’s formatting and compatibility discipline.

## Non-goals

This RFC does **not** include:
- effect inference across arbitrary expressions
- effect polymorphism (`fn map[F](...) !F`)
- row-polymorphic effects
- effect handlers
- async/effect redesign
- effect annotations on variables, types, or blocks
- optimizer use of effects
- changes to generated Go runtime behavior

## Proposed syntax

### Function declarations

Effects appear after the return type:

```ki
fn read_file(path:str) -> R[str, str] !fs
fn serve(addr:str) -> R[nil, str] !net
fn log_startup(msg:str) -> nil !log
fn now_unix() -> i64 !time
fn add(a:i32, b:i32) -> i32
```

### Multiple effects

Multiple effects are written as repeated `!name` markers:

```ki
fn bootstrap() -> R[nil, str] !env !log !net
```

### Block-bodied functions

```ki
fn main() -> i32 !env !log !net {
  let port = env.get_or("PORT", ":8080")
  log.info("starting ${port}")
  http.serve(port, app)?
  return 0
}
```

### Methods

```ki
fn (s:Server) start() -> R[nil, str] !log !net
```

### Pure functions

A function with no declared effects is treated as pure:

```ki
fn text(code:i32, body:str) -> http.Resp =
  http.Resp{code:code body:body}
```

## Initial built-in effect names

V1 should support a fixed, documented set of effect names.

Recommended built-ins:
- `env`
- `fs`
- `io`
- `json`
- `log`
- `net`
- `panic`
- `proc`
- `time`

Notes:
- `fs` can be used instead of folding everything into `io`
- `http` does not need to be a separate effect if it is already covered by `net` and `json`
- keep the set small
- parser may accept any identifier after `!`, but sema should reject unknown effect names in v1

Recommended initial rule:
- accept only the built-in effect names above in v1
- reject unknown effect names with a clear diagnostic

## Semantics (v1)

### Core rule

If function `A` calls function `B`, then `A` must declare all effects required by `B`.

Example:

```ki
fn read_port() -> str !env {
  return env.get_or("PORT", ":8080")
}

fn main() -> i32 {
  let port = read_port()
  return 0
}
```

This should be an error because `main` is missing `!env`.

Correct version:

```ki
fn main() -> i32 !env {
  let port = read_port()
  return 0
}
```

### Transitive behavior

Effects are propagated through calls transitively by declaration checking, not by inference magic.

Example:

```ki
fn load_config() -> AppConfig !env
fn start_server(cfg:AppConfig) -> R[nil, str] !log !net

fn main() -> i32 !env !log !net {
  let cfg = load_config()
  start_server(cfg)?
  return 0
}
```

### Pure functions

A pure function cannot call an effectful function.

### Calls to stdlib

Stdlib functions/modules must have declared effects, for example:
- `fs.read_file` => `!fs`
- `env.get_or` => `!env`
- `log.info` => `!log`
- `http.serve` => `!net`
- `json.encode` => optionally pure or `!json`

Recommended decision:
- treat `json.encode` / `json.decode` as `!json` for clarity, even though they may not do external I/O
- this keeps serialization visible

### `panic`

If Kiro has or introduces panic-like runtime behavior intentionally exposed in source, mark it as `!panic`.

## Conservative v1 checking rules

The first implementation should be deliberately simple.

### Check effect requirements at:
- direct function calls
- method calls
- known stdlib calls
- calls inside block bodies
- calls inside `if`, `when`, loops, `group`, `spawn`, `await`

### Do not try to infer effects from:
- data values
- constructors
- constant expressions
- string interpolation alone
- local bindings unless they are function calls
- generated Go internals

### `spawn` and `await`

For v1:
- `spawn f()` requires the enclosing function to include the effects of `f`
- `await` itself does not add a new effect
- no separate `async` effect is introduced

Example:

```ki
fn fetch_a() -> str !net
fn fetch_b() -> str !net

fn app(req:http.Req) -> R[http.Resp, str] !json !net {
  group {
    let a = spawn fetch_a()
    let b = spawn fetch_b()
    let ra = await a
    let rb = await b
    let body = json.encode(Msg{a:ra b:rb})?
    return Ok(http.json(200, body))
  }
}
```

### `defer`

`defer` should not itself add an effect. The deferred call’s callee effects still count as effects of the enclosing function.

## AST changes

Add effect declarations to function and method declarations.

Suggested shape:

```go
type EffectName string

type FuncDecl struct {
    Name       string
    Receiver   *ReceiverDecl
    Params     []Param
    ReturnType TypeRef
    Effects    []EffectName
    Body       Node
    Doc        *DocComment
    Span       Span
}
```

Constraints:
- preserve declared order in the AST
- sema/formatter may normalize for comparison or printing
- duplicates should be rejected

## Parser changes

Extend function/method signature parsing.

Current:
- `fn name(args) -> Ret`
- `fn (r:T) name(args) -> Ret`

New:
- `fn name(args) -> Ret !env !net`
- `fn (r:T) name(args) -> Ret !log`

Parsing rules:
1. parse return type first
2. then parse zero or more `!identifier`
3. then parse body (`=` expr or `{ ... }`)

Example valid forms:

```ki
fn add(a:i32, b:i32) -> i32 =
  a + b

fn run() -> i32 !env {
  return 0
}
```

Invalid:

```ki
fn run() !env -> i32
```

Keep the position fixed after the return type.

## Formatter changes

`kiro fmt` should print effects in canonical order.

Recommendation:
- sort effects lexicographically for canonical formatting

Example:

Input:

```ki
fn main() -> i32 !net !env !log {
  return 0
}
```

Formatted:

```ki
fn main() -> i32 !env !log !net {
  return 0
}
```

Also:
- one space between return type and first effect
- one space between effects
- no punctuation between effects

## Semantic analysis

### Validations

1. effect names must be known
2. no duplicate effects in one signature
3. pure functions cannot call effectful functions
4. effectful functions must declare all effects used by callees
5. diagnostics should point to:
   - missing effect on enclosing function
   - unknown effect
   - duplicate effect

### Diagnostics examples

#### Missing effect

```text
main.ki:10:1: function `main` calls `env.get_or` which requires effect `!env`
hint: add `!env` to the function signature
```

#### Duplicate effect

```text
main.ki:3:26: duplicate effect `!env` in function signature
```

#### Unknown effect

```text
main.ki:3:26: unknown effect `!database`
known effects: !env, !fs, !io, !json, !log, !net, !panic, !proc, !time
```

### Call-check approach

V1 may use a simple call-check strategy:
- after symbol resolution and function signature collection
- while checking each function body
- each call returns callee effect set
- enclosing function declaration must be a superset

No need for a separate global effect inference engine.

## Stdlib annotation plan

Annotate current stdlib declarations/documentation with effects.

Suggested initial mapping:

### `env`
- `env.get` -> `!env`
- `env.get_or` -> `!env`
- `env.require` -> `!env`

### `fs`
- `fs.read_file` -> `!fs`
- `fs.write_file` -> `!fs`

### `log`
- `log.info` -> `!log`
- `log.warn` -> `!log`
- `log.error` -> `!log`

### `http`
- `http.serve` -> `!net`
- helpers like `http.text`, `http.json`, `http.not_found` -> pure

### `json`
Two possible choices:

#### Option A (recommended for clarity)
- `json.encode` -> `!json`
- `json.decode` -> `!json`

#### Option B
- treat them as pure

Recommendation: choose **Option A** for now because the language goal is explicitness more than theoretical purity.

### `time`
- `time.now_unix` -> `!time`
- `time.sleep_ms` -> `!time`

### `proc`
- process execution or exit helpers -> `!proc`

## Examples

### Pure helper + effectful main

```ki
fn greet(name:str) -> str =
  "hello ${name}"

fn main() -> i32 !log {
  log.info(greet("toni"))
  return 0
}
```

### Service startup

```ki
type AppConfig {
  port:str
}

type Msg {
  status:str
}

fn load_config() -> R[AppConfig, str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(AppConfig{port:port})
}

fn app(req:http.Req) -> R[http.Resp, str] !json {
  when req.path
    "/health" => {
      let body = json.encode(Msg{status:"ok"})?
      return Ok(http.json(200, body))
    }
    _ => {
      return Ok(http.not_found())
    }
}

fn main() -> i32 !env !log !net {
  let cfg = load_config()?
  log.info("starting ${cfg.port}")
  http.serve(cfg.port, app)?
  return 0
}
```

### File tool

```ki
fn load(path:str) -> R[str, str] !fs {
  return fs.read_file(path)
}

fn main() -> i32 !fs !log {
  let s = load("README.md")?
  log.info(s)
  return 0
}
```

## Backward compatibility

This is a source-language extension.

Recommended compatibility stance:
- existing effect-free programs continue to compile if they do not call effectful functions
- programs that call stdlib effectful functions may now need effect annotations added
- this is a deliberate tightening and should be called out in docs/migration notes

Suggested rollout:
1. parser/formatter support
2. stdlib declarations annotated
3. sema checks enabled
4. compatibility corpus/examples updated
5. docs and migration notes updated

Optional softer rollout:
- first ship warnings for missing effects
- then later upgrade to errors

Recommendation:
- go straight to **errors** in compiler checks for consistency, unless the current repo state is too broad and warnings are needed to avoid huge churn

## Tooling implications

### Language server
- hover should show effects in function signatures
- definition info should include declared effects

### Formatter
- canonicalize effect order

### Compatibility corpus
Add fixtures for:
- valid pure functions
- valid effectful declarations
- missing effect diagnostics
- duplicate effect diagnostics
- stdlib usage requiring effects
- `spawn`/`group` with effectful callees

## Documentation changes

Update:
- `docs/language_tour.md`
- `docs/stable_core.md`
- `docs/design_principles.md`
- `docs/compatibility.md`
- `README.md`

Add:
- `docs/effects.md`

`docs/effects.md` should explain:
- syntax
- built-in effect names
- why Kiro uses explicit effects
- current limitations of the system

## Implementation plan

### Milestone 1
- parser support for `!effect` list after return type
- AST updates
- formatter support
- parser/formatter tests

### Milestone 2
- stdlib declaration annotations
- sema support for function effect sets
- missing/duplicate/unknown effect diagnostics
- tests

### Milestone 3
- call-site propagation checks
- checks for method calls, `spawn`, `defer`, control-flow bodies
- tests

### Milestone 4
- docs
- compatibility fixtures
- language server hover/signature updates if present
- migration notes

## Acceptance criteria

1. Kiro parses and formats function effects correctly.
2. Effect-free pure code still works.
3. Calls to effectful functions require matching declared effects.
4. Missing/unknown/duplicate effects produce clear diagnostics.
5. Stdlib modules have documented effect annotations.
6. Docs and compatibility fixtures cover the feature.
7. If `kiro-lsp` exists, hover/signature info includes effects.

## Open questions for implementation

1. Should `json.encode/decode` be `!json` or pure?
   - Recommendation: `!json` in v1 for clarity.

2. Should effect order be preserved or canonicalized?
   - Recommendation: canonicalize lexicographically.

3. Should unknown effect names be allowed as user-defined tags?
   - Recommendation: no, not in v1. Keep a fixed built-in set.

4. Should missing effects be warnings first?
   - Recommendation: only if migration churn is too high. Otherwise make them errors immediately.

## Request to implement

Please implement this RFC incrementally, reusing existing parser/sema/formatter infrastructure, and update:
- docs
- compatibility fixtures
- examples
- LSP hover/signature support if present

Keep the design intentionally small and explicit.
