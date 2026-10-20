# Phase 19: a versioned language reference for unfamiliar-language agents

No training-time familiarity with Kiro is assumed. Before model benchmarking, agents need
readable, versioned contracts and evidence that the supplied examples agree with execution.
This phase changes documentation/discovery/onboarding, not language syntax.

- Added `docs/llm/references/language.md` as the primary current language reference, bundled
  in the CLI and new scaffolds. `docs/language-reference.md` links the same document.
- Consolidated source/project rules, declarations/types, canonical grammar, scope/order,
  keyed calls/defaults/functions, results/optionals, matching, loops, collection transforms,
  effects/concurrency, JSON, executable builtin signatures and explicit unsupported forms.
- Added conformance tests that extract complete programs from the reference itself, check
  valid/rejected programs and execute assertions/runtime-failure cases. Builtin base
  signatures are checked against the executable registry; nested bundled projects now
  also check and run their tests.
- `kiro describe --reference` retrieves the exact installed document; `--json` adds its
  guide revision, schema/path and SHA-256. Normal describe includes identity without text.
- Scaffold metadata adds guide_version and language_reference identity while preserving
  existing compiler/skill release fields. AGENTS and the compact guide direct agents to
  the reference; legacy overview pages defer to it and concurrency guarantees are qualified.
- Guide revision: phase19-language-reference. No source migration is required.
- Added `docs/research/benchmark_onboarding.md`: pin compiler/reference/model/task inputs,
  disclose context acquisition and documentation costs, use fresh trials, enforce tool
  conditions, withhold acceptance checks outside the implementing agent's environment,
  retain failures and separate measured results from interpretations in the article/video.

Reference examples test bounded concrete behaviors, not all possible source programs.
Checking remains incremental; effects/concurrency and typed lowering have their documented
limits. Analysis stays opt-in. No model benchmark or Go comparison has been run in this phase.

## Validation

- `go test ./...` and `go vet ./...` pass.
- `go test -race ./docs/llm -count=20` passes, including reference execution.
- The reference has 20 extracted programs: 8 checking/executing successes, 11 expected
  compiler rejections with diagnostic codes, and 1 checked runtime failure. All executable
  builtin base signatures match the registry. Both nested bundled example projects pass.
- Native CLI tests verify exact text/JSON retrieval, identity/hash agreement, invalid option
  rejection, vendoring and scaffold guidance. Fresh hello/service scaffolds pass check,
  lint, read-only formatting and native tests with matching reference identities.
- All 61 compatibility fixtures and checks for all 33 example projects pass.

These are compiler/documentation checks. Model maintenance benchmarks and the Go comparison
remain subsequent work; there are no new agent-performance results to present yet.

Phase 20 adds the workload, deterministic evaluator and external-driver trial harness.
See `PHASE20_NOTES.md`; model trials remain blocked by missing execution prerequisites.
