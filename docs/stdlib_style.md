# Stdlib style

Kiro's stdlib surface is intentionally small and regular.

## Core rules

### Naming

- prefer short lower_snake names
- prefer one public name per operation
- remove aliases that do not pay for themselves

### Effects

Operational APIs carry effects; pure transforms do not.

Examples:

- `env.get_or` -> `!env`
- `fs.read_file` -> `!fs`
- `http.serve` -> `!net`
- `log.info` -> `!log`
- `json.encode` -> pure `R[T,E]`
- `parse.i32` -> pure `R[T,E]`

### Result and optional shape

- use `R[T,E]` for recoverable failure
- use `?T` for optional values
- use `nil` only for optional absence

## Canonical module notes

### `env`

- `get_or(key, default)` is the defaulted configuration path
- environment access is always `!env`

### `fs`

- prefer `read_file(path)`
- prefer `write_file(path, body)`
- `fs.read` has been removed from the canonical surface

### `http`

- handlers should use `fn handler(req:http.Req) -> R[http.Resp, str]`
- prefer `http.text`, `http.json`, `http.not_found`, and `http.with_header`
- prefer direct handler testing with `http.test_req`

### `json`

- `encode` and `decode` are pure
- JSON does not imply an effect declaration by itself

### `log`

- log calls are operational and require `!log`

### `test`

- use `test.eq` for simple direct assertions in canonical examples
