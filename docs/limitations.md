# Limitations

Kiro is still experimental.

## Current limitations

- semantics remain intentionally small and incomplete compared with mature languages
- effect checking is explicit and conservative rather than inferred
- diagnostics are improving, but not every runtime/type mistake is caught statically yet
- named-call checking resolves direct signatures and some receiver types; aliases, higher-order calls and calls embedded in string interpolation need runtime signature validation
- generated Go is inspectable but not a stable public API
- example breadth is larger than fully battle-tested runtime coverage
- executable builds currently package a Go-hosted interpreter; Go inspection emits structural stubs rather than faithful direct lowering
- checking rejects bodies outside the executable grammar; not every accepted expression has complete static inference
- runtime contracts enforce returns and closed schemas; static checks cover known types and conservative return paths, not a complete type proof
- result propagation is function-local and caller-inspectable; unrelated error channels need explicit conversion
- `group` does not yet enforce child-task joining/cancellation, and context helpers do not implement real deadlines

See [the implementation audit and executable probes](research/agent_language_iteration.md) for evidence and proposed priorities.

## Intentional non-goals for now

- backwards compatibility with every historical syntax variant
- feature growth through aliases or overlapping constructs
- implicit async/effect systems

Agent discovery and JSON feedback are available in Phase 17; see [the tool contract](agent-tools.md).
Checking is first-error; some legacy diagnostics retain broad fallback codes and declaration
locations. Symbol queries show declared contracts rather than resolved dynamic callers.
