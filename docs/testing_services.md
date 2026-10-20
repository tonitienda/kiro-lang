# Testing services

Service tests should target handlers directly before testing full process wiring.

## Canonical handler test

```ki
mod health_test

import app
import http
import test

fn test_health_handler() -> nil {
  let req = http.test_req(method : "GET", path : "/health", body : "")
  let res = app.handler(req : req)?
  test.eq(actual : res.code, expected : 200)
}
```

## Why this style

- easy for models to generate repeatedly
- keeps assertions close to handler behavior
- avoids unnecessary network/process setup for most checks
