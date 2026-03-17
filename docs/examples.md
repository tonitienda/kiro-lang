# Flagship examples and templates

This page lists the primary examples/templates that represent the current Kiro way.

## Templates

- `kiro new hello`
  - minimal entry project
- `kiro new service`
  - tiny service layout with config + handler + handler test

## Flagship examples

- Tiny CLI example: `examples/cli_counter/`
- Tiny HTTP JSON service: `examples/http_json/`
- Handler testing example: `examples/test_http_handlers/`
- Concurrency/group example: `examples/group_parallel/`

## Validation expectations

For flagships and templates, maintainers should verify:

- `kiro fmt` succeeds and is idempotent
- `kiro check` succeeds
- template outputs stay aligned with README/docs
- compatibility fixtures continue to reflect canonical style
