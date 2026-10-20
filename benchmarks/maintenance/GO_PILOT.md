# Matched Go and Kiro maintenance pilots

The primary comparison is **each agent across languages**: Claude/Kiro against
Claude/Go, and Codex/Kiro against Codex/Go. This is a language + toolchain + documentation
comparison. It cannot isolate syntax, prior training familiarity, or universal language
quality. Optional Kiro analysis is a separate ablation, not a language condition.

Go fixture `go-maintenance-v1` and `tasks-go-v1/*.json` are new versioned inputs.
The existing Kiro workload, tasks and v1 prepared cohort remain unchanged. The new native
Kiro preparation is `kiro-scaffold-pilot-v2`, because scaffold-generated guidance differs
from the original broker workspace. Never relabel old cohorts or silently refresh hashes.

## Prepare independent workspaces

Build the pinned Kiro compiler once, outside measured implementation time. From the repo:

```sh
go build -o /tmp/kiro ./cmd/kiro
python3 -B benchmarks/maintenance/runner/prepare_pilot.py \
  --language kiro --kiro /tmp/kiro --task payment-boundary --out /tmp/kiro-pilot-v2
python3 -B benchmarks/maintenance/runner/prepare_pilot.py \
  --language go --task payment-boundary --out /tmp/go-pilot-v1
```

Each destination must be absent. Repeat preparation for every task, agent and repetition;
changes never carry from one task to the next. The Kiro command actually invokes
`kiro new service`, retains generated `AGENTS.md` and `.kiro/` byte for byte, removes sample
application sources, copies the workload and adds `task.json`. Go gets a separate module,
Go-specific `AGENTS.md`, six equivalent public tests and `task.json`, without Kiro docs.
The sibling `PROMPT.md` is the complete prompt to supply to a fresh session; it contains the
same maintenance requirement and measurement/isolation instructions in both languages.
`preparation.json` records source commit/dirty state, preparation time, toolchain identity,
prompt hash and every workspace hash. Retain the actual toolchain artifact and full initial
workspace, not just a release label. No model is run by preparation.

Use identical exact model/settings, reasoning effort, budget, resources and tool access
within each agent's language pair. Counterbalance order and predeclare repetitions. Start
measurement before reading the prompt/guidance; include documentation/context acquisition,
failed attempts, implementation, agent review and verification. Keep preparation and final
independent evaluation time separate. Record only observed telemetry; unavailable tokens,
cache, cost or timing stay unavailable. Retain unsuccessful and timed-out runs.

## Provision isolation before starting a model

A generated workspace and an instruction to stay within it do not establish isolation.
For actual candidates, use the documented network-disabled Docker workflow in
[README.md](README.md) and [DRIVER.md](DRIVER.md): mount only that fresh service workspace,
keep `AGENTS.md`, `task.json` and `.kiro` read-only, expose no host repository, evaluator,
scripted controls, other trials, credentials or network, and retain the runtime probe.
Use a pinned image ID, resource limits and fresh private caches for each run.

The existing JSONL model broker is Kiro-specific. Do not route Go actions through its
Kiro command allowlist, or claim an automated Go model run is implemented by preparation.
Native Go agents need an equivalent externally provisioned isolated tool environment with
standard Go commands; record its configuration and transcript. Native-agent runs where only
instructions constrained filesystem/tools are **instruction-constrained pilots**, distinct
from broker-isolated trials. They must not be reported as held-out isolated evaluations.
There is no host-execution fallback for actual model candidates.

Docker CLI exists on the development host, but its daemon is unavailable. No configured
external model driver was supplied. Therefore no new model trials or candidate evaluations
were run here. The Go isolated evaluation path is described by `go_benchmark.py --help`;
its container orchestration must be verified on a Docker host before interpreting results.
Use `--image sha256:<64-hex-image-id> --project /path/to/candidate --task payment-boundary`
for isolated candidate evaluation; `--trusted-fixture` is reserved for setup code.

## Deterministic validation

Trusted repository fixtures and scripted fixes may run locally during harness development:

```sh
python3 -B -m unittest discover -s benchmarks/maintenance/runner -v
python3 -B benchmarks/maintenance/runner/benchmark.py selftest \
  --kiro /tmp/kiro --out /tmp/kiro-controls.json
python3 -B benchmarks/maintenance/runner/go_benchmark.py --selftest \
  --trusted-fixture --out /tmp/go-controls.json
python3 -B benchmarks/maintenance/runner/test_go_parity.py \
  --kiro /tmp/kiro --trusted-fixture --out /tmp/parity.json
(cd benchmarks/maintenance/workload-go-v1 && go test ./... && go vet ./...)
```

These are setup controls, never model results. Both baseline services consume the unchanged
353-case language-neutral HTTP matrix. Additional independent parity checks cover malformed
and missing input, schema/type boundaries and safe integer arithmetic. Reference and
incorrect patches for every independent task test acceptance/rejection; unchanged payment
extraction preserves HTTP behavior but must fail structural acceptance.

The Go evaluator reports HTTP behavior, public API and architecture separately. Nine compile
probes map the Kiro contracts to Go's exported functions/fields and error-return conventions.
`app.Handler` uses the standard `http.HandlerFunc` signature; `QuoteFor` and `StatusAt` avoid
colliding with Go type names. `checkout.Error` may alias `error`. The payment task permits
Go aliases or wrappers for `Authorization` and ordinary error propagation. Structural checks
use parsed Go declarations/references/imports, rather than forcing Kiro's nominal conversion
syntax. Payment policy must be owned and used by `internal/payments`, with no dependency
back toward business/HTTP packages. This is an explicit architecture contract, not a general
proof of modularity; syntactic call edges cannot establish arbitrary runtime reachability
or prove that every error reference is returned. The HTTP/API checks supply complementary
evidence.

## Parity limits and existing observations

Successful JSON shapes and business-error status/text match the shared oracle exactly.
Malformed JSON returns 400 in both languages; diagnostic wording is deliberately unspecified
by the existing cases and differs between the implementations. Required fields are
case-sensitive; duplicate/unknown fields, null required values, invalid UTF-8, trailing JSON
and non-integer/out-of-int32 values are rejected before business logic. Coupon may be absent
or null. JSON whitespace, response object field order and headers are not scored.

Inputs are signed 32-bit integers. The original workload explicitly limits arithmetic to
non-overflowing i32 operations. Go wraps overflowing int32 arithmetic whereas Kiro rejects
it at runtime; refund overflow outside that predeclared domain is not parity-tested or
claimed equivalent. Reservation min/max values and refund arithmetic close to the maximum
safe value are included in independent validation. Extending the numeric domain requires a
new fixture/cohort and predeclared expectations, rather than weakening current checks.

The supplied Codex report (95.83 seconds, 11 project tests) and Claude report (51 seconds,
9 project tests) have not had candidate source/diffs independently evaluated. The earlier
72-second Claude report is a separate run. They are instruction-constrained observations,
not correctness results, evidence of language advantage, or a model-speed ranking. Do not
combine these observations with scripted controls or preparation timings.
