# Limitations

Kiro is still experimental.

## Current limitations

- semantics remain intentionally small and incomplete compared with mature languages
- effect checking is explicit and conservative rather than inferred
- diagnostics are improving, but not every runtime/type mistake is caught statically yet
- generated Go is inspectable but not a stable public API
- example breadth is larger than fully battle-tested runtime coverage
- executable builds currently package a Go-hosted interpreter; Go inspection emits structural stubs rather than faithful direct lowering
- accepted frontend syntax is broader than executable body syntax
- declared return types and JSON field schemas are not fully enforced
- failed result propagation currently aborts interpretation rather than returning a caller-inspectable `Err`
- `group` does not yet enforce child-task joining/cancellation, and context helpers do not implement real deadlines

See [the implementation audit and executable probes](research/agent_language_iteration.md) for evidence and proposed priorities.

## Intentional non-goals for now

- backwards compatibility with every historical syntax variant
- feature growth through aliases or overlapping constructs
- implicit async/effect systems
