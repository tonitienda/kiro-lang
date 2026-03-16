# Phase 9 notes

## Milestone 1 progress (quality + consistency baseline)

This update starts the Phase 9 documentation and policy baseline with focus on consistency and workflow clarity.

## Stdlib/API consistency decisions

- Added `docs/stdlib_style.md` as the canonical naming/shape guide for stdlib modules (`env`, `parse`, `http`, `json`, `log`, `test`, `fs`, `ctx`, and concurrency helpers).
- Locked conventions for naming (`get/get_or/require/must` families), argument ordering, and result/error surface consistency.
- Added a concrete review checklist so stdlib edits remain coherent.

## Deprecation decision

- Adopted a **documented deprecation policy** first (no compiler warning mechanism yet).
- Policy: keep aliases for at least one phase, document migrations, and update examples/templates immediately.
- Rationale: minimal implementation risk while improving API cleanup discipline.

## Semantics/codegen hardening status

- No semantic or codegen engine changes in this milestone.
- This milestone intentionally establishes policy/docs before deeper invariants and snapshot expansion.

## Mini-project acceptance strategy

- Documented service testing layers and acceptance expectations in `docs/testing_services.md`.
- Next milestones should add `projects/` acceptance samples and wire CI checks for them.

## Compatibility corpus organization notes

- Updated `docs/compatibility.md` to classify fixture folders by contract strength (`stable`, `regression`, `experimental`) to reduce accidental compatibility drift.

## Contributor workflow changes

- Updated `docs/contributing.md` with Phase 9 workflow expectations and doc touchpoints.
- Updated `AGENTS.md` milestone context from Phase 8 to Phase 9.

## Known limitations

- Compiler/runtime behavior for `kiro build/run/test` remains placeholder in this repo slice.
- No warning emission for deprecated APIs yet.
- Generated-Go snapshots and realistic mini-project acceptance remain to be expanded in later Phase 9 milestones.

## Recommended next Phase 9 scope

1. Add focused sema invariants tests (lists/maps, `R[T,E]`, optionals, return paths).
2. Add representative generated-Go snapshot tests with stable normalization.
3. Add `projects/tiny-service`, `projects/tiny-cli`, `projects/tiny-file-tool` acceptance checks and CI wiring.
4. Improve diagnostics regression fixtures for high-friction type/error cases.
