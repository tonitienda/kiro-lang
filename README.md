# kiro-lang

Kiro is an **experimental Go-backed language** for **LLM-generated, LLM-reviewed, and LLM-maintained code**.

The repository now treats **aggressive simplification** as a language feature, not a migration inconvenience.

## LLM-oriented design goals

Kiro is designed so a model can:

- generate code in **one obvious syntax**
- recover from diagnostics with **small, local edits**
- distinguish **effects**, **fallibility**, **optionality**, and **concurrency** without guesswork
- rely on **canonical formatting** as part of the language definition
- inspect generated Go when debugging or reviewing lowered code

Kiro optimizes for:

1. **predictability**
2. **regularity**
3. **semantic clarity**
4. **repairability from compiler diagnostics**
5. **canonical formatting**
6. **low ambiguity**
7. **small explicit surface area**
8. **high semantic information per token**

These are design goals, not a measured advantage over existing languages. Read the rationale in `docs/why_kiro_for_llms.md` and the redesign record in `LLM_REDESIGN_NOTES.md`.

The [agent-language research plan](docs/research/agent_language_iteration.md) records the current implementation audit, comparison methodology, and proposed iterations. The [multi-module quote service](examples/quote_service/README.md) is the first runnable integration example for that work.

The [agent-only core reassessment](docs/research/agent_core_reassessment.md) records the next design decisions. Phase 12 introduces breaking case syntax, `when (value) { ... }`, required named value arguments (`f(name: expression)`), and preserves literal contents during formatting. Phase 13 adds [typed function values, explicit partial application, visible defaults and eager pure collection transforms](docs/functional-core.md), demonstrated by the [functional basket example](examples/functional/README.md). Phase 14 adds explicit ordered transducers without intermediate data lists and renames `fold` to `reduce`. Phase 15 adds [compositional results and validated records/JSON](docs/results-and-schemas.md), demonstrated by [validated aggregation](examples/validated_results/README.md). Phase 16 adds [typed variants, exhaustive payload matching, explicit error conversion and executable list loops](docs/variants-and-matching.md), demonstrated by [typed errors](examples/typed_errors/README.md) and the quote service’s 400/422 responses. Phase 17 adds [structured feedback, language/symbol discovery, formatter checks, return-path checking and maintenance-task infrastructure](docs/agent-tools.md). Phase 18 adds [explicit analysis commands](docs/analysis-tools.md) for dependencies, impact, metrics, cycles, candidate boundaries, architecture rules, slicing, Kiro coverage and targeted mutation. These do not run in normal compiler commands. Tagged optional/unit values and structural edit protocols remain proposals.

The [current language reference](docs/language-reference.md) is bundled and available via
`kiro describe --reference`. It includes tested complete programs, explicit rejection cases
and the executable builtin API. Agents receive it through new-project guidance; the
[benchmark onboarding protocol](docs/research/benchmark_onboarding.md) pins versions and
accounts for learning/context costs before comparison with Go.

### Current backend boundary

Runnable programs currently package a Go-hosted interpreter into a Go executable. `kiro inspect go` currently emits structural stubs, not direct typed lowering of the executable business logic. Checking, runtime semantics, and generated-Go inspection are still incomplete; see the research plan for reproducible probes and limitations.

## Core language contract

Kiro now teaches one canonical style:

- **block-only functions** with explicit `return`
- **keyed function types** and explicit `partial` specialization
- **visible literal defaults** selected with `key:default`
- **eager pure collection transforms** via `list.map`, `list.filter`, `list.reduce`
- **explicit fused transducers** with typed `Transform[Input,Output]` plans
- **explicit effects** for operational impurity only
- **`R[T,E]`** for fallibility
- **`?`** propagates an error from the current result-returning function
- **closed record schemas and strict typed JSON**, including nested fields and integer bounds
- **closed typed variants** with explicit payload bindings and exhaustive matching
- **finite list loops** with scoped immutable iteration bindings
- **`?T`** for optional values
- **`nil`** only where an optional or `nil` return type is allowed
- **`group` + `spawn` + `await`** for visible structured concurrency
- **deterministic formatting** via `kiro fmt`

### Canonical function signature

```ki
fn handler(req:http.Req) -> R[http.Resp,str] !net {
  return Ok(value : http.not_found())
}
```

The signature carries the information an LLM most often needs to reason correctly about behavior:

- inputs
- return type
- failure channel
- operational impurity

## Effects are operational, not fallibility

Built-in effects are intentionally small:

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

Pure transforms are **not** effects, even when they return `R[T,E]`:

- `json.encode`
- `json.decode`
- `parse.i32`

## Install Kiro

### Build from source

```bash
go build ./cmd/kiro
go build ./cmd/kiro-lsp
./kiro new hello
./kiro fmt hello
./kiro check hello
./kiro inspect go hello --out-dir hello/.kiro-gen
cat hello/.kiro/version.json
```

Building from source is the right choice for contributors working inside `kiro-lang`. Normal users should prefer the installer plus the packaged VS Code extension.

### Install a pinned release

