# AGENTS Guidelines

## Scope
These instructions apply to the full repository.

## Implementation expectations
- Read `docs/llm/references/language.md` as the authoritative current language reference. `docs/phase1-spec.md` and phase notes are historical/secondary, not extra syntax.
- Read `PHASE20_NOTES.md`, `benchmarks/maintenance/README.md`, `PHASE19_NOTES.md` and `docs/research/benchmark_onboarding.md` for current agent onboarding/conformance; keep reference examples executable and label expected rejections.
- For current language direction and limits, read `PHASE6_NOTES.md` first.
- For the current LLM-focused cleanup decisions, read `LLM_REDESIGN_NOTES.md` before changing syntax, diagnostics, templates, or public docs.
- For the active agent-only reassessment, read `PHASE18_NOTES.md`, `docs/analysis-tools.md`, `PHASE17_NOTES.md`, `docs/agent-tools.md`, `PHASE16_NOTES.md`, `docs/variants-and-matching.md`, `PHASE15_NOTES.md`, `docs/results-and-schemas.md`, `PHASE14_NOTES.md`, `PHASE13_NOTES.md`, `docs/functional-core.md`, `PHASE12_NOTES.md` and `docs/research/agent_core_reassessment.md`; distinguish implemented syntax from proposals.
- Prefer straightforward, readable Go over abstractions.
- Keep parser/formatter behavior deterministic.
- Expensive analysis is opt-in via `kiro analyze`; never add graph construction, coverage, mutation, clustering or policy hooks to normal check/lint/LSP/build paths. Invite agents to select tools when useful; users control hooks.
- Every value argument is named; preserve source evaluation order when formatting or migrating calls. Built-in names come from `runtimekit/arguments.go` and the validated compact manifest.

## Documentation policy
- Any user-visible or architectural change should update relevant docs (`README.md`, `docs/*`, examples).
- Keep examples runnable and aligned with implemented syntax.
- If command behavior changes, update command docs in `README.md` and related docs pages.
- Keep agent-facing guidance and milestone notes (`AGENTS.md`, `PHASE*_NOTES.md`) aligned with workflow changes.
- Keep `docs/llm/SKILL.md`, `docs/llm/references/kiro.json`, and `docs/llm/references/examples/` aligned with the stable core whenever syntax, stdlib guidance, or canonical project conventions change.
- Keep installer and release docs aligned with artifact naming, checksum generation, the bundled-toolchain layout, and the packaged VS Code `.vsix` flow.
- Keep `kiro new` templates plus the project-local `.kiro/` skill snapshot aligned with `docs/llm/` and release versioning.

## Testing policy

- The versioned Go counterpart and scaffolded native-pilot workflow are documented in `benchmarks/maintenance/GO_PILOT.md`. Keep the original Kiro v1 fixture/cohort unchanged; new fixture/task mappings require explicit versions. Go fixture tests/vet run separately because it is a nested module.
- For benchmark changes run `python3 -B -m unittest discover -s benchmarks/maintenance/runner -v` and the documented `selftest` with a built Kiro CLI. Distinguish scripted controls from model trials; never run model candidates on the host as an isolation fallback.
- Add or update tests when changing lexer, parser, formatter, project resolution, or CLI behavior.
- Run `go test ./...` before finishing.
- When compatibility fixtures change, run `go run ./cmd/kiro compat`.
- When changing the release installer or bundle layout, run `scripts/write_release_checksums.sh` plus `scripts/verify_install.sh` against a local artifact.

## Current milestone context
- Phase 20 is active: prioritize enforceable semantics and agent/tool interfaces for agent-only implementation/review/audit. Preserve the Phase 10 core concepts, with breaking changes explicitly recorded in the active milestone notes.
- The active redesign direction favors a smaller stricter language for LLM generation over compatibility with older prototype forms.
- Keep `PHASE11_NOTES.md` and `PHASE10_NOTES.md` aligned with implementation decisions and tradeoffs.
- Keep `PHASE8_NOTES.md` plus `PHASE6_NOTES.md`, `PHASE5_NOTES.md`, `PHASE4_NOTES.md`, and `PHASE3_NOTES.md` aligned when touching earlier-phase assumptions.
