# PHASE7 Notes

## What changed

- Added clearer package/module and service-structure documentation.
- Added dedicated docs for config, concurrency, generated-Go debugging, and stability expectations.
- Updated service-oriented examples (`service_config`, `service_health`, `service_parallel`) and added missing Phase 7 examples (`test_http_handlers`, `request_json`, `group_ctx`).
- Improved `kiro new service` template to scaffold app/config/test directories.
- Improved generated-Go inspect layout by separating output into `.kiro-gen/src` and `.kiro-gen/runtime` with source-origin declaration comments.
- Expanded CLI tests for new scaffold and inspect layout behavior.

## Package/project decisions

- Keep project loading simple: one root, all `.ki` files under root included.
- Treat `main` as composition root; move service logic into `app` and `internal/*` modules.
- Keep import rules deterministic and path-friendly.

## Service-style decisions

- Explicit startup flow: load config -> log startup -> serve handler.
- Handler-level tests should call handlers directly using tiny request helpers.
- Preserve tiny stdlib style; avoid framework-level abstractions.

## JSON/decode decisions

- Document explicit decode-in-handler style (`json.decode[T]` where available).
- Keep request parsing local and obvious; no reflection-heavy automatic binding.

## Known limitations

- `kiro build`, `kiro run`, and `kiro test` are still placeholders in this frontend-focused slice.
- Type checking/runtime behavior remains partial relative to language aspirations.
- Several example APIs represent intended surface and may depend on upcoming runtime completion.

## Recommended Phase 8 scope

- Land concrete runtime support for `kiro run/test` with handler-level test execution.
- Improve diagnostics around group/spawn/await misuse.
- Align stdlib helper names and signatures with implemented runtime behavior.
- Expand generated-Go source mapping with precise line-level origin comments.
