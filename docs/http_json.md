# HTTP + JSON style

Kiro Phase 7 keeps HTTP ergonomics small and explicit.

## Canonical handler shape

```ki
fn app(req:httpReq) -> Resp {
  when req.path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

## Recommended helpers/patterns

- `http.text(code, body)` for plain responses.
- `http.json(code, body)` for JSON responses.
- `http.not_found()` for default misses.
- `http.test_req(method, path, body)` in handler tests.

## JSON request bodies

If decode support is available in your slice, prefer explicit decoding in handlers:

```ki
let input = json.decode[CreateReq](req.body)?
```

Keep request parsing local to the handler for readability.

See examples:

- `examples/service_health`
- `examples/request_json`
- `examples/test_http_handlers`
