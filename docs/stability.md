# Stability policy (Phase 8)

Kiro remains experimental, but Phase 8 introduces explicit compatibility commitments.

## Intended to remain source-compatible (high stability)

- core file/module shape (`mod`, `import`, project directory entry with `main.ki`)
- deterministic formatter behavior
- declaration-level syntax already in broad use in examples/templates
- `kiro fmt`, `kiro check`, `kiro inspect go`, `kiro new` command entrypoints
- compatibility corpus fixture format and execution model

## Stable-ish but still evolving (medium stability)

- diagnostics text details (message wording may evolve, key signal substrings should stay meaningful)
- generated-Go layout (`src/` + `runtime/`) for inspection workflows
- service template directory structure
- stdlib helper naming for service-centric modules

## Experimental / allowed to change (lower stability)

- semantics and type-checking completeness for advanced cases
- runtime behavior of commands still marked as placeholder (`build`, `run`, `test` in this frontend slice)
- generated Go as an external API contract (not promised stable)
- advanced concurrency and generic-heavy patterns not yet locked down by broad acceptance tests

## Compatibility matrix (current focus)

Supported well:

- CLI-style tools and small file-processing utilities
- small HTTP service layout patterns
- JSON API handler shapes
- handler-oriented test module structure
- moderate orchestration patterns using `group` + `spawn` + `await`

Still weak / experimental:

- large multi-package applications
- complex generic typing edge-cases
- advanced JSON decoding behavior guarantees
- portability beyond Linux/macOS
