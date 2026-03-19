# Stable experimental core

The stable-core promise in this repository slice is intentionally narrow.

## Included in the current stable core

- deterministic lexer/parser/formatter behavior for the documented syntax slice
- project/module resolution rules
- canonical CLI workflow: `fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`
- explicit effect annotations in function signatures
- generated-Go visibility as part of the developer workflow

## Stable-core expectations

- command names and the broad workflow are intended to stay recognizable
- formatting and project layout rules should remain predictable
- generated Go remains a debugging/trust-building tool, not a frozen public API

## Still experimental around the core

- the exact standalone execution workdir used by `kiro build/run/test`
- release bundle structure details beyond the current documented `bin/kiro` + `toolchain/go` shape
- breadth of execution coverage across all experimental examples
