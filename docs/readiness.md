# Readiness checklist

Kiro is still experimental. This page defines what is solid today and what must improve before broader recommendation.

## Solid today

- Deterministic parser/formatter workflows.
- Module/import project loading for small projects.
- Compatibility corpus workflow (`kiro compat`).
- Generated-Go inspection flow (`kiro inspect go`).
- Starter scaffolding (`kiro new hello|service`) for small repos.

## Still experimental

- Full semantic coverage for advanced typing edge cases.
- Runtime behavior behind placeholder command paths (`build/run/test`).
- Stability of exact generated-Go structure as a public contract.
- Large multi-module project ergonomics.

## Quality bars before wider use recommendation

- Green CI across tests, compatibility corpus, templates, and project acceptance checks.
- Representative generated-Go snapshots for key language features.
- Stable and explainable diagnostics for common mistakes.
- Documented stdlib naming/shape consistency with clear deprecation handling.

## Acceptable breakage during current development

- Internal compiler refactors with no user-facing behavior change.
- Minor diagnostic wording adjustments when intent remains clearer.
- Generated-Go formatting/layout shifts unless guarded by selected snapshots.

## Not acceptable breakage

- Silent source compatibility breaks in compatibility corpus stable fixtures.
- Formatter nondeterminism.
- Template output drift without docs/tests updates.
- CLI command behavior changes without README/docs updates.
