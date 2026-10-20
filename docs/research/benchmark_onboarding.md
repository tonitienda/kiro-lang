# Benchmark onboarding and reporting protocol

Status: protocol only. No model benchmark or Go comparison has been run for Phase 19.
The next experiment evaluates agent maintenance with an unfamiliar language and explicit
reference/tool access. Agents implement, audit and verify; there is no human code-review
gate. Decisions about task requirements and acceptance criteria must precede implementation.

## Freeze the experiment inputs

Record a repository commit/tree, compiler binary SHA-256 and release/toolchain version,
`kiro describe --json`, exact model identifier/configuration, harness version, prompts,
resource limits and allowed tools. Development release labels such as `dev` are insufficient.

Retrieve `kiro describe --reference --json` and retain the exact reference plus its SHA-256
and guide revision. Include the entire bundled SKILL/manifest/examples snapshot; hash the
individual files or archive. The scaffold's `.kiro/version.json` identifies the compiler
release label, guide revision and reference hash; it does not identify every example by hash
or a compiler git commit. Store those additional identities in the experiment record.
Do not update docs/compiler midway through a trial series without creating a new cohort.

## Give the agent an explicit bootstrap

Every fresh Kiro trial receives the same startup instruction:

> This project uses Kiro. Do not assume syntax from another language. Read AGENTS.md,
> .kiro/skill/SKILL.md and .kiro/skill/references/kiro.json before editing. Use
> .kiro/skill/references/language.md as the primary language reference, reading the sections
> needed for the task. Retrieve the installed reference with kiro describe --reference
> --json if a vendored copy is unavailable. Compare its hash/revision to the pinned run.
> Use kiro describe --json for capabilities and kiro symbols . --json for project contracts.
> Follow the task's allowed-tool list; analysis is optional when available. Resolve work
> with agents and deterministic tools, without waiting for human source review.

Provide documentation through paths the agent can actually access. References in a prompt
without readable files/commands are not onboarding. A stale vendored reference versus a
pinned compiler is an environment failure to repair before starting the measured task.
Record what documentation the agent actually reads and which examples/commands it uses.
Do not silently prefill Kiro knowledge in one condition or reuse a previous trial's context.

For the first experiment, onboarding is part of the measured fresh-session task, including
failed syntax attempts, repair, documentation reads and tool calls. Optionally report a
separate warm-session series with its preparation cost and context explicitly disclosed.
Do not subtract documentation cost after seeing results or describe the warm series as a
cold-start result. Repository documentation size is not the same as model tokens consumed.

## Conditions and controls

1. Kiro with compiler, formatter, tests and declaration/builtin discovery.
2. Identical Kiro task/reference/model with optional `kiro analyze` tools available.
3. Later: Go implementation with equivalent externally specified behavior and its declared
   documentation/tool environment. This is a separate comparison from the tool ablation.

The first two conditions use identical source fixtures, guides, task wording, deterministic
acceptance behavior, model settings and resource limits. The harness enforces unavailable
commands; an instruction alone is not proof that an analysis-free condition stayed so.
The guides may describe analysis in both conditions, but the allowed-tool list is explicit.
Do not require every analyzer: whether agents discover and use tools is part of the question.

Use multiple tasks and independent fresh trials. Predeclare repetition counts, randomize or
counterbalance condition order, and retain unsuccessful/time-limited runs. Report the sample
size and uncertainty rather than selecting the best trajectory. Do not provide hints or
repair code mid-trial. Any extra agents, model calls or deterministic verifier preparation
are disclosed and included in their appropriate cost categories.

For Go, make acceptance behavior language-neutral before porting tasks: request/response
contracts, state transitions and failure cases, not a required Kiro variant or module name.
Document ecosystem differences such as JSON validation and testing facilities. Go prior
training familiarity cannot be made equal to Kiro merely by giving both a manual. Report
results as this model + language + docs + tools + tasks, not an intrinsic universal language
ranking. Compare correctness and total effort before token density or source line counts.

## Acceptance and independence

Prepare acceptance checks before the implementation trial, using an agent with the task
specification and a verification goal. Audit the checks against that specification separately.
Use negative controls: unchanged baselines must fail changed requirements, and deliberately
incorrect implementations must be detected. Shared agent/model mistakes remain possible.

Keep held-out checks and their implementation outside the implementing agent's readable
filesystem/tool environment. Merely omitting their filename from its prompt is insufficient.
The existing public maintenance script's acceptance cases are reproducible fixed checks;
they must not be advertised as unseen tests for an agent that can read that script.
Record provenance and access boundaries. Prefer externally observable assertions alongside
typed unit contracts, and avoid judging an architecture by the clustering tool's own score.

If a task exposes a reference/compiler disagreement, retain its full trace and classify it.
For a defective task/environment, fix and rerun as a new version, reporting the excluded
original and reason. Do not silently attribute undefined behavior to model incompetence or
change acceptance rules to accommodate one candidate. A passing documented example proves
that example, not a complete language specification or independent correctness.

## Record and publish outcomes

Per trial, retain:

| Field | Meaning |
| --- | --- |
| Condition/task/trial IDs | Stable identifiers, task version and starting source hash |
| Model/harness/compiler/reference IDs | Reproducible configuration and documentation hashes |
| Correctness | Held-out behavioral passes, regressions and runtime/contract failures |
| Completion | Passed, failed, timed out, budget exhausted or environment/spec defect |
| Total usage | Input/output tokens, cached usage and cost when exposed by the provider |
| Time | End-to-end wall time plus agent/tool subcategories when observed |
| Context acquisition | Files/bytes or tokens actually read, documentation reads and tool output |
| Repairs | Compiler/test failure-and-repair cycles, using a predeclared counting rule |
| Tool behavior | Invocations, duration, output size, limits/unknowns and observed use |
| Change footprint | Changed symbols/files and regressions; not a goal to minimize blindly |
| Raw evidence | Transcript, commands, reports, final diff and evaluator output |

Do not estimate unavailable token/cost data as measured data; mark it unavailable. Tool
subprocess costs belong in total elapsed time; a cheap model run with expensive mutation
still consumes resources. Separate one-time compiler/tool construction costs from per-task
costs and disclose both scopes. Do not claim test coverage, low complexity, modularity or a
killed-mutant percentage as correctness by themselves.

For the article/video, show at least one full maintenance task, the supplied language guide,
what the agent inspected, a failure/repair and the acceptance result. Label each claim as
measured, an interpretation of a trace, or an untested hypothesis. Publish the fixtures,
versioned docs, run procedure, unsuccessful trials and limitations alongside aggregate
results. The single quote-service smoke run in Phase 18 is a tooling check, not this benchmark.

## Phase 20 implementation status

The [maintenance suite](../../benchmarks/maintenance/README.md) implements a larger workload,
five tasks, HTTP/API/structure evaluation, control validation and a paired external-driver
harness. Deterministic controls are recorded separately from model trials. Actual model
execution is blocked in the development environment by missing Docker/model transport;
Docker end-to-end validation and a separate agent audit of the oracle remain outstanding.
The current language reference remains Phase 19; no model or Go advantage is measured.
