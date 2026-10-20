# Kiro maintenance benchmark

Status: workload, tasks, deterministic evaluator and external-driver harness. No model
performance results yet. The experiment has no human source-review gate. It is separate
from ordinary compiler workflows and never runs through check/lint/build hooks.

The evaluator currently contains 353 HTTP cases per task; these are concrete checks, not
a completeness proof.

The paired experiment asks whether optional analysis helps a fresh agent complete a Kiro
maintenance change correctly at a useful total cost. It does not measure a universal language
ranking. A later Go port can reuse the HTTP requirements/case matrix; Go fixture/tooling and
structural-check mapping are not implemented in this cohort.

| Task | Requested change | Evaluation |
| --- | --- | --- |
| discount-policy | Add SAVE10 with a 5000-cent threshold | Receipt amounts, fee/reference and regressions |
| reservation-expiry | Expire at the exact deadline | Before/equal/after, including negative timestamps |
| partial-refund | Refund shipping only for full returns | Zero/partial/full returns, validation and regressions |
| payment-boundary | Extract payment policy with compatibility adapters | Unchanged HTTP behavior plus explicit dependency/ownership contract |
| payment-declined | Add a distinct typed payment failure | HTTP 402, error priority, typed constructor usage and regressions |

Every task starts from the same independent baseline, not the previous task's solution.
The [workload](workload/README.md) is an in-memory HTTP service with ten modules and six
public tests, not a persistent order/payment system. Some behaviors are intentionally buggy
maintenance targets. Do not “fix everything” during a trial. All amounts are synthetic.

The evaluator computes expectations independently in Python and checks the built service
through HTTP, rather than trusting candidate assertions. It also runs canonical checks,
public tests and a pinned nine-symbol public API contract (key labels, types and fields). Expected values/oracle files never enter the candidate container. Structural
scores are reported separately from behavioral scores. This oracle and fixture were authored
by the same setup agent; a separate agent audit is still needed to reduce shared assumptions.
Negative controls test the checks but do not prove their correctness or independence.

## Validate the harness (not a model benchmark)

```sh
python3 -B -m unittest discover -s benchmarks/maintenance/runner -v
python3 -B benchmarks/maintenance/runner/benchmark.py selftest \
  --kiro /absolute/path/to/kiro --out /tmp/kiro-controls.json
```

Selftest executes trusted local fixtures: unchanged baseline, five reference patches and
five deliberately incorrect patches, plus baseline-contract validation (16 evaluations).
An unchanged task copy must fail its changed requirement; extraction can pass all behavioral
cases while failing structure. Reference patches must pass and incorrect ones fail. These
are scripted harness controls, never agent trials. They run locally only because their
source is trusted setup code. Model candidates use Docker, with no host-execution fallback.

## Freeze a cohort

```sh
python3 -B benchmarks/maintenance/runner/benchmark.py prepare \
  --kiro /absolute/path/to/kiro --out benchmarks/maintenance/runs/cohort-001 \
  --repetitions 3 --seed 20261020
```

The plan contains 30 scheduled trials: five tasks × two conditions × three repetitions.
Conditions are core tools and core + optional analysis. Seeded ordering keeps each pair
adjacent and randomizes pair/condition order. The compiler binary, reference, guide bundle,
fixtures, tasks and harness are hashed. Any input change requires a new cohort. Git commit
and dirty-state metadata supplement those exact snapshot hashes. Preparation runs no model.
The actual model/settings/driver/image configuration is fixed before the first trial.

## Provision an isolated runner and model transport

This environment cannot run model trials: no Docker runtime or model adapter/API is
configured, and attempted bubblewrap namespace isolation is denied. There is no weaker
fallback labelled as held-out evaluation. Use a host with Docker and a separately configured
model driver conforming to [DRIVER.md](DRIVER.md).

Generate an image context containing only the compiler and its Go toolchain:

```sh
python3 -B benchmarks/maintenance/runner/benchmark.py image-context \
  --kiro /absolute/path/to/kiro --go-root /absolute/path/to/go \
  --out /tmp/kiro-benchmark-image
docker build -t kiro-benchmark /tmp/kiro-benchmark-image
python3 -B benchmarks/maintenance/runner/benchmark.py doctor \
  --kiro /absolute/path/to/kiro --image kiro-benchmark \
  --driver /absolute/path/to/model-driver
```

Build/provision outside measured trials. The base image must be reviewed/pinned by digest
for the actual cohort; the generated Dockerfile uses a convenience tag, not a frozen base
artifact. Doctor resolves the built image ID, compares its compiler hash to the selected
binary, probes host-file exclusion/read-only docs/network isolation and rejects missing prerequisites. Trial containers use that immutable local image
ID, no network, read-only root/docs, a writable task directory, resource limits and private
tmp/cache mounts. The host repository, other trials and provider credentials are absent.
Docker end-to-end execution is unverified in this development environment; first run and
retain the isolation checks on the intended host before interpreting model results.

```sh
python3 -B benchmarks/maintenance/runner/benchmark.py run \
  --plan benchmarks/maintenance/runs/cohort-001/plan.json \
  --kiro /absolute/path/to/kiro --image kiro-benchmark \
  --model exact-model-id --settings /path/to/nonsecret-model-settings.json \
  --driver /absolute/path/to/model-driver
```

Each trial gets a fresh driver process, container, workspace, docs and cache. Task/guide
instructions are supplied identically apart from the allowed-tool condition. The model
has only mediated workspace file actions and Kiro commands; core cannot invoke analyze.
The driver is trusted model transport and must not supply additional tools or memory.
The implementing model must not already have read the public oracle/reference patches;
repository publication alone does not make checks “held out” from every possible model.
A candidate's own assertions are not the sole correctness measure. Model/evaluation
failures are retained, and existing trial directories are never silently overwritten or
resumed. Use a new cohort for retries/configuration changes and disclose the earlier one.

## Evidence and reporting

Each trial retains the initial hashes, exact JSONL transcript, driver stderr, completion
status, public verification, every HTTP case outcome, structural checks, changed files,
final hashes and evaluation output. Summary files exclude scripted-control/unidentified
sessions from model aggregates; failures after valid model identity remain in the denominator.
A model pass additionally needs completion, pinned-file integrity and evaluation success.

Time includes onboarding, reads, tool execution, driver and container startup/teardown.
Evaluation is separate verification work. No cached-doc discount is silently applied.
File-read bytes (including repeats), documentation-read bytes for `.kiro` file reads, model-visible tool-output
bytes and tool attempts are distinct measures; do not add overlapping byte measures.
Reference text retrieved through describe is recorded in tool output, not the file-read counter.
`verification_recovery_cycles` counts a failed check/lint/test episode followed by a success
of the same command; it is an operational proxy, not a count of distinct bugs. Transcripts
retain command arguments and output, including denied/truncated calls. Token totals are
provider-adapter telemetry when complete, otherwise null. Unavailable cost stays null.

For the video, show the supplied reference, one full task trace, failure/repair and final
acceptance, alongside all repetitions and unsuccessful runs. Compare correctness first,
then total effort/context/tool costs. No conclusions about Kiro versus Go, agent improvement
or analysis value follow from the current deterministic controls. Follow the broader
[onboarding/reporting protocol](../../docs/research/benchmark_onboarding.md).
