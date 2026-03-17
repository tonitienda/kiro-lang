# Current limitations

Kiro is still experimental. These limits are intentional to set expectations.

## Backend and portability

- Compilation strategy is Go backend first.
- Linux is primary target; macOS is expected to work, but coverage is lighter.
- No browser/JS/WASM target.

## Type system and semantics

- Advanced generic/union edge cases may evolve.
- Result/optional flow is stable in common patterns but not yet proven across large codebases.

## JSON and HTTP surface

- JSON decode/validation ergonomics are still evolving.
- HTTP helpers are intentionally small and may refine naming/details.

## Concurrency model

- `group`/`spawn`/`await` supports tiny structured concurrency patterns.
- Broader scheduling/cancellation patterns are still maturing.

## Diagnostics

- Diagnostic quality is improving, but exact wording/location can still change.

## Generated Go

- Generated Go is for debugging and trust-building, not a stable public API.
- File layout and helper internals may change between experimental releases.

## Not yet ideal for

- large multi-team codebases requiring strict long-term compatibility guarantees
- projects needing multiple backend targets
- heavy framework/macro-based metaprogramming styles
