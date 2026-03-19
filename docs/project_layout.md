# Project layout

Kiro encourages one obvious project shape.

## Tiny CLI tool

```text
hello/
  main.ki
```

Canonical entrypoint:

```ki
mod main

fn main() -> i32 !io {
  println("hello")
  return 0
}
```

## Tiny service

```text
service/
  main.ki
  app/
    main.ki
  internal/
    config/
      main.ki
  test/
    health.ki
```

## Layout rules

- keep `main.ki` at the entry root
- keep request handlers in `app/`
- keep env/config loading in `internal/config/`
- keep tests in `test/`
- prefer module names that match paths predictably

## Why this matters

Stable layout conventions help both humans and LLMs generate coherent projects without inventing competing repository shapes.
