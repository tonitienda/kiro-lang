# HTTP and JSON

Kiro keeps HTTP and JSON semantics intentionally separate.

## Canonical handler

```ki
fn handler(req:http.Req) -> R[http.Resp,str] {
  when req.path
    "/status" => {
      let body = json.encode(Status{status:"ok"})?
      return Ok(http.json(200, body))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

## Important semantic rule

- `http.serve` is operational and requires `!net`
- `json.encode` and `json.decode` are pure and return `R[T,E]`

Do not declare `!json`.
