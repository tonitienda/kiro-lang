# Install and Skill Notes

## How version pinning works

Kiro release installs are version-pinned through `scripts/install.sh --version <tag>`.

The installer resolves the current platform, downloads the matching release bundle plus the release checksum file, verifies the bundle checksum, and installs:

- `kiro`
- `kiro-lsp` when present
- the bundled Go toolchain in a sibling `toolchain/` directory

Pinned tags are the intended downstream workflow. `latest` is only a convenience path.

## Artifact naming assumptions

The installer assumes every release publishes these assets:

- `kiro-vX.Y.Z-linux-amd64.tar.gz`
- `kiro-vX.Y.Z-linux-arm64.tar.gz`
- `kiro-vX.Y.Z-darwin-amd64.tar.gz`
- `kiro-vX.Y.Z-darwin-arm64.tar.gz`
- `kiro-vX.Y.Z-checksums.txt`

Each archive contains:

- `bin/kiro`
- `bin/kiro-lsp`
- `toolchain/go/...`
- release docs plus a `VERSION` file

## What the LLM skill package contains

`docs/llm/` is the compact Kiro handoff for downstream repos and automation.

It contains:

- `KIRO_SKILL.md` — concise canonical syntax, semantics, project layout, testing style, mistakes, and examples
- `kiro.json` — compact machine-readable summary of the stable core
- `examples/` — short `.ki` examples for hello, service shape, effects, results/optionals, and concurrency

This is intentionally not a replacement for the full repo docs. It is the small high-signal package to hand to an LLM first.

## How downstream repos should consume it

For a repo such as `kiro-playground`:

1. pin a release with `scripts/install.sh --version <tag> --bin-dir ./.kiro/bin`
2. add `./.kiro/bin` to `PATH`
3. scaffold projects with `kiro new hello` or `kiro new service` so the repo gets a project-local `.kiro/skill/` snapshot plus `.kiro/version.json`
4. give `.kiro/skill/KIRO_SKILL.md` and `.kiro/skill/kiro.json` to the model first; they are pinned to the installed Kiro version used for scaffolding
5. keep local project files and diagnostics small and focused
6. run `kiro check` in CI today, then `kiro build`, `kiro run`, and `kiro test` as that repo adopts the bundled release flow

## `kiro new` vendored skill snapshot

`kiro new` now vendors a small project-local snapshot by default:

```text
.kiro/
  README.md
  version.json
  skill/
    KIRO_SKILL.md
    kiro.json
```

Why this helps:

- downstream repos keep a compact Kiro language handoff next to the code they want an LLM to edit
- editors, agents, and CI prompts can read `.kiro/skill/` without cloning `kiro-lang`
- `.kiro/version.json` records which Kiro version produced the scaffold, which keeps language guidance and release behavior aligned

Current limitations and likely follow-up ideas:

- the vendored snapshot is created at scaffold time; updating it later still means re-running `kiro new` in a fresh project or copying forward manually
- only the compact canonical bundle is vendored today; extra examples stay in the main repo to avoid noisy new-project scaffolds
- `--no-skill` is available for the rare case where a consumer wants to skip the snapshot entirely

## Known limitations

- the installer currently supports linux/darwin on amd64/arm64 only
- release installs are for consumers; building `kiro-lang` itself still requires Go
- `docs/llm/` is compact by design, so deeper language details still live in the main docs
- runtime coverage is still narrower than a full general-purpose language toolchain
- `latest` resolution depends on GitHub release metadata; pinned tags are more reproducible

## Documented compromises

- The installer stays simple and uses release tarballs plus a checksum file instead of inventing a package manager.
- The compact LLM package summarizes the stable core instead of duplicating every doc page.
- The release layout keeps the bundled Go toolchain because downstream repos need `kiro build/run/test` without separately provisioning Go.
