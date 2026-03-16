# AGENTS Guidelines

## Scope
These instructions apply to the full repository.

## Implementation expectations
- Follow the Phase 1 plan in `docs/phase1-spec.md` and keep milestones incremental.
- Prefer straightforward, readable Go over abstractions.
- Keep parser/formatter behavior deterministic.

## Documentation policy
- Any user-visible or architectural change should update relevant docs (`README.md`, `docs/*`, examples).
- Keep examples runnable and aligned with implemented syntax.

## Testing policy
- Add or update tests when changing lexer, parser, formatter, or CLI behavior.
- Run `go test ./...` before finishing.
