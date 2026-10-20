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
