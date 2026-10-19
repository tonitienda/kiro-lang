# Phase 12: agent-only core reassessment

- Agents implement, review, audit and create deterministic validation tools; human source
  review is outside this experiment's objective.
- `when (subject) { pattern => { ... } }` is the only supported case-header/block form.
  Old unbraced headers fail with a migration diagnostic. Parentheses remove subject/record
  literal ambiguity, and braces let execution continue after a case block.
- Updated examples, compatibility fixtures, service scaffolding and the bundled skill.
  The compact manifest now identifies `phase12-agent-core`; regenerate downstream snapshots.
- The formatter preserves string contents while normalizing surrounding non-literal tokens.
- `docs/research/agent_core_reassessment.md` distinguishes implemented changes from proposed
  tagged optionals/unit, typed errors/exhaustiveness, named calls and structural agent tools.
- Layered verification deliberately varies inputs/goals and measures overlapping detections,
  false alarms and escaped defects; different agents are not assumed statistically independent.
- Phase 10's results/effects/optionals/concurrency concepts remain the design base. Existing
  body/type/JSON/result/concurrency enforcement gaps remain until explicitly implemented.
