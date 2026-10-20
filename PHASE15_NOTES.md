# Phase 15: compositional results and closed schemas

The agent-only implementation/audit experiment continues without a human source-review gate.

- Results have explicit success/error discriminators, including nil payloads. `?` returns
  an Err from the current user function, preserving its payload and validating its error
  channel. It requires a result-returning enclosing function; affected examples, service
  scaffolds and tests now return `Ok(...)` explicitly.
- Process/test/HTTP adapters handle result failures explicitly. An unhandled Err cannot
  silently pass a test or produce exit code zero. HTTP Err remains status 500; status
  mapping is application work. Typed/partial HTTP handler values are supported.
- Source record construction checks all keys before effects, then field types. Runtime
  boundaries check nested schemas and retain native result/callable values inside fields.
  Records carry declaration modules; callable/schema references are qualified across modules.
- Typed JSON rejects missing required/unknown/duplicate fields, wrong nested/list types,
  trailing input and i32 range/representation violations. Omitted optional JSON fields
  become nil. Integer text is parsed without float64 rounding. Input must be UTF-8, with
  a nesting limit of 128. `Any` remains an explicit untyped escape hatch.
- JSON record output excludes internal metadata. Result JSON has an explicit Ok/Err key,
  including null payloads. Decoder partials retain their original schema owner.
- Escaped string literals work consistently through lexing, formatting, constants and
  runtime evaluation. Interpolated data is appended literally, not evaluated as more source.
- Inline `?` inside spawn expressions is rejected. Spawn a result-returning function,
  await its result and propagate explicitly in the awaiting function.
- Added a three-module validated aggregation example with five tests, and expanded the
  quote service to sixteen tests including parent/handler failure observability.
- Guide snapshot: `phase15-results-schemas`. See `docs/results-and-schemas.md` for contracts,
  breaking migrations, error policy and test coverage.

Validation combines source rejection cases, unchecked runtime cases, compiled integration,
HTTP adapter tests, schema round trips, fuzzing, compatibility fixtures and existing suites.
Original gap probes are preserved; new probes distinguish JSON schema validation from
rejection of the old non-result entry signature. Loop execution remains an observed gap.

No general tagged-variant syntax, payload-binding/exhaustiveness analysis, automatic error
conversion, native typed Go lowering or complete concurrency guarantees are claimed. The
Phase 14 allocation measurements are historical and do not characterize the new checks.
