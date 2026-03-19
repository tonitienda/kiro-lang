# Testing

Kiro keeps testing simple.

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

- test functions are named `test_*`
- handler and helper tests should call functions directly when possible
- prefer small, local tests over broad integration harnesses

## Run tests

```bash
kiro test .
```
