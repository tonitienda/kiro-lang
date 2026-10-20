# Typed errors and explicit recovery

```bash
kiro check examples/typed_errors
kiro test examples/typed_errors
kiro run examples/typed_errors
```

The seven tests cover exhaustive payload matching, explicit error conversion, a partial
constructor passed through a typed function, variant JSON round trips, nil error tags,
list-loop exits/propagation and binding scopes. Main prints `missing: item`.

See [the language contract](../../docs/variants-and-matching.md). The quote service extends
this pattern across catalog, shipping, pricing and HTTP boundaries.
