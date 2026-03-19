# Effects

Kiro uses **explicit function effects** for operational impurity.

## Built-in effects

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

## Canonical signature shape

```ki
fn load() -> R[Config,str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}
```

## Rules

- pure functions declare no effects
- callers must declare every effect required by callees
- unknown effects are rejected
- duplicate effects are rejected
- `kiro fmt` sorts effects canonically

## What is not an effect

Fallibility is not an effect. These stay pure:

- `json.encode`
- `json.decode`
- `parse.i32`
