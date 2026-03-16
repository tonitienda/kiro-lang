# Stdlib API style guide (Phase 9)

This document defines naming and shape conventions for Kiro stdlib modules.

## Goals

- Keep common operations predictable across modules.
- Prefer small, explicit APIs over many aliases.
- Preserve backwards compatibility unless a cleanup materially improves clarity.

## Module surface conventions

Applies to: `env`, `parse`, `http`, `json`, `log`, `test`, `fs`, `ctx`, and task/concurrency helpers.

### Naming

- Use short lower_snake names (`get`, `get_or`, `require`, `must`) instead of near-duplicates.
- Prefer verb-first function names for effects (`read_text`, `write_text`, `with_header`).
- Keep booleans explicit (`ok_json`, `not_found`, `bad_request` are action/result names, not ambiguous toggles).

### Argument order

- Prefer **primary input first**, then options/config, then context-like values.
- For helpers that mutate/build a value, put the base value first:
  - `with_header(resp, key, value)`
  - `with_query(req, key, value)`

### Error/result shape

- Use `R[T,E]` for fallible operations where callers may recover.
- Reserve `must_*` for crash-on-error convenience wrappers.
- Keep paired APIs aligned:
  - `get` -> optional/maybe style
  - `require` -> `R[T,E]` with descriptive error
  - `must` -> panic/abort behavior

### Output helpers

- `print` means no newline.
- `println` means trailing newline.
- `json` and `text` response constructors should have consistent defaults per module docs.

## Compatibility + deprecation policy

Phase 9 uses a **documented deprecation policy** (without compiler warnings yet):

1. Keep deprecated names as aliases for at least one phase.
2. Mark deprecations in docs and phase notes.
3. Update examples/templates to preferred names immediately.
4. Remove aliases only in a later phase with explicit migration notes.

If warnings are added later, they should start with stdlib aliases only and remain opt-out-free (always visible) until a warning configuration story exists.

## Review checklist for stdlib changes

When changing a stdlib module:

- Is the name aligned with existing module vocabulary?
- Is argument ordering consistent with similar helpers?
- Is return shape (`T`, `?T`, `R[T,E]`) justified and documented?
- Are docs/examples/tests updated together?
- If a rename occurred, is alias + migration note present?
