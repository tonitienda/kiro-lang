# Readiness snapshot

## What is ready enough for experimental downstream use

- deterministic formatting (`kiro fmt`)
- parser/import/effect validation (`kiro check`)
- generated-Go inspection (`kiro inspect go`)
- standalone CLI builds for hello/service/test flows (`kiro build`, `kiro run`, `kiro test`)
- bundled-toolchain release packaging for Linux/macOS amd64/arm64

## What still needs careful expectations

- breadth of runtime coverage across all examples
- exact generated-Go layout for executable workdirs
- release hardening (versioning, signing, checksums, broader smoke coverage)
