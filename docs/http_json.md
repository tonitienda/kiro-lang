# HTTP + JSON

Kiro treats HTTP as operational and JSON as pure.

## Canonical handler shape

```ki
fn handler(req:http.Req) -> R[http.Resp, str]
```

## Example

```ki
fn handler(req:http.Req) -> R[http.Resp, str] {
  when req.path
    "/health" => {
      let body = json.encode(Status{status:"ok"})?
      return Ok(http.json(200, body))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

## Semantic split

- `http.serve` is `!net`
- request handling may return `R[http.Resp, str]`
- `json.encode` and `json.decode` remain pure
