# Phase 16: typed errors, exhaustive recovery and executable list loops

Agent implementation, auditing and deterministic verification continue without a required
human source-review gate.

- Closed variant declarations use `type Error { variant Missing { key:str } variant Offline {} }`.
  Keyed pure constructors are typed function values and support explicit partial application.
- Patterns bind payloads explicitly with `let`, ignore them with `_`, or compare literals.
  They never execute arbitrary calls/interpolation. Binding scope and immutability are checked.
- Known variant matches require explicit total arms for every tag, with no wildcard catch-all.
  Result matches cover Ok/Err (or a final compatibility wildcard). Missing/unreachable arms
  and incompatible payload/branch types are diagnosed. Nested matches handle payload decisions.
- Error conversion is ordinary typed code using exhaustive matches. `?` does not invent
  conversions, error unions or automatic wrapping.
- Variants preserve module identity across contracts, captures and nested JSON schemas.
  JSON uses exactly one tag and an object payload, with existing strict schema validation.
- Quote service explicitly converts catalog/shipping errors to pricing errors and maps
  invalid requests to 400 and all domain errors to 422. The suite now has twenty tests.
- Added a seven-test typed-errors example, including typed callbacks, partial constructors,
  JSON round trips, explicit recovery, lexical scopes and list loops.
- Implemented finite list `for` loops with immutable iteration bindings, scoped locals,
  outer mutable updates, nested loops, return and result propagation.
- Checking now rejects every body that cannot parse in the executable grammar. Old map
  literals and unsupported control constructs no longer silently pass. Unknown identifiers
  also fail checking; expression inference still defers some unknown types to runtime.
- Added source/runtime/native tests, compatibility fixtures and an isolated mutation probe
  showing new catalog/shipping/pricing variants invalidate the respective explicit handler.
- Guide snapshot: `phase16-typed-errors`. See `docs/variants-and-matching.md` for contracts,
  breaking changes, evidence and limits.

Coverage is finite constructor coverage, not arbitrary nested-domain constraint solving or
complete return-path/type inference. Tagged optionals/unit, native typed Go lowering,
structured diagnostics/queries and complete concurrency/effect enforcement remain future
work. No comparative model or performance claim is made.
