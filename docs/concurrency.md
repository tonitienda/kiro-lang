# Concurrency with `group`, `spawn`, and `await`

Kiro keeps concurrency explicit and local.

## Canonical pattern

```ki
group {
  let ta = spawn fetch_a()
  let tb = spawn fetch_b()

  let a = await ta
  let b = await tb

  return Ok(http.json(200, json.encode(Resp{a:a b:b})?))
}
```

## Preferred rules

- use `group` for structured concurrent work
- spawn related work together
- await every spawned task explicitly
- keep task lifetimes inside the nearest readable block

## Why this style is preferred

This style makes the task lifecycle visible to readers and to LLMs:

- creation is explicit: `spawn`
- synchronization is explicit: `await`
- scope is explicit: `group`

## Diagnostics

Kiro now uses actionable diagnostics when `await` is used on a non-task and points users toward `spawn`.

## Relationship to effects and results

- concurrency is **not** the same concept as effects
- `spawn`/`await` do not replace `R[T,E]`
- pure functions may still be concurrent
- `json.encode`/`json.decode` remain pure inside concurrent code
