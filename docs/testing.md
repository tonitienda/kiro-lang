# Testing

Kiro testing favors explicit test functions and local assertions.

## Canonical test shape

```ki
mod math_test

import test

fn add(a:i32, b:i32) -> i32 {
  return a + b
}

fn test_add() -> nil {
  test.eq(add(2, 3), 5)
}
```

## Rules

- test entry points start with `test_`
- tests use ordinary functions; there is no special test syntax
- keep tests deterministic and local
- keep effectful setup in thin helpers when needed
