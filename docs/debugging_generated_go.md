# Debugging generated Go

Generated Go is part of Kiro's developer experience.

## Command

```bash
kiro inspect go <entry-or-path> --out-dir .kiro-gen
```

## Output layout

- `.kiro-gen/src/...`: translated user modules.
- `.kiro-gen/runtime/...`: runtime/helper partition.
- Headers include source file comments and declaration origin notes.

Use this flow to understand parser/codegen behavior and to report minimal reproducible issues.
