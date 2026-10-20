# Validated, compositional aggregation

Three modules combine strict typed JSON, optional fields, a reusable mapping transducer,
a result-valued reducer, typed higher-order functions and a captured decoder schema.

```sh
go run ./cmd/kiro fmt examples/validated_results
go run ./cmd/kiro check examples/validated_results
go run ./cmd/kiro run examples/validated_results
go run ./cmd/kiro test examples/validated_results
```

The program prints `3`. Five tests verify successful aggregation, a number-parsing error
reaching the caller, a schema error reaching the caller, partial/typed callback boundaries,
and distinct nil result tags surviving JSON. The optional request note may be omitted from
JSON and becomes nil; source constructors still require explicit optional fields.

Errors remain values across the pipeline. Result-valued reduction does not automatically
stop iteration; each reducer chooses how to handle its state. Runtime errors such as
arithmetic overflow remain separate from domain failures. See
[the contracts and limits](../../docs/results-and-schemas.md).
