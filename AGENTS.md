# AGENTS Guidelines

## Scope
These instructions apply to the full repository.

## Implementation expectations
- Follow `docs/phase1-spec.md` for foundation and `docs/roadmap-phase2.md` for current incremental Phase 2 work.
- Prefer straightforward, readable Go over abstractions.
- Keep parser/formatter behavior deterministic.

## Documentation policy
- Any user-visible or architectural change should update relevant docs (`README.md`, `docs/*`, examples).
- Keep examples runnable and aligned with implemented syntax.

## Testing policy
- Add or update tests when changing lexer, parser, formatter, or CLI behavior.
- Run `go test ./...` before finishing.

## Current milestone context
- Phase 3 has started; use `PHASE3_NOTES.md` for the latest implemented scope and limitations before adding new Phase 3 features.
