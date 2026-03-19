# AGENTS Guidelines

## Scope
These instructions apply to the full repository.

## Implementation expectations
- Follow `docs/phase1-spec.md` for foundation and `docs/roadmap-phase2.md` for incremental history.
- For current language direction and limits, read `PHASE6_NOTES.md` first.
- For the current LLM-focused cleanup decisions, read `LLM_REDESIGN_NOTES.md` before changing syntax, diagnostics, templates, or public docs.
- Prefer straightforward, readable Go over abstractions.
- Keep parser/formatter behavior deterministic.

## Documentation policy
- Any user-visible or architectural change should update relevant docs (`README.md`, `docs/*`, examples).
- Keep examples runnable and aligned with implemented syntax.
- If command behavior changes, update command docs in `README.md` and related docs pages.
- Keep agent-facing guidance and milestone notes (`AGENTS.md`, `PHASE*_NOTES.md`) aligned with workflow changes.
- Keep `docs/llm/SKILL.md`, `docs/llm/references/kiro.json`, and `docs/llm/references/examples/` aligned with the stable core whenever syntax, stdlib guidance, or canonical project conventions change.
- Keep installer and release docs aligned with artifact naming, checksum generation, the bundled-toolchain layout, and the packaged VS Code `.vsix` flow.
- Keep `kiro new` templates plus the project-local `.kiro/` skill snapshot aligned with `docs/llm/` and release versioning.

## Testing policy
- Add or update tests when changing lexer, parser, formatter, project resolution, or CLI behavior.
- Run `go test ./...` before finishing.
- When compatibility fixtures change, run `go run ./cmd/kiro compat`.
- When changing the release installer or bundle layout, run `scripts/write_release_checksums.sh` plus `scripts/verify_install.sh` against a local artifact.

## Current milestone context
- Phase 11 is active: prioritize editor tooling stability (LSP, syntax highlighting, packaged VS Code `.vsix` delivery, setup docs) while preserving the Phase 10 stable-core contract.
- The active redesign direction favors a smaller stricter language for LLM generation over compatibility with older prototype forms.
- Keep `PHASE11_NOTES.md` and `PHASE10_NOTES.md` aligned with implementation decisions and tradeoffs.
- Keep `PHASE8_NOTES.md` plus `PHASE6_NOTES.md`, `PHASE5_NOTES.md`, `PHASE4_NOTES.md`, and `PHASE3_NOTES.md` aligned when touching earlier-phase assumptions.
