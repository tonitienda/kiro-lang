# Language reference

The [current Kiro language reference](llm/references/language.md) is the primary source for
canonical syntax, semantics, builtin contracts and implementation limits. It is bundled
with the compiler and copied into new projects; this page links it rather than maintaining
a second specification.

Agents should start with the [compact guide](llm/SKILL.md), consult the reference's relevant
sections, and use the [machine-readable manifest](llm/references/kiro.json) for capability
and argument-key discovery. No training-time knowledge of Kiro is assumed.

```sh
kiro describe --reference
kiro describe --reference --json
```

The JSON form includes the guide revision, reference schema/path, SHA-256 and exact text.
`kiro describe --json` includes its identity without returning the whole document. New
scaffolds include the same reference identity in `.kiro/version.json`; the existing
`kiro_version`/`skill_version` release fields remain for compatibility, while `guide_version`
identifies the guide revision separately.

Run `go test ./docs/llm` to check and execute the reference's annotated programs, verify
expected compiler rejections/runtime failures, validate builtin signatures and check bundled
multi-module examples. This is example conformance, not a complete formal verification.
No extra analysis runs during ordinary compilation/checking.

Historical milestone notes and the language tour are secondary explanations; proposed
syntax is not supported merely because an old document mentions it. Report disagreements
with the compiler/guide versions and a minimal source example. The
[benchmark onboarding protocol](research/benchmark_onboarding.md) defines how to supply
and account for these materials in fresh agent sessions and a later Go comparison.
