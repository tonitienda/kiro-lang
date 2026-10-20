# Explicit analysis for agents

Phase 18 provides opt-in evidence for maintenance and refactoring. None of these analyses
runs automatically during check, lint, formatting, LSP updates, build, run or ordinary test.
Agents choose commands when their expected value justifies the cost. Projects may invoke
selected commands in hooks/CI; no hook or architecture policy is installed automatically.

| Command | Evidence | Executes tests? |
| --- | --- | --- |
| `kiro analyze graph . --json` | Typed symbol dependency edges and local program dependences | No |
| `kiro analyze graph . --symbol pricing.quote --json` | Immediate dependencies/callers plus this function's operations | No |
| `kiro analyze impact . --symbol catalog.price --json` | Reverse dependency closure and candidate tests | No |
| `kiro analyze metrics . --symbol pricing.quote --json` | Statements, decisions, nesting, fan-in/out and unresolved calls | No |
| `kiro analyze cycles . --json` | Function/module strongly connected components with explanatory edges | No |
| `kiro analyze boundaries . --max-nodes 200 --json` | Explained candidate groups and crossing edges | No |
| `kiro analyze rules . --rules architecture.json --json` | Explicit dependency-policy violations | No |
| `kiro analyze slice . --symbol pricing.quote --value return --json` | Backward local data/control context with opaque call boundaries | No |
| `kiro analyze coverage . --timeout 10s --max-steps 100000 --json` | Kiro function/statement/branch execution, attributed per test | Yes |
| `kiro analyze mutate . --symbol pricing.quote --max-mutants 10 --json` | Selected tests' response to bounded operator mutations | Yes |

All commands first load/check the project. Static analysis is recomputed only on the explicit
call; there is no background graph service or implicit cache. Normal native builds exclude
the analysis/tracing/mutation implementations, retaining only disabled observation hooks.
The CLI binary includes the tools. This is a separation of execution paths, not a promise
of mathematically zero binary-size or hook-branch overhead.

## Reports, identities and limitations

Commands use the existing `kiro.tools.v1` stdout envelope, with `command:analyze`. Data has
schema `kiro.analysis.v1`, operation, source revision, result, `execution:explicit_opt_in`
and `runs_in_check:false`. Failures use nonzero exit codes with structured diagnostics and
no duplicate prose on stderr. Boundary budgets, architecture violations and failed test
execution are distinguished. Static reports have deterministic ordering and tie-breaking.
Runtime observations can depend on test effects and concurrency.

IDs use the same module-qualified function/method identities as symbols. Operations use
structural body paths, e.g. `pricing.quote/body/s0`; these are executable AST locations,
not original source lines. Use `kiro symbols` for declaration positions. A source change
may change paths; compare/save reports with their exact source revision. Full reports can
be large: save JSON externally and use focused graph/impact/slice queries for agent context.
No files are written by static analysis, and mutations never rewrite the candidate's source.
Tests themselves execute their ordinary declared effects.

## Dependency graph and impact

Edges distinguish call, reference, capture, callback, type, constant read, effect and
unresolved call dependencies. Local aliases and explicit partial targets are tracked;
constructor and method contracts use the runtime's resolution rules. Callback arguments
are potential dependencies even when execution is conditional. Returned closures, opaque
callable fields, unknown callbacks and transform plans remain visible as unresolved edges.
These are conservative possible relationships, not a trace of guaranteed calls.

There is no interprocedural points-to/heap/alias analysis, reflection inference or model of
all external builtin behavior. Unused import declarations are not dependency edges. Analysis
rejects colliding symbol identities rather than silently choosing one. `all_call_sites_resolved`
means the report has no unresolved call markers; it is not a whole-program soundness proof.

Impact follows incoming dependency edges, including type/callback relationships. Candidate
tests are statically connected `test_*` functions. Unknown edges remain in the report rather
than being used to assert that unlisted callers/tests cannot be affected. Observed tests can
supplement static candidates:

```bash
kiro analyze coverage . --json > /tmp/kiro-coverage.json
kiro analyze impact . --symbol pricing.quote --coverage /tmp/kiro-coverage.json --json
```

A stale source revision or wrong report type is rejected. Observed tests are evidence about
that run, not a complete test selection proof. Core runtime and compiler changes can affect
many programs without changing a project's source hash; retain compiler/guide versions too.

## Complexity and cycles

Complexity is a structural Kiro CFG estimate: one plus loop decisions, when alternatives
(including possible no-match), and result-unwrapping decisions. Nesting records nested
blocks/dispatch. Fan-in/out counts distinct call/callback/capture relationships, including
builtins and test callers; unresolved calls are counted separately. Primitive finite-domain
feasibility and equivalent branches are not solved. A large exhaustive error dispatcher is
not automatically a design defect, and splitting functions merely to reduce a metric may
increase navigation and coupling. Metrics are evidence, not mandatory thresholds.

Cycle reports use strongly connected components. Function cycles follow call/callback
relationships, which may be conservative; module cycles also reflect type/reference/data
relationships. Reported edges explain each component. Finding a component is deterministic;
choosing whether to merge it, split it or change an interface remains an architectural decision.

## Boundary candidates