Use the installer for release bundles that include `kiro`, `kiro-lsp`, and the bundled Go toolchain used by `kiro build`, `kiro run`, and `kiro test`. At runtime Kiro probes that bundled `go` binary before using it, and falls back to `go` on `PATH` when the bundled toolchain is missing or unusable on the current machine.

```bash
./scripts/install.sh --version v0.1.0-experimental
./scripts/install.sh --version v0.1.0-experimental --bin-dir ./bin
```

Artifact naming is predictable and version-pinned:

- `kiro-vX.Y.Z-linux-amd64.tar.gz`
- `kiro-vX.Y.Z-linux-arm64.tar.gz`
- `kiro-vX.Y.Z-darwin-amd64.tar.gz`
- `kiro-vX.Y.Z-darwin-arm64.tar.gz`
- `kiro-vX.Y.Z-checksums.txt`
- `kiro-vscode-vX.Y.Z.vsix`

See `docs/install.md` and `docs/releasing.md` for the full release/install workflow.

## VS Code

The supported VS Code workflow for normal users is:

1. install `kiro` with the normal release installer
2. install the matching `kiro-vscode-vX.Y.Z.vsix` from a release
3. open a folder containing `.ki` files
4. use syntax highlighting, diagnostics, hover, go to definition, document symbols, formatting, and basic completion through `kiro lsp`

The extension treats Kiro as a black-box CLI install and starts the language server with `kiro lsp`. Advanced overrides remain available, but they are optional and not part of the normal install path.

## Quick start

`kiro new hello` scaffolds the entrypoint plus a project-local compact skill snapshot under `.kiro/`:

```text
hello/
  main.ki
  .kiro/
    README.md
    version.json
    skill/
      SKILL.md
      references/
        kiro.json
        examples/
          hello.ki
          service.ki
          effects.ki
          result_optional.ki
          concurrency.ki
  AGENTS.md
```

`version.json` pins the Kiro/skill version used for scaffolding, `.kiro/skill/` is copied from the canonical compact package bundled with the installed CLI, and a root `AGENTS.md` tells Codex/agents to read that vendored skill before editing `.ki` files. Pass `--no-skill` only if you explicitly do not want that snapshot.

## Canonical service shape

The service template and docs now teach a single service layout:

- `main.ki` owns startup and effect boundaries
- `app/` owns request handlers
- `internal/config/` owns environment loading
- `test/` owns handler-level tests

See:

- `docs/project_layout.md`
- `docs/service_structure.md`
- `docs/testing_services.md`
- `docs/http_json.md`

## Compact LLM package

Downstream repositories do not need to feed the whole Kiro repo into a prompt.

Use the compact package in `docs/llm/`:

- `docs/llm/SKILL.md` — Codex-native skill entry with YAML frontmatter
- `docs/llm/references/kiro.json` — compact machine-readable manifest
- `docs/llm/references/examples/` — short canonical examples aligned to the stable core

`kiro new <hello|service>` now vendors a project-local snapshot of that package into `.kiro/skill/`, writes `.kiro/version.json`, and generates a root `AGENTS.md`. Downstream repos such as `kiro-playground` can keep LLM/editor guidance pinned to the installed Kiro version without cloning `kiro-lang`. Keep the source package in sync whenever syntax, stdlib guidance, project layout, or formatting expectations change.

## CLI commands

```bash
./kiro fmt <paths...> [--check [--json]]
./kiro check <entry-or-path> [--json]
./kiro lint <entry-or-path> [--json]
./kiro describe [builtin | --reference] [--json]
./kiro symbols <entry-or-path> [--symbol module.name] [--json]
./kiro analyze <graph|impact|metrics|cycles|boundaries|rules|slice|coverage|mutate> <path> [options] [--json]
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro build <entry-or-path> [--out <file>] [--keep-gen]
./kiro run <entry-or-path> [--keep-gen] [-- <program args...>]
./kiro test <entry-or-path> [--keep-gen]
./kiro compat [root] [--mode fmt,check,inspect]
./kiro new <hello|service> [--no-skill]
./kiro lsp
```

## Compatibility and generated Go

Kiro treats both as first-class trust mechanisms:

- compatibility fixtures protect the **new canonical language**, not old accidental syntax
- `kiro inspect go` keeps lowering visible and debuggable

## Documentation map

### Language

- `docs/design_principles.md`
- `docs/stable_core.md`
- `docs/language_tour.md`
- `docs/effects.md`
- `docs/stdlib_style.md`
- `docs/llm/SKILL.md`

### Install and release

- `docs/install.md`
- `docs/releasing.md`
- `RELEASE_TOOLCHAIN_NOTES.md`
- `INSTALL_AND_SKILL_NOTES.md`

### Projects

- `docs/project_layout.md`
- `docs/service_structure.md`
- `docs/testing.md`
- `docs/testing_services.md`
- `docs/http_json.md`
- `docs/editor_setup.md`

### Rationale and process

- `docs/why_kiro_for_llms.md`
- `docs/compatibility.md`
- `docs/migration.md`
- `LLM_REDESIGN_NOTES.md`
- `docs/contributing.md`

## Development checks

```bash
go test ./...
go run ./cmd/kiro compat
go run ./cmd/kiro check examples/hello
go run ./cmd/kiro inspect go examples/hello --out-dir .kiro-gen-example
```
