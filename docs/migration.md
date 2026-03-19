# Migration guide

## Expression-bodied functions

Old:

```ki
fn greet(name:str) -> str = "hello ${name}"
```

New:

```ki
fn greet(name:str) -> str {
  return "hello ${name}"
}
```

## Filesystem reads

Old:

```ki
fs.read(path)
```

New:

```ki
fs.read_file(path)
```

## Service handlers

Old examples in the repo mixed shorthand type names.

New canonical shape:

```ki
fn handler(req:http.Req) -> R[http.Resp, str]
```
