# Contributing and intentional change workflow

Kiro prioritizes compatibility discipline and readable implementations.

## Required checks before submitting

```bash
go test ./...
go run ./cmd/kiro compat
```

## Phase 10 expectations

When a change is user-visible or architectural, update docs/notes in the same change:

- `README.md`
- relevant `docs/*` pages
- current phase notes (`PHASE10_NOTES.md`)
- `AGENTS.md` if workflow guidance changes

## Making intentional language/CLI changes

### Syntax or parser behavior

- add/update parser/formatter tests
- add/update compatibility fixtures
- document behavior in language docs (`docs/language_tour.md`, `docs/stable_core.md`)

### Stdlib API changes

- keep naming consistent with `docs/stdlib_style.md`
- document migration notes when behavior/spellings shift
- update examples/templates and compatibility fixtures

### Generated-Go changes

- preserve readability and origin tracing
- update inspect-go docs (`docs/debugging_generated_go.md`)
- adjust codegen fixtures intentionally

### Deprecations

- prefer conservative de-emphasis before removal
- document rationale and migration path in phase notes
- keep templates/examples on canonical APIs immediately

## Updating compatibility fixtures

1. choose appropriate category (`stable-core`, `regression`, `experimental`, `templates`)
2. create/update fixture directory with `main.ki`
3. add/adjust `fixture.json` if needed
4. run `kiro compat` locally

## Stable-core decision check

Before adding surface area, verify:

1. does it make README/templates/examples materially clearer?
2. can it be tested cheaply and deterministically?
3. can we maintain compatibility expectations for it?

If not, prefer docs/polish/cleanup over new surface.
