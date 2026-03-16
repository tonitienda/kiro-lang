# HTTP + JSON style

Kiro aims for tiny service ergonomics.

## Suggested handler shape

```ki
fn app(req:httpReq) -> Resp {
  when req.path
    "/health" => {
      return http.text(200, "ok")
    }
    _ => {
      return http.text(404, "not found")
    }
}
```

See examples:

- `examples/service_health`
- `examples/service_config`
- `examples/service_parallel`
