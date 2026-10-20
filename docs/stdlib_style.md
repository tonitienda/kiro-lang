# Standard library style

Kiro's stdlib surface is intentionally narrow and regular.

## Naming policy

- use short module names with clear semantic ownership: `env`, `fs`, `http`, `json`, `log`, `parse`, `test`, `time`, `list`
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

## Collection style

Use eager `list.map(items:xs, transform:f)`, `list.filter(items:xs, predicate:p)`, and
`list.reduce(items:xs, initial:zero, reduce:r)`. Callbacks are pure with explicit `value`
roles; reducers additionally use `state` and return the state type. See
[the functional contract](functional-core.md).

For explicit fusion, use typed `Transform[Input,Output]` values built by `list.mapping`,
`list.filtering` and `list.compose(first:a, then:b)`. `list.transduce` produces an accumulator
without intermediate data lists; `list.collect` builds only the final list. These terminals
execute eagerly per item. `list.fold` was renamed to `list.reduce`.

## JSON and result boundaries

`json.encode` returns `R[str,str]`; `json.decode[T]` requires an explicit schema and returns
`R[T,str]`. Unknown/duplicate/missing required fields and wrong types are rejected. Encoding
records excludes internal metadata. `parse.i32` returns Err for out-of-range text. Use `?`
only in a result-returning function with a compatible error channel. See
[results-and-schemas.md](results-and-schemas.md).
