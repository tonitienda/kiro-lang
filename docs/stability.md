# Experimental stability and constraints

Kiro is an experimental language with an intentionally small backend/service focus.

## Supported use cases (current)

- language frontend development (lexer/parser/formatter/project loading)
- generated-Go inspection workflows
- small service-oriented examples and scaffolding patterns

## Unsupported / not goals

- native backend
- browser/JS/WASM targets
- macro/metaprogramming systems
- package registry and heavy framework abstractions

## Expected churn

More likely to change:

- stdlib naming and helper granularity
- semantic/codegen completeness
- runtime behavior of commands still marked placeholder

Less likely to change:

- explicit project structure
- deterministic formatting
- inspectable generated-Go workflow
- opinionated small-service philosophy
