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

## Scaffolded projects and local skill snapshots

`kiro new hello` and `kiro new service` now also write a small `.kiro/` directory into the generated project:

- `.kiro/skill/KIRO_SKILL.md` and `.kiro/skill/kiro.json` are canonical copies from `docs/llm/`
- `.kiro/version.json` pins the Kiro version used to scaffold the repo
- `.kiro/README.md` is the minimal human/agent reminder for editing `.ki` files

This keeps fresh example repos and downstream playgrounds self-contained for LLM/editor workflows without requiring the whole `kiro-lang` repo checkout.
