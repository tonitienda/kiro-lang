# Agent/compiler feedback protocol

Phase 17 adds a small discover → inspect → edit → validate loop for agents. All commands
below work locally; no human source-review gate or network service is required.

```bash
kiro describe --json
kiro describe json.decode --json
kiro symbols . --json
kiro symbols . --symbol pricing.quote --json
kiro check . --json
kiro fmt . --check --json
kiro fmt .
kiro lint . --json
kiro test .
```

## Machine envelopes and diagnostics

`check`, `lint`, `describe`, `symbols` and `fmt --check` accept `--json`. They emit exactly
one JSON document on stdout with schema `kiro.tools.v1`, `command`, `ok`, `diagnostics`
(an array, including on success) and optional `data`. Validation/query failure exits 1;
a successful command exits 0. Reported failures do not also print prose to stderr. Unknown
flags and missing arguments also use the envelope when `--json` is present. Build/run/test
retain their existing process output; they do not acquire a JSON mode in this batch.

Diagnostics have stable `code`, `severity` and `message`. Native checks supply relevant
`expected`, `actual`, `missing`, `symbol`, `target` and `hint` fields instead of requiring
agents to classify prose. For example:

```json
{
  "code": "KIRO_NON_EXHAUSTIVE",
  "severity": "error",
  "message": "...",
  "symbol": "pricing.from_catalog",
  "expected": "catalog.Error",
  "missing": ["OutOfStock"],
  "location_kind": "declaration"
}
```

Use `kiro describe --json` for the current code catalog. Native structured families include
syntax expectations, missing returns, known type mismatches, missing variant arms, keyed
argument errors, unresolved identifiers and missing effects. Remaining legacy checks use
`KIRO_CHECK_FAILED`, not a guessed category. Filesystem/import failures use `KIRO_PROJECT`.
Fields that cannot be established are omitted. Codes are stable within this protocol;
message wording remains explanatory rather than a parsing interface.

`file` is an absolute source path. Ranges use one-based Unicode code-point columns and
exclusive ends. `location_kind:token` identifies parser token spans; `declaration` identifies
the enclosing function name because reconstructed bodies do not yet retain full source
maps. Legacy prose-derived positions are explicitly marked `legacy`. Some errors have no
range. A declaration span must not be treated as the exact faulty expression or an automatic
edit target. Human-readable output retains source excerpts where available.

Compiler checking remains fail-fast: `check` reports the first error. `lint` checks the
same contracts first, then all selected files' canonical formatting. It is currently that
composition, not a separate speculative style analyzer. Recheck after a repair. This
protocol does not imply a complete static type system, a security sandbox or automatic fixes.

## Discovery and source snapshots

`describe` reads language guidance from the bundled manifest and builtin types/keys/effects
from the actual runtime registry. `describe json.decode` narrows the output to one builtin.
These are **base signatures**: schema/callback/type-argument specialization still applies;
`Any` is not a claim that every value is valid for every collection or generic operation.
The report includes the compiler version and guide version.

`symbols` reads declarations even when semantic checking fails. Its `validation` is
`syntax_only`. Results are sorted and include module-qualified IDs, declaration ranges,
function parameters/defaults/effects/returns, record fields, variant constructors and docs.
Methods use receiver-qualified IDs. Declared types are shown as authored; this is not a
resolved call graph or a query for local inferred values. A filter uses an exact ID, and
an unknown ID is an error. Duplicate declarations remain visible rather than silently
choosing one in discovery output.

Both symbol queries and successful checking include `revision`, a SHA-256 hash of sorted
relative `.ki` paths and exact source contents. It is independent of the absolute project
folder. Whitespace edits change it; compiler versions, external dependencies and hidden
files are not included. It identifies a source snapshot, not a lock or edit transaction.
Structural edits with revision preconditions remain future work.

## Formatter checks

`fmt --check` never writes. Changed files produce `KIRO_FORMAT_REQUIRED` diagnostics and
exit 1. Regular `fmt` prepares every selected file before writing, so a syntax/read failure
does not leave earlier files reformatted. Existing file permissions are preserved. I/O
failure during the write phase is still possible; the filesystem writes are not a transaction.
Paths are deduplicated and sorted. Directory traversal skips hidden directories, consistent
with project loading; explicitly selected files remain available.

## Return-path analysis

Functions whose result cannot accept nil must return on every successful path. An exhaustive
match with returning arms and a returning group establish an exit; a primitive match needs
a catch-all/binding or both boolean literals. Return inside a match expression is recognized.
An expression value alone is not a return. A loop may be empty, even when its source is a
literal list; `?` may succeed. Neither alone proves a return. Optional, nil and Any returns
retain the runtime's admitted nil fallthrough. The analysis is conservative, structural and
intraprocedural; it does not prove termination, infer callees' non-returning behavior or
solve arbitrary payload conditions.

## LSP alignment

`kiro lsp` now publishes the same semantic codes and structured data as command-line
checking, including types, missing returns and missing variant arms. Unsaved buffers are
overlays: checking never writes them to disk. With a workspace root, checking chooses the
outermost ancestor containing `main.ki` inside that workspace. Without a workspace root,
an existing document uses its containing project directory; virtual documents receive
standalone checking. Open the Kiro project root for reliable cross-module diagnostics.

Project feedback is still first-error. Diagnostics are routed to the source file and cleared
on repair/close. LSP ranges use zero-based UTF-16 coordinates, converted from compiler
positions. Completion is deterministic, includes builtin contracts and variant constructors,
and excludes unsupported control-flow keywords. Hover/definition remain document-local;
this batch does not add workspace references, rename or a complete language-server API.

## Maintenance experiment setup

The runner provides three repeatable Kiro tasks: a new stock error, a delivery request schema
change, and a coupon minimum-subtotal rule. It prepares isolated copies of the quote service,
a task specification and the bundled language guide:

```bash
python3 scripts/agent_maintenance.py prepare --task stock-error --out /tmp/stock-task
# Point the agent at /tmp/stock-task/task.json and /tmp/stock-task/project.
python3 scripts/agent_maintenance.py evaluate --task stock-error \
  --candidate /tmp/stock-task/project --kiro ./kiro
```

Task IDs are `stock-error`, `delivery-schema` and `coupon-minimum`. Evaluation copies the
candidate, adds fixed acceptance tests from the runner, and checks lint, semantic contracts,
existing project tests and the additional behavior. Original candidate files are untouched.
The report schema is `kiro.maintenance.v1`; it records verification invocations, outputs,
statuses and verification time. An unchanged baseline fails all three tasks. A scripted
stock-error reference repair passes the full evaluator.

This is experiment infrastructure, not an agent result or adversarial execution sandbox.
Acceptance cases live outside the prepared candidate; they are not claimed to have been
created by a statistically independent evaluator. The calling experiment must separately
record agent version, prompts, tokens, agent tool calls, repair attempts and elapsed time,
and retain failed runs. Go/guided-Go treatments and comparative model runs remain to be
prepared before making performance claims. The existing variant evolution probe now checks
codes/symbols/missing tags through JSON rather than parsing diagnostic prose.

## Optional maintenance analysis

Phase 18 adds [explicit analysis commands](analysis-tools.md) for dependency/impact queries,
metrics, cycles, candidate boundaries, user-selected architecture rules, context slices,
Kiro coverage and bounded targeted mutation. They never run as a side effect of check,
lint, formatting or LSP feedback. Agents choose relevant tools; users choose hooks.
