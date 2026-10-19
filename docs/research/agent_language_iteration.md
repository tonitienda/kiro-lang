# Revisiting Kiro for coding agents

Date: 2026-10-19. Starting repository revision: `681c6c1`.
Status: initial implementation audit, runnable example and experiment design.
No comparative model runs have been performed. This document is not evidence of a Kiro advantage.

## Research question

Can a small explicit language help coding agents implement, review, audit, repair and extend
software more reliably or cheaply than an established language with equally good guidance?

For this experiment, human code review is outside the required workflow. Implementation,
review, audits and creation of deterministic verification tools are performed by agents.
Human source readability is not an acceptance criterion. Requirements still need an explicit
meaning; a human requirement author or article reader is not a required code reviewer.

Treat Kiro-specific tooling as something to build and evaluate, not as a mature ecosystem
we can assume already exists. The current compiler, formatter and probes are starting points.

Keeping Go as the backend is compatible with this investigation. The source language,
compiler feedback and agent tools are the variables of interest. Native code generation
is a separate engineering project and should not block the first language experiments.
However, current executable packaging and direct Go transpilation must be distinguished.

## What “optimal” means

There is no model-independent optimum. State the workload, model revision, tools,
documentation, context budget and acceptance criteria first. Compare trade-offs rather
than merging every dimension into an arbitrary weighted score.

Primary objective: maximize the fraction of tasks satisfying independent behavior and
regression checks within a fixed time/token budget, including initial language-learning cost.
Then compare the total cost and latency needed to reach those accepted results.
For a deployment decision, also report the resource cost of failed attempts. The measured
system includes implementation agents, review/audit agents and deterministic tools created
by agents; their aggregate resource use counts toward efficiency.

| Dimension | Measurement | Common misleading shortcut |
| --- | --- | --- |
| Behavioral reliability | Independent acceptance checks; regression and invariant violations | Compiling or passing agent-written tests alone |
| Total agent cost | All available input, output and reasoning tokens; language-guide tokens; tool output; billed cost | Source tokens only |
| Repairability | Failed checks, repair rounds, time to accepted fix, errors left after budget | A diagnostic that merely sounds helpful |
| Change robustness | Acceptance at each new requirement; cumulative regressions; abandoned attempts | A successful initial implementation |
| Context needs | Files/bytes read, retrieved tokens, documentation reads; controlled budget reductions | Shorter source automatically means less context |
| Agent review and audit | Detection and localization of seeded defects; false positives; missed violations; audit tokens/time | Reviewer approval or agreement between agents |
| Verification tooling | Tool construction/repair cost; defect detection on withheld cases; incorrect acceptance/rejection | Treating determinism as evidence that a tool is correct |
| Runtime practicality | Separate CPU, memory, startup and binary-size measurements | Assuming a Go executable has native Go business-logic performance |

Token counts depend on the actual model tokenizer and APIs. Do not equate characters,
lines or a single tokenizer's count with a universal efficiency score. Reasoning usage may
not be exposed by every provider; record that as unavailable rather than estimating it silently.

A useful Kiro result may be better reliability at equal cost, equal reliability at lower cost,
or a clearer failure boundary. It may also be that better Go tooling supplies the same benefit.

### Verification without a human code reviewer

A deterministic tool gives repeatable answers; the correctness of those answers still needs
evidence. An implementing agent, reviewing agent and agent-authored checker can share the
same mistaken interpretation. Separate roles alone do not establish independence.

- Express requirements as observable behavior and invariants before implementation.
- Have acceptance tools authored from that specification without access to the candidate
  implementation, then freeze them for scored runs. Keep evaluator fixtures outside the
  implementation agent's workspace; isolate execution so it cannot modify the evaluator.
- Challenge the evaluator with deliberately incorrect programs, boundary cases and changed
  inputs. Report which defect classes it detects and which it misses. Withheld mutants must
  include cases not used while building the tool; a mutation score is not a correctness proof.
- Where practical, triangulate with separately constructed Go/reference implementations,
  properties, metamorphic relations and exhaustive checks over bounded inputs. Agent-created
  reference code is another fallible implementation, not an unquestioned oracle.
- Score review/audit agents against known defects and false-alarm cases. Preserve unresolved
  disagreements and the evidence behind decisions rather than replacing them with majority votes.

