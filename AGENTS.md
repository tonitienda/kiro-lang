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
- Phase 4 has started; check `PHASE4_NOTES.md` first for latest implemented scope and limitations.
- Keep `PHASE3_NOTES.md` aligned when touching earlier-phase assumptions.
