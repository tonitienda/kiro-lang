# Concurrency with `group`, `spawn`, `await`

Kiro concurrency stays explicit.

## Recommended pattern

```ki
group {
  let ta = spawn fetch_a()
  let tb = spawn fetch_b()

  let a = await ta
  let b = await tb
  return Ok(http.json(200, json.encode(Resp{a:a b:b})?))
}
```

Guidelines:

- Spawn related work in a single `group` scope.
- Await every spawned task before leaving the group.
- Prefer small fan-out/fan-in over deep task trees.
- When request context exists, pass it explicitly into worker calls.
