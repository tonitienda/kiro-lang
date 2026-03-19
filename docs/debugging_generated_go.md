# Debugging generated Go

Kiro intentionally keeps generated Go visible.

## Use `kiro inspect go` for source-oriented inspection

```bash
kiro inspect go <entry-or-path> --out-dir .kiro-gen
```

This is the clearest way to inspect the compiler-facing Go output that mirrors the Kiro project layout.

## Use `--keep-gen` for executable workdir inspection

`kiro build`, `kiro run`, and `kiro test` now generate a temporary Go work directory for the standalone execution path.

```bash
kiro build <entry> --keep-gen
kiro run <entry> --keep-gen
kiro test <entry> --keep-gen
```

When `--keep-gen` is set, Kiro prints the preserved work directory so you can inspect the exact Go that was compiled into the executable or test runner.

## When to use which path

- Use `inspect go` when you want the explicit source-to-Go inspection story.
- Use `--keep-gen` when you are debugging release/runtime/toolchain behavior.
- Use both when comparing the human-facing generated-Go view with the standalone execution harness.
