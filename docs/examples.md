# Examples

The examples directory is meant to teach the canonical Kiro style.

## Recommended starting points

- `examples/hello/` — tiny CLI entrypoint
- `examples/http_hello/` — canonical HTTP handler shape
- `examples/http_json/` — pure JSON encode/decode with `R[T,E]`
- `examples/service_parallel/` — structured concurrency with `group`
- `examples/service_config/` — explicit environment-based config loading
- `examples/test_demo/` — canonical direct tests with `test.eq`

## What examples should demonstrate

- block-only function bodies
- explicit `return`
- explicit effect boundaries
- separation of `R[T,E]`, `?`, `?T`, and `nil`
- structured concurrency with local task lifecycles
