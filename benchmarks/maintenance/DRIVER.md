# External model driver protocol

The runner starts a fresh driver process for every trial, using argv after `--driver`.
The driver is trusted transport code, not another agent with host tools. It must invoke
only the configured remote/local model and return model-selected broker actions; it must
not grant the model its own shell, filesystem, network browsing, persistent memory, other
agents, source-repository access or acceptance fixtures. Provider keys belong in the
transport environment, never JSON settings/logs. No provider is selected by this repository.

There is no model CLI/API configured in the development environment, so the repository
includes transport tests but no completed model trial. Supplying an executable does not
prove it is a correct provider adapter. Pin/audit the adapter and record its version/hash
before making performance claims; this protocol deliberately separates it from the tools.

The driver reads one JSON line on stdin and writes one JSON line on stdout. Diagnostic
logs go to stderr and must exclude credentials. Return one reply per request. A request
contains no actual host paths. The workspace is identified abstractly as `/work`.

1. `start` includes schema `kiro.benchmark.driver.v1`, requested model/settings, task,
   bootstrap, tool descriptions, allowed Kiro commands and limits. Reply with identity
   metadata, without invoking the model:

```json
{"type":"identity","execution_kind":"model","model":"exact-requested-model-id","provider":"provider-name","adapter_version":"version-or-commit","resolved_model":"provider-returned-snapshot-if-available"}
```

The model ID must equal the requested ID. `execution_kind:scripted-control` is reserved for
transport/evaluator tests and is excluded from model summaries. The adapter must not call
a scripted solver a model. Identity is a declared transport claim, not external attestation.

2. `ready` starts inference with the startup/task information in context. Return an action:

```json
{"type":"action","action":{"tool":"read","path":".kiro/skill/SKILL.md"},"usage":{"input_tokens":100,"output_tokens":20}}
```

3. For each `observation` request, append the observation to this fresh model conversation
   and return another action or finish:

```json
{"type":"action","action":{"tool":"kiro","args":["check",".","--json"]},"usage":{"input_tokens":300,"output_tokens":30}}
```

```json
{"type":"finish","message":"Implemented and verified the change.","usage":{"input_tokens":400,"output_tokens":15}}
```

Only the model conversation for the current trial may persist. Do not reuse context from
another condition/task. No evaluator feedback is delivered during the implementation trial.
Usage fields are incremental per inference request, not cumulative session totals. Preserve
provider usage metadata (e.g. cached tokens) in additional usage fields when available;
unsupported or missing values remain unavailable. The runner sums input/output only when
all inference replies provide nonnegative integer counts; it retains raw usage entries.
Costs are unavailable unless measured separately with the provider/version/rate recorded.
Never estimate model tokens from file byte counts.

Tools: `list(path)`, `read(path,offset?,length?)`, `write(path,content)`, `delete(path)` and
`kiro(args,save_as?)`. Actions are JSON objects, not shell strings. No arbitrary exec exists.
Paths resolve beneath the task workspace. Pinned `.kiro/`, `AGENTS.md` and `task.json` are
read-only; editable files are .ki/.md/.json. Reads/writes are capped at 256 KiB, with ranged
reads for larger files and an 8 MiB workspace budget. Tool output is capped at 128 KiB;
full graph reports up to 8 MiB can be saved as JSON and read in ranges. Use focused queries
instead of flooding context. The core condition rejects analyze before invoking a process.

Default limits: 120 tool attempts, 900 seconds including context acquisition/container work,
120 seconds per driver response and 60 seconds per Kiro invocation. A command timeout
terminates the candidate container; terminating only a Docker exec client is insufficient.
Driver stdout is limited to 1 MiB per JSONL reply. Requests/observations and responses are
retained verbatim in transcript.jsonl; do not inject private reasoning requirements into
model output. The adapter should request only actions and concise completion text.

The adapter runs outside Docker so it can contact its model provider, but only the broker
executes candidate code, in a network-disabled container. Kiro cannot inspect the host
repository/oracle through that container. During evaluation, expected values and oracle
source remain on the host; a fixed HTTP transport inside the container sees only one request
and its actual response. Model process/container exit precedes evaluation. Docker resource
and mount settings need end-to-end verification on the intended runner; that verification
could not be performed in the development environment, which has no Docker runtime.
