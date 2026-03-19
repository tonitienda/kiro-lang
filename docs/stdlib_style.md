# Standard library style

Kiro's stdlib surface is intentionally narrow and regular.

## Naming policy

- use short module names with clear semantic ownership: `env`, `fs`, `http`, `json`, `log`, `parse`, `test`, `time`
- use verb-oriented helpers for actions: `read_file`, `write_file`, `get_or`, `require`
- keep constructors and pure transforms explicit

## Effect policy

Operational APIs carry effects; pure transforms do not.

Examples:

- `env.get_or` -> `!env`
- `fs.read_file` -> `!fs`
- `http.serve` -> `!net`
- `log.info` -> `!log`
- `json.encode` -> pure `R[T,E]`
- `json.decode` -> pure `R[T,E]`
- `parse.i32` -> pure `R[T,E]`

## Result/optional policy

- do not use effects to model fallibility
- do not use `nil` as a substitute for `Err(...)`
- keep optional values typed as `?T`

## HTTP style

Prefer:

- `http.text`
- `http.json`
- `http.not_found`
- `http.with_header`

Keep handlers on `R[http.Resp,str]` so error flow stays explicit.
