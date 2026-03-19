# Testing services

For small services, the preferred pattern is **handler-level testing**.

## Canonical example

```ki
mod health_test

import app
import http
import test

fn test_health_handler() -> nil {
  let req = http.test_req("GET", "/health", "")
  let res = app.handler(req)?
  test.eq(res.code, 200)
}
```

## Why this pattern is preferred

- no server boot required
- request and response types stay explicit
- failures stay close to handler logic
