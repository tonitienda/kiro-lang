# Kiro language tour

Kiro is a small, explicit backend language with a Go backend.

## Kiro way

- Keep syntax small and readable.
- Use canonical formatting via `kiro fmt`.
- Prefer explicit results (`R[T,E]`) and optionals (`?T`).
- Use `spawn`, `await`, and `group` for pragmatic concurrency.

## Minimal shape

```ki
mod main

fn main() -> i32 {
  println("hello")
  return 0
}
```

## Tooling

- `kiro check <entry-or-path>` validates parse + module/import resolution quickly.
- `kiro inspect go <entry-or-path>` emits inspectable Go into `.kiro-gen`.
- `kiro new <hello|service>` scaffolds tiny starter projects.
