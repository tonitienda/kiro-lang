# Explicit transducers: semantics and allocation observations

Date: 2026-10-19. This is a runtime microbenchmark, not an agent or language-comparison
benchmark. The change introduces explicit typed stateless transducers; it does not silently
optimize existing eager calls.

## What is compared

Both paths filter a finite list to nonnegative integers, increment retained values through
a pure partial function, and accumulate a sum. Eager execution materializes the selected
list and the mapped list. Fused execution reuses a filter-then-map `Transform` plan and
reduces each retained item immediately.

A separate test checks eager and fused outputs against an independent Go calculation over
80 deterministic inputs of varying lengths, including empty inputs. Native tests also
establish that eager and fused execution can intentionally expose different first errors:
a later predicate overflow wins in the eager path, while an earlier mapper division by zero
wins in the fused path. Both programs pass source checking; purity does not imply totality.

## Reproduce

```sh
go test ./internal/buildsys/runtimekit -run '^$' \
  -bench BenchmarkCollectionPipelines -benchmem -benchtime=200ms -count=3
```

The benchmark code is in `internal/buildsys/runtimekit/transducers_test.go`. It excludes
input/plan construction from timed sections and includes callback interpretation, partial
binding, argument/return checks and accumulation. Both paths use the same callbacks and
source data in the same runtime; this is not a comparison between compiler commits. The plan is reused; this is not a claim about one-shot construction cost.

Observed with Go 1.23.12 on linux/amd64, AMD EPYC 9V74, and Go's reported benchmark
concurrency suffix `-9`.
Numbers below are the median of three runs in this environment; timings are noisy and
are not suitable as release performance thresholds.

| Input items | Eager ms/op | Fused ms/op | Eager B/op | Fused B/op | Eager allocs/op | Fused allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 0.314 | 0.334 | 149181 | 146029 | 1747 | 1733 |
| 1000 | 3.250 | 3.253 | 1489128 | 1462031 | 17745 | 17728 |
| 10000 | 31.498 | 33.549 | 14960572 | 14638536 | 179757 | 179735 |

## Interpretation and practical limits

The fused loop creates no intermediate data lists. Plan storage is proportional to the
number of stages; it retains no source list. `collect` allocates a final output list,
whereas `transduce` stores a current element and accumulator. These bounds exclude input,
accumulator contents, callback-produced values and interpreter machinery. Typed function
boundaries can still validate lists separately from processing them.

Total allocation volume in this benchmark falls by roughly 2%, rather than approaching
zero. Per-element interpreted function frames, partial binding and value checks dominate.
The fused path is slower in these measurements, so this batch establishes an explicit
execution model and removes intermediate collection storage; it does not establish a
runtime speed improvement. Native typed lowering or cheaper callable invocation should be
measured separately before making a speed claim.

The difference in error order is part of the new API contract. Replacing eager code with
transducers is a visible semantic decision, even when both paths produce the same successful
result. An agent-focused experiment should measure repair/correctness consequences as well
as bytes, time, diagnostics and model/tool usage.

Only mapping/filtering stages over finite lists are implemented. Early termination,
stateful stages/completion, lazy sequences, parallel reduction and stream adapters remain
separate work. Existing JSON-schema, result-propagation and broader control-flow/concurrency
limitations are unchanged.