This is a proposal for an automated evidence chain, not a claim that testing can prove arbitrary
programs correct. Specify what each check establishes and leave unverified properties explicit.

Without a human-review constraint, candidate interfaces can prioritize typed AST operations,
machine-readable contracts, stable symbol identifiers, dependency/effect queries and structured
diagnostics. A textual syntax can remain the current input format. Compare text and structural
editing only after tools exist, and include serialization, schema-learning and repair costs;
JSON or AST output is not automatically more token-efficient or more reliable.

## Current implementation: observed boundaries

The language already has useful design ingredients: one function form, canonical formatting,
explicit operational effects, results distinct from optionals, visible concurrency constructs,
project conventions and a compact agent skill. Their benefits remain hypotheses to measure.

The implementation does not yet enforce the full apparent contract:

| Area | Evidence in the current source | Consequence |
| --- | --- | --- |
| Executable backend | `internal/buildsys/buildsys.go` serializes function bodies and embeds `runtimekit` into a generated Go project | Executables run a Go-hosted interpreter, not directly lowered typed business functions |
| Go inspection | `internal/codegen/gogen.go` emits stub functions and string fields | Inspection is not yet a faithful view of executable business logic |
| Body checking | `project.Load` parses files, resolves imports and checks effects; bodies remain largely text in the frontend AST | A successful `check` is not a full type/semantic guarantee |
| Runtime syntax | The body parser has a narrower grammar than the frontend/guide | Even documented code can pass `check` and then fail on execution |
| Results | `evalUnwrap` converts failed results to Go errors | Failed `?` does not currently compose as an ordinary returned `Err` |
| JSON | `json.decode[T]` labels maps without checking declared fields; encoding adds `_type` | Invalid payloads can cross a nominally typed boundary; runtime metadata leaks into JSON |
| Concurrency | `group` executes a block; spawned futures are not registered with the group; context helpers return strings | Syntax alone does not establish join/cancellation/timeout guarantees |

The last row is source inspection, not a concurrent stress-test result. Do not describe the
current effect checker as a capability sandbox or the concurrency syntax as an enforced lifecycle.

### Reproducible contract probes

```bash
go build -o kiro ./cmd/kiro
python3 scripts/probe_agent_contract.py --kiro ./kiro
```

The script emits observations as JSON and does not assert that today's gaps should remain.
All four probes passed `check` on the starting implementation:

| Probe | Execution observed | Intended question |
| --- | --- | --- |
| Function declares `i32` but returns a string | Succeeds and prints the string | Does a signature constrain behavior? |
| Decode `{}` into a type with required `quantity:i32` | Succeeds; encoded result has only `_type` | Does typed decode validate required fields? |
| Sum a list with documented `for` syntax | Fails: `expected IDENT, got NEWLINE` | Do checked examples actually execute? |
| Compare a propagated failure with `Err(...)` in the caller | Aborts with the domain error before the assertion | Can callers inspect a failure after propagation? |

These are implementation findings, not measurements of agent performance. Fixes should
be accompanied by negative compiler/runtime tests, not just more optimistic documentation.

## Larger example and incremental requirements

The first slice is [the quote service](../../examples/quote_service/README.md): seven source
modules and 14 domain/handler tests. It exercises modules, HTTP methods, typed-looking JSON,
pure rules, integer cents, optional fields and error-producing leaf operations.
It deliberately stays within the executable subset. Its two product positions are a real
scope limit, not a substitute for arbitrary collection handling.

Use it as an integration seed and a story viewers can follow. It is not an independent
benchmark sample: this implementation and its tests were developed together.

| Checkpoint | User-visible requirement | What it stresses |
| --- | --- | --- |
| A: implemented | Quote one item and optional add-on; standard/express delivery; optional free-shipping coupon | Composition, module boundaries, optionals, basic failure paths |
| B: conformance gate | Required/wrong-type JSON errors and domain errors become explicit 400/422 responses; no `_type` in public JSON | Typed decoding, result propagation and recovery |
| C | Arbitrary cart items and quantities, bounded totals, invalid quantity rejection | Working iteration, arithmetic, validation, overflow policy |
| D | Country delivery rules, coupon restrictions and a versioned quote response | Local policy changes without unrelated regressions |
| E | Persist orders with idempotency keys and explicit order-state transitions | Stateful invariants, filesystem effects, retries, restart behavior |
| F | Independent stock/shipping providers with timeout, failure and cancellation tests | Actual concurrent I/O and enforced task lifecycles |

