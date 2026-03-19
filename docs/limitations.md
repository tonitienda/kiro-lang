# Limitations

Kiro is still experimental.

## Current limitations

- semantics remain intentionally small and incomplete compared with mature languages
- effect checking is explicit and conservative rather than inferred
- diagnostics are improving, but not every runtime/type mistake is caught statically yet
- generated Go is inspectable but not a stable public API
- example breadth is larger than fully battle-tested runtime coverage

## Intentional non-goals for now

- backwards compatibility with every historical syntax variant
- feature growth through aliases or overlapping constructs
- implicit async/effect systems
