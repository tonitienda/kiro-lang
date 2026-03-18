# Phase 10 notes

## Stable-core decisions

- Added `docs/stable_core.md` to explicitly define stable experimental core vs flexible areas.
- Core includes language constructs already used by templates/examples and canonical CLI workflow (`fmt`, `check`, `inspect go`, `new`, `compat`).
- Explicit function effect declarations are now part of the stable experimental core for operational behavior.

## Deprecation/cleanup decisions

- No hard removals in this milestone.
- CLI surface kept conservative; placeholder commands (`build/run/test`) remain visible but explicitly marked as not implemented in this repository slice.

## Release-shape decisions

- README now presents Kiro as experimental but deliberate, with quick-start and command/docs index.
- Added release process/checklist in `docs/releasing.md`.
- Added `CHANGELOG.md` with lightweight milestone history model.

## CLI/workflow decisions

- Added top-level help command and unified usage text for discoverability (`kiro help`, `--help`, `-h`).
- Unknown command errors now include usage guidance.
- `kiro check` now covers effect declaration validation in addition to parse/import checks.

## Compatibility/CI decisions

- Updated compatibility docs with explicit fixture categories (stable-core, compatibility, regression, diagnostics, templates, codegen).
- Stable-core fixtures are the strongest source-compatibility guardrail.
- Added regression fixtures for missing, duplicate, and unknown effect diagnostics.

## Known limitations

- `kiro build`, `kiro run`, and `kiro test` remain placeholders in this frontend-focused slice.
- Generated Go remains a debugging-oriented surface, not a stability guarantee.
- Effects are declaration-based and conservative; there is no effect polymorphism or general inference.

## Recommended post-Phase 10 direction

1. Implement runtime-backed behavior for build/run/test in a disciplined, compatibility-aware way.
2. Expand stable-core acceptance fixtures and selected codegen snapshots.
3. Continue diagnostics quality work with targeted regression fixtures.
