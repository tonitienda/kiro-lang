# Contributing and intentional change workflow

Kiro prioritizes compatibility discipline, readable implementations, and honest release behavior.

## Required checks before submitting

```bash
go test ./...
go run ./cmd/kiro compat
```

If you touch the standalone runtime/release path, also run:

```bash
go run ./cmd/kiro build examples/hello --out ./hello-example
./hello-example
go run ./cmd/kiro test examples/test_demo
KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)" ./scripts/package_release.sh dev 1.22.12 linux amd64
./scripts/write_release_checksums.sh dev
./scripts/verify_release_bundle.sh dist/kiro-dev-linux-amd64.tar.gz
./scripts/verify_install.sh dev
```

## Documentation expectations

When a change is user-visible or architectural, update docs and notes in the same change:

- `README.md`
- relevant `docs/*` pages
- `docs/llm/KIRO_SKILL.md` and `docs/llm/kiro.json` when syntax, stdlib guidance, project conventions, or canonical examples change
- `RELEASE_TOOLCHAIN_NOTES.md` for runtime/release decisions
- `INSTALL_AND_SKILL_NOTES.md` for the compact installer + skill handoff story
- current phase notes (`PHASE10_NOTES.md`, `PHASE11_NOTES.md`, or newer notes when added)
- `AGENTS.md` if workflow guidance changes

## Making intentional language/CLI changes

### Syntax or parser behavior

- add/update parser and formatter tests
- add/update compatibility fixtures
- document behavior in language docs (`docs/language_tour.md`, `docs/stable_core.md`)
- keep the compact `docs/llm/` package aligned with the stable core

### Build/run/test behavior

- keep `kiro build`, `kiro run`, and `kiro test` boring and explicit
- preserve `kiro inspect go` as the first-class debugging path
- keep the bundled-toolchain story honest in docs and release notes
- validate standalone downstream usage, not just in-repo source builds

### Generated-Go changes

- preserve readability and origin tracing where possible
- update `docs/debugging_generated_go.md`
- adjust release/runtime notes if the executable workdir layout changes

### Release automation changes

- keep artifact names predictable
- produce a release-level checksum file that matches installer expectations
- avoid runtime network downloads in normal CLI commands
- prefer deterministic packaging/verification scripts over ad hoc release steps

## Stable-core decision check

Before adding surface area, verify:

1. does it make README/templates/examples materially clearer?
2. can it be tested cheaply and deterministically?
3. can we maintain compatibility expectations for it?
4. does it keep the standalone release story more reliable rather than more magical?
5. will downstream repos and `docs/llm/` stay aligned without copying the whole repo?

If not, prefer docs/polish/cleanup over new surface.