Checkpoint B is prerequisite to a public HTTP playground. Do not paper over it by
representing errors as successful quotes. Before C, specify integer width/overflow behavior;
before E, specify duplicate-request atomicity; before F, specify join and cancellation behavior.
Use deterministic fakes for providers and clocks so a network outage cannot decide a score.

## Research that informs the experiment

These works motivate test design; none establishes that Kiro will improve agent performance.

1. **Can LLM Coding Assistants Support Emerging Programming Languages? An Empirical Study on
   Cangjie**, Cheng et al., arXiv v2, August 2026.
   [Paper](https://arxiv.org/html/2603.14501v2).
   Compares prompt references, retrieval and agent loops while recording correctness and token
   costs. A concise syntax reference performs well among prompt methods; more expensive agent
   loops can improve absolute correctness. Implication for Kiro: ship a small verified reference,
   count its cost, and measure unfamiliar-language errors separately from business-logic errors.

2. **Frontier Coding Agents Use Metaprogramming to Adapt to Unfamiliar Programming Languages**,
   Sharma, Thorat and Chopra, June 2026, preprint.
   [Paper](https://arxiv.org/abs/2606.10933).
   In esoteric-language tasks, stronger agents sometimes build generators in a familiar language.
   Implication: decide whether generators are permitted, preserve tool traces and check what the
   agent actually authored. Esoteric-language results are not direct predictions for Kiro.

3. **SlopCodeBench: Benchmarking How Coding Agents Degrade Over Long-Horizon Iterative Tasks**,
   Orlanski et al., March 2026, preprint.
   [Paper](https://arxiv.org/abs/2603.24755).
   Evaluates successive extensions to the agent's own prior code and tracks structural changes.
   Implication: preserve implementation history through checkpoints and measure future changes,
   not just initial correctness. Complexity/duplication metrics are proxies, not a full quality score.

4. **Type-Constrained Code Generation with Language Models**, PLDI 2025.
   [Primary conference page](https://pldi25.sigplan.org/details/pldi-2025-papers/25/Type-Constrained-Code-Generation-with-Language-Models).
   Studies type-guided constrained decoding beyond syntax constraints. Implication: a small,
   enforceable type core may help generation and feedback. Kiro does not currently implement
   this decoding method; ordinary compiler checking must not be presented as the same technique.

## Comparison protocol

### Separate language, tooling and maturity

Use at least these three arms on identical externally observable requirements:

1. Idiomatic Go with its normal formatter, tests and concise project guidance.
2. Go with a restricted documented style, compact API guide and equivalent agent-facing checks.
3. Kiro with its compact language package, formatter, checker and executable tests.

Arm 2 asks how much benefit comes from conventions and tools without creating a language.
Kiro must still meet the same behavior. Do not grant it custom domain helpers unavailable to
the Go baseline. Backend/library maturity is a practical cost; label it rather than claiming
every observed difference is caused by syntax.

All three arms use agent implementation/review/audit and deterministic validation, with no
required human source-review stage. Tool creation needs two separate accounting views:

- **Bootstrapping:** start from a declared toolkit/specification and count agent effort to
  construct, validate and repair missing tools. Existing Go tools are declared dependencies;
  requiring agents to rebuild Go's ecosystem would answer a different question.
- **Steady state:** pin the validated tool versions and compare subsequent application tasks.
  Report Kiro's one-time setup cost separately, and show any amortization assumptions explicitly.

The existing three arms primarily compare steady-state application work. A bootstrapping
study is a separate experiment, with equal resource ceilings and explicit initial dependencies.
Neither view silently grants Kiro free tools. Offline language/tool development is also distinct
from per-run tool construction and application maintenance.

Later, run within-Kiro ablations for specific ideas: diagnostic format, compact vs full docs,
signature/context views and explicit contracts. Do not change syntax, model and tool budget
simultaneously and then attribute the outcome to a language feature.

### Pilot first, then a broader study

- Pin compiler commits, model identifiers/settings, agent versions, prompts and tools.
- Begin with two model configurations of different capability/cost and 3–5 independent runs per
  arm for a pilot. This estimates feasibility and variance, not a publishable general advantage.
- Include creation, bug repair, feature extension and code explanation/audit tasks.
- Pin reviewer/auditor configurations as well as implementation agents. Account for all roles;
  evaluate known-defect detection and false alarms separately from application acceptance.
- For extension tasks, each trajectory retains its own prior solution. Never substitute the
  author's reference implementation between checkpoints.
- Freeze acceptance checks before scored runs. Keep public development tests separate from
  withheld tests, and verify that deliberate incorrect solutions fail the evaluation. Evaluators
  may be created by agents; their provenance and validation evidence must be recorded.
- Give the agent only the current checkpoint requirement. Split development workloads from
  held-out evaluation workloads; use additional domains before making broad language claims.
- Give arms the same model configuration, external interfaces, wall-time/token ceilings and
  policy on network access. Count language setup and documentation reads. Record warm-start
  and cold-start runs separately.
- Randomize run order, use clean workspaces and preserve prompts, patches, traces and failures.
- Report every attempted run, per-task results, uncertainty and budget-exhaustion outcomes.
  Distinguish first-attempt success from success after a repair budget. Repeated runs on one
  task are not additional independent task samples.
- Define whether metaprogramming is allowed. Generated solutions must execute the scored
  business logic in the target language; delegating everything to an embedded Go/Python
  implementation does not answer the source-language question.

No provider benchmark jobs, paid API runs or advertising campaigns are part of this first slice.

### Minimal run record

Record: run ID; task/checkpoint; arm; source/compiler/model/agent versions; random seed where
supported; documentation bundle hash; budgets; first-check outcome; acceptance and regression
outcomes; tool calls; repair rounds; wall time; input/output/reasoning usage where available;
billed cost; files read/changed; final patch; and infrastructure/compiler/model failure category.
Also record role-specific usage, tool-construction/repair effort, checker versions/provenance,
validation mutants, audit true/false positives and missed defects where ground truth is known.

Report final accepted-run cost distributions alongside total spend divided by accepted runs,
so failed attempts do not disappear from the efficiency story. Do not imply an infinite-cost
zero-success arm has an ordinary numeric cost-per-success estimate.

## Optimization priorities

1. **Make the contract true.** Align frontend and executable grammars; implement compositional
   results, schema checking and useful source locations; verify lifecycle claims. Stable error
   codes and structured diagnostics are candidates, not existing features.
2. **Make changes local.** Test whether explicit signatures, effects and module/API summaries
   reduce retrieval and repair work. Consider an agent-facing dependency/contract query before
   inventing another surface notation. Optimize these interfaces for agent consumers; human
   source-review convenience is outside this experiment's objective.
3. **Reduce accidental choice.** Preserve canonical formatting and a verified compact guide.
   Measure guide coverage and invented-API failures before shortening it further.
4. **Tune syntax last.** Compare token counts and semantic mistakes with real tokenizers and
   several models. Extra explicit tokens may be worthwhile if they prevent repair rounds;
   punctuation changes alone may add novelty without measurable benefit.

Keep a Go backend through these stages. Direct Go lowering becomes valuable for faithful
inspection, static guarantees and runtime performance; treat it as its own measured milestone.

## Article, video and invitation

Working question: **“Would coding agents benefit from their own programming language?”**

Suggested story:

1. Revisit the original Kiro idea and state a falsifiable hypothesis.
2. Walk through the quote service and one concrete change request.
3. Show where explicit intent helps, then run the checker/runtime disagreement probes.
4. Repair a small contract gap and show its effect on the development loop.
5. Compare Go, guided Go and Kiro with the same requirements and published method.
6. Explain measured trade-offs and invite independent replication or counterexamples.

Before comparative runs, publish this as an experiment in progress, with no speed or cost
claim. For the video, the quote request, policy module, diagnostic and diff are useful visual
anchors. Do not present a manually edited demonstration as an autonomous-agent benchmark.

Before inviting people to try it, provide a tested pinned install, a short quick start, a
supported-syntax reference that actually executes, known limits, and an issue template for
reproducible failures. Check naming/search confusion with other projects using “Kiro” before
promotional spend. Verify release support on the advertised platforms rather than assuming it.
Start with an open invitation and concrete tasks; collect completion, failures and feedback
from volunteers before deciding whether paid promotion is useful.
