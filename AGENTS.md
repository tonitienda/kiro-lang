# AGENTS Guidelines

## Scope
These instructions apply to the full repository.

## Implementation expectations
- Follow `docs/phase1-spec.md` for foundation and `docs/roadmap-phase2.md` for incremental history.
- For current language direction and limits, read `PHASE6_NOTES.md` first.
- Prefer straightforward, readable Go over abstractions.
- Keep parser/formatter behavior deterministic.

## Documentation policy
- Any user-visible or architectural change should update relevant docs (`README.md`, `docs/*`, examples).
- Keep examples runnable and aligned with implemented syntax.
- If command behavior changes, update command docs in `README.md` and related docs pages.

## Testing policy
- Add or update tests when changing lexer, parser, formatter, project resolution, or CLI behavior.
- Run `go test ./...` before finishing.

## Current milestone context
- Phase 6 has started; keep `PHASE6_NOTES.md` aligned with implementation.
- Keep `PHASE5_NOTES.md`, `PHASE4_NOTES.md`, and `PHASE3_NOTES.md` aligned when touching earlier-phase assumptions.
