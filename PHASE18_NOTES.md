# Phase 18: opt-in analysis and refactoring evidence

Agents explicitly call analysis tools; normal compiler/check/lint/LSP workflows do not run
them. No human source-review gate is introduced.

- Added `kiro analyze graph/impact/metrics/cycles/boundaries/rules/slice/coverage/mutate`.
  Reports reuse kiro.tools.v1 and include kiro.analysis.v1 data with source revisions.
- Graphs track known/possible calls, aliases/partials, callbacks, types, constants, effects
  and unresolved calls. Impact, SCCs and complexity metrics share that evidence.
- Candidate boundaries use deterministic weighted greedy modularity with degree-normalized
  affinity and a node budget. They explain entry points/crossing edges and never move code.
- Explicit architecture JSON rules support opt-in hook/CI checks. No auto-discovery or
  silent integration with the compiler/linter is added.
- Context slices follow local data/control/loop-carried dependences, with optional known
  callee result expansion. They are conservative context queries, not executable reductions.
- Kiro function/statement/branch coverage has per-test attribution, fresh interpreters,
  subprocess timeouts and step limits. No Go line-coverage proxy is used.
- Targeted bounded operator mutation distinguishes invalid, killed, surviving and inconclusive
  cases; requires a passing exercised baseline and never edits candidate sources.
- Native builds exclude analysis/coverage/mutation implementations and retain disabled
  observation stubs. Ordinary checking does not construct analysis graphs/catalogs.
- Added tests for dependency precision/unknowns, slices, deterministic clusters, rules,
  coverage attribution/interpolation, budgets, source preservation and native CLI workflows.
- Agent guidance invites analysis when useful without making it a mandatory gate. Compact
  guide version: phase18-opt-in-analysis. Full contracts and costs: docs/analysis-tools.md.

No interprocedural points-to/heap analysis, field-sensitive semantic slicing, global
architectural optimum, history mining, automatic refactoring, security sandbox or measured
agent-performance advantage is claimed. Unknown dependencies and concurrency limitations
remain explicit; users choose their own analysis cadence and hooks.

## Validation snapshot

- `go test ./...`, `go vet ./...` and race checks for runtimekit, analysis and project pass.
- All 61 compatibility fixtures and all 33 example project checks pass.
- Fresh hello/service projects pass check, lint, read-only format and native test with the
  phase18-opt-in-analysis guide.
- Every analysis operation runs on quote_service, including explicit architecture rules
  and impact with saved, revision-matched coverage. Its 20 tests cover 13/15 production
  functions, 51/56 attempted statements and 40/44 catalogued branch outcomes. Both supported
  pricing.quote arithmetic mutants are killed; this is a small smoke case, not adequacy proof.
- One local alternating 40-sample check run on quote_service measured median 6.116 ms for
  Phase 17 versus 6.316 ms for Phase 18 (p95 7.888/7.710 ms), including process startup.
  This small warm-cache observation does not establish a speed difference or a general
  overhead bound. Opt-in operations on the same project took roughly 11–17 ms for static
  queries, 35 ms for coverage and 98 ms for the two-mutant run. Costs vary with project,
  tests, machine and effects; no analysis is scheduled by ordinary check.
