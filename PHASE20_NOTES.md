# Phase 20: maintenance benchmark fixtures, evaluator and trial harness

This phase prepares an executable experiment, not evidence of model performance. Agent
implementation/audit and deterministic tools remain the workflow; no human review gate is
introduced. Language/compiler behavior and the Phase 19 language reference are unchanged.

- Added benchmarks/maintenance/workload: ten modules and six public tests for catalog,
  stock validation, pricing/shipping, checkout/payment policy, reservation status, refund
  calculation and HTTP adaptation. It is stateless/in-memory, not a real order/payment system.
- Five independent versioned tasks cover threshold discounts, expiry boundary repair,
  partial refunds, payment-module extraction and a typed declined-payment outcome.
- A Python oracle checks 353 HTTP cases per task against the built service, plus nine
  pinned public function/record contracts. Boundary/type checks are reported separately.
  Candidate tests alone cannot establish success.
- Added deterministic control validation: baseline contracts, five unchanged task copies,
  five reference patches and five incorrect patches. These are not agent trials.
- Added seeded paired plans (default 30 trials), pinned compiler/reference/input hashes,
  fresh workspaces/drivers, a JSONL model transport protocol and per-trial evidence/summary.
- Model actions use a bounded broker for workspace files and Kiro commands; core rejects
  analyze. Candidate processes require network-disabled Docker containers with resource
  limits, read-only compiler/docs and no host repository/oracle mount. Expected values stay
  outside the candidate even during evaluation; only requests/actual responses cross the
  fixed HTTP transport. No host-execution fallback exists for model candidates.
- Reports retain transcripts, failed attempts, byte/tool/timing metrics, optional provider
  telemetry, verification recovery cycles, API/behavior/structure outcomes and final hashes.
  Scripted/unidentified sessions are excluded from model aggregates; missing usage/cost is null.

Validation: Go tests/vet, Python broker/transport/oracle tests and deterministic fixture
controls. See benchmarks/maintenance/results for the recorded control summary and preflight.

Execution limits: this environment has no Docker CLI or configured external model driver;
bubblewrap namespace creation was also denied by the host. No model trials were run.
Docker orchestration is covered by argument/mount tests but is not end-to-end verified here.
The model-provider adapter is a trusted external component, not supplied/configured in this
phase. The oracle and fixture were authored by the same setup agent; an independent agent
audit is still needed to reduce correlated specification/checker mistakes. The later Go
fixture/adapter and structural-check mapping are not implemented yet. No performance or
language-comparison conclusion follows from harness controls.
