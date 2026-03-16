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
- Keep agent-facing guidance and milestone notes (`AGENTS.md`, `PHASE*_NOTES.md`) aligned with workflow changes.

## Testing policy
- Add or update tests when changing lexer, parser, formatter, project resolution, or CLI behavior.
- Run `go test ./...` before finishing.
- When compatibility fixtures change, run `go run ./cmd/kiro compat`.

## Current milestone context
- Phase 8 is active: prioritize compatibility corpus coverage, deterministic checks, and CI reliability over new syntax.
- Keep `PHASE8_NOTES.md` aligned with implementation and tradeoffs.
- Keep `PHASE6_NOTES.md`, `PHASE5_NOTES.md`, `PHASE4_NOTES.md`, and `PHASE3_NOTES.md` aligned when touching earlier-phase assumptions.