The first implementation is deterministic greedy agglomeration using weighted, degree-normalized
undirected affinities and a modularity objective. Calls weigh 3, callbacks/captures 2, and
other structural relationships 1 before degree normalization. High-degree utilities therefore
do not dominate solely by having many users. Production functions and associated record/
variant/constructor nodes participate; test entry functions and external HTTP records do not.
Groups smaller than three nodes are omitted.

Each candidate includes symbols, current modules, external entry points, all crossing edges,
internal affinity and an explanation. This proposes a boundary; it does not move code, name
a domain, establish information hiding or prove reduced future effort. There is no universal
architecture score and no automatic refactor. Compare candidates against policy ownership,
shared representations, actual task impact and tests. History-based co-change analysis and
semantic/domain clustering are not implemented in this batch.

The default boundary budget is 200 nodes, explicitly raisable to 500. Exceeding it produces
`KIRO_ANALYSIS_BUDGET` with counts instead of silently truncating the graph. Larger analyses
need scoped support or a more scalable clustering algorithm in a later iteration.

## Explicit architecture rules

Rules are passed explicitly. They check reported symbol edges, not unused imports or runtime
behavior hidden behind unresolved calls:

```json
{
  "schema": "kiro.architecture.v1",
  "unresolved": "report",
  "rules": [
    {
      "name": "pricing-does-not-know-http",
      "from": "pricing.*",
      "deny": ["app.*", "http.*", "builtin:http.*"]
    }
  ]
}
```

Patterns are case-sensitive glob patterns over symbol IDs. Optional kinds limit a rule to
call/reference/callback/capture/type/read/effect edges. Unknown fields, invalid patterns,
repeated rule names and invalid kinds fail explicitly. Unresolved calls are always reported;
`unresolved:error` additionally makes any unresolved call fail this explicitly requested check.
A rule violation uses `KIRO_ARCHITECTURE`. No rules file is auto-discovered. See the quote
service's `architecture.json` for an example that can be run manually or in a chosen hook.

## Context slicing

`--value return` selects the function result; a binding name selects its parameter/local
definitions. `--direction forward` follows effects on dependent operations instead. Local
use-def, loop-carried and control relationships keep relevant conditions, inputs and updates.
Branch/loop merges deliberately overapproximate alternatives; shadowing may introduce extra
context. This is a context-selection tool, not a tool for deleting the omitted statements.

Calls remain explicit boundaries by default. `--interprocedural` expands known callees' result
dependences, with visited-node protection for recursion. This includes caller argument context
conservatively, without precise interprocedural parameter mapping, field sensitivity or heap
analysis. Effects and unknown external state are opaque. It does not produce a runnable,
semantics-preserving reduced program or support a field-specific criterion such as
`quote.total_cents` yet. Use the full type/call boundary when a precise slice is unavailable.

## Kiro execution coverage

Coverage instruments Kiro execution, not Go interpreter lines. Stable structural IDs cover
function entry, attempted statements, selected when arms/no-match, loop body/normal exit,
and Ok/Err unwrapping, including interpolation. Per-test attribution is recorded. Summary
counts exclude `test_*` entry functions; other helpers remain in the non-test totals.

Each test gets a fresh interpreter and failures are collected rather than stopping at the
first one. This differs from ordinary `kiro test`, which keeps its existing shared runtime
and fail-fast behavior. The worker is a separate process, inheriting the invoking working
directory and test effects. There is no filesystem/network sandbox or per-test filesystem reset.
Unjoined spawned work makes a report incomplete if pending work is observed; no complete
concurrent coverage guarantee is made. Budget exhaustion is not a passed test.

Defaults are 100,000 statement/call steps per test, at most 256 active calls, and a 10-second
worker timeout. Step budgets may be explicitly raised to 10,000,000; timeouts to 10 minutes.
Timeouts, pending tasks, missing tests and worker errors are visible. Counts show what was
executed, not whether assertions establish the intended behavior.

## Targeted mutation testing

Mutation requires one covered production symbol. Supported mutations flip comparisons,
change +/binary - and flip boolean literals. Selection is bounded and source-ordered within
the chosen function. Strings/comments and match arrows are not rewritten. Coverage guides
function selection, not exact per-operator reachability. Unsupported targets or zero tests
are not reported as successful mutation adequacy.

The unchanged baseline must pass completely. Each mutant is checked before execution:

- `invalid`: rejected by static contracts; never counted as a killed mutant.
- `killed`: valid mutant caused an observed test failure.
- `survived`: valid mutant passed these tests; it may be equivalent or expose weak assertions.
- timeout/budget/incomplete/worker failure: inconclusive, never counted as killed.

The default limit is 10 candidates (maximum 100), each worker has the explicit timeout/step
budget, and total mutation execution has a 60-second budget. `--total-timeout` can change
that up to 10 minutes; setup/validation/process teardown can add overhead. A depleted budget
is reported with unexecuted counts. Reports expose raw categories rather than a misleading
single success percentage. All baseline tests run for each valid mutant; selective reruns
are deferred until test selection is sufficiently trustworthy.

By default survivors are reported. A project can explicitly request a hook failure:

```bash
kiro analyze mutate . --symbol pricing.quote --max-mutants 4 \
  --timeout 10s --total-timeout 30s --fail-on-survivor --json
```

`KIRO_MUTATION_SURVIVOR` identifies that requested gate. Inspect equivalents and test intent
before adding assertions simply to improve a score. No global coverage/complexity/mutation
threshold or refactor action is added to ordinary compiler commands.
