# Experimental release process

Kiro uses a lightweight release checklist for experimental tags.

## Pre-release checklist

1. Run formatter/parser/sema/codegen test suite:
   - `go test ./...`
2. Run compatibility corpus:
   - `go run ./cmd/kiro compat`
3. Run selected template verification:
   - `go run ./cmd/kiro new hello`
   - `go run ./cmd/kiro new service`
   - `go run ./cmd/kiro check hello`
   - `go run ./cmd/kiro check service`
4. Verify generated-Go inspection flow:
   - `go run ./cmd/kiro inspect go examples/hello --out-dir .kiro-gen`
5. Confirm docs/examples coherence:
   - README quick start
   - `docs/stable_core.md`
   - `docs/examples.md`
   - `docs/compatibility.md`
6. Update release notes/changelog and phase notes.
7. Tag release.

## Release notes shape

Keep notes short and practical:

- stable-core decisions
- compatibility policy updates
- template/example updates
- known limitations
