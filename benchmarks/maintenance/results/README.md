# Recorded preparation evidence

These are deterministic harness checks, not model benchmark results.

- controls.json: 16 evaluations; baseline contracts pass, all five unchanged task copies
  fail changed requirements, all five reference fixes pass, all five incorrect fixes fail.
  It includes case groups/failures, API/structure checks, process outcomes and snapshot hashes.
  Full verbose diagnostics are omitted from this compact record; regenerate with selftest.
- preflight.json: blocked because Docker and an external model driver are unavailable.
- prepared-plan.json: a 30-trial paired schedule, prepared but never executed. Model is unset.
  It was prepared from the Phase 19 compiler plus an uncommitted benchmark worktree, so git
  commit/dirty metadata is supplemented by exact input hashes. Prepare a new cohort for
  actual model execution rather than treating this preparation record as a completed run.

No tokens, model timings, comparison scores or Go results are inferred from these controls.
The same setup agent authored the fixture/oracle; an independent agent audit remains pending.
Docker path/mount argument tests pass, but actual container isolation remains unverified here.

## Go v1 counterpart preparation

`go-v1-controls.json` records new deterministic preparation evidence. Both baselines pass
519 HTTP checks each: the unchanged 353 shared cases plus 166 independent schema/numeric
checks. Go and Kiro each satisfy the expected outcomes across 16 evaluations: baseline,
five unchanged task copies, five reference fixes and five incorrect fixes. Go control
reports separate HTTP, compile-probe API and structural acceptance. Unchanged/unused
payment extraction passes all HTTP cases while failing architecture. This does not prove
arbitrary candidate correctness. Source hashes identify the versioned fixtures/tasks/runner.

Go tests/vet pass for both the root repository and nested fixture. All 31 Python harness tests pass and
cover preparation, compatibility aliases, dependency ownership, transitive imports and
blocked isolation with no host fallback. Two setup subagents contributed evaluator and
independent parity/structure review; their work is preparation, not model trial data. This
reduces some shared assumptions but does not constitute a complete independent oracle audit.

The development host has a Docker CLI but no running daemon, and no external driver was
supplied. Container orchestration has argument/failure-path tests, not end-to-end runtime
validation. No model candidates were run. Existing native-agent report times remain
unvalidated instruction-constrained pilot observations, separate from this evidence.
