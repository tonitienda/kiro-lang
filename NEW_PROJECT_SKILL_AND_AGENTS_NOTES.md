# New Project Skill and AGENTS Notes

## What `kiro new` now generates

Both `kiro new hello` and `kiro new service` now scaffold:

```text
AGENTS.md
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
```

Template-specific source files still stay small and boring:

- `hello/` keeps only `main.ki` plus the shared agent/skill metadata.
- `service/` keeps `main.ki`, `app/`, `internal/config/`, `test/`, `README.md`, plus the shared agent/skill metadata.

## Why `AGENTS.md` is included

Codex and similar agents should not be expected to inspect hidden `.kiro/` directories automatically.

The scaffolded root `AGENTS.md` solves that explicitly:

- it tells agents that the repo contains Kiro source files
- it tells them to read `.kiro/skill/SKILL.md` before editing `.ki` files
- it points them at `.kiro/skill/references/kiro.json` as the compact machine-readable summary
- it reminds them to run `kiro fmt` and `kiro check`
- the service template also reminds them to run `kiro test .`

This keeps downstream repos self-describing even when they do not vendor the full `kiro-lang` repository.

## Why the skill is vendored locally

The vendored bundle is a compact, pinned snapshot of the canonical Kiro LLM guidance from `docs/llm/`.

Vendoring it into each scaffolded project means:

- downstream repos can hand a model one small local skill bundle instead of the whole language repo
- the syntax/examples used by the model stay aligned with the exact installed Kiro version that created the project
- maintainers do not need to invent a second prompt format or custom metadata system per template

The bundle stays intentionally small: one Codex-native `SKILL.md`, one `kiro.json`, and a few canonical examples.

## How version pinning works

`kiro new` writes `.kiro/version.json` with both:

- `kiro_version`
- `skill_version`

Today both values are sourced from `internal/version.KiroVersion`, which is the version embedded into the built or released CLI via linker flags when needed.

That means:

- source builds scaffold with the repository's default version string
- release builds scaffold with the tagged release version embedded in the binary
- the vendored skill snapshot and the CLI version stay pinned together by design

## Canonical source of truth

The scaffolded bundle is copied from `docs/llm/`:

- `docs/llm/SKILL.md`
- `docs/llm/references/kiro.json`
- `docs/llm/references/examples/`

This is the only canonical compact skill package. `kiro new` copies that package recursively instead of maintaining template-specific skill forks.

## Limitations and future improvements

Current limitations:

- the vendored bundle is a scaffold-time snapshot; updating it later is still a manual or re-scaffold task
- `skill_version` currently matches the CLI version exactly instead of carrying a separate bundle revision
- `--no-skill` is still supported for the rare case where a consumer wants to skip vendoring entirely; in that case `AGENTS.md` explains that no local skill snapshot exists

Reasonable future improvements if they become necessary:

- a dedicated refresh command for `.kiro/skill/`
- a separate skill bundle revision only if the project ever needs it
- template-specific README polish without changing the shared skill contract
