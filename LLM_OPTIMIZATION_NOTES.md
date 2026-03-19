# LLM Optimization Notes

This repository now treats language simplification as a first-class goal.

## Summary of the cleanup

### Removed or tightened

- expression-bodied functions
- mixed handler return styles in templates/examples
- the `fs.read` alias
- docs that blurred effects with fallibility

### Canonical forms now preferred

- `fn ... { ... }` for every function
- explicit `return`
- `fn handler(req:http.Req) -> R[http.Resp, str]`
- `group { let t = spawn ...; let v = await t }`
- `fs.read_file(...)`
- `env.get_or(...)` for defaulted environment access

## Why these changes were made

The repository is optimizing for:

- smaller surface area
- stronger canonicality
- explicit semantics
- easier repair from diagnostics
- better alignment across compiler, formatter, docs, examples, templates, and editor tooling

## Migration notes

### Old

```ki
fn add(a:i32, b:i32) -> i32 = a + b
```

### New

```ki
fn add(a:i32, b:i32) -> i32 {
  return a + b
}
```

### Old

```ki
fs.read(path)
```

### New

```ki
fs.read_file(path)
```

### Old mixed handler style

```ki
fn handler(req:httpReq) -> Resp
```

### New canonical handler style

```ki
fn handler(req:http.Req) -> R[http.Resp, str]
```

## Still intentionally experimental

- breadth of runtime execution coverage across all examples
- exact standalone workdir details used by build/run/test
- generated-Go API shape
