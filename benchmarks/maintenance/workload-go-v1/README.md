# Go maintenance fixture v1

Standard-library Go counterpart of the sibling Kiro workload. This is a stateless,
in-memory checkout HTTP service, not a real payment or inventory system. The nine
application packages (including main) and root public-test file cover the same business
responsibilities as the ten Kiro modules. Six initial public tests are equivalent to the
Kiro public tests; they deliberately omit maintenance acceptance cases.

Run `go test ./...`, `go vet ./...` or `go run .`. `ADDR` defaults to `127.0.0.1:8080`.

- `GET /health`: 200 `ok`.
- `POST /checkout`: `{sku,units,available,delivery,coupon?,payment}`. Book/poster/mug/notebook
  cost 1500/700/1200/500 cents. Units must be 1..100 and available stock must cover units.
  Standard/express/pickup cost 300/800/0 cents. SHIPFREE removes shipping; card adds 30,
  bank adds 0. References are method:amount before fee. Inventory errors precede pricing
  errors, which precede authorization errors. Domain errors are 422 text.
- `POST /reservation`: `{now,expires_at}` returns `{active}`.
- `POST /refund`: `{unit_cents,units,returned_units,shipping_cents}` returns
  `{refund_cents,remaining_cents}`. Invalid counts/negative amounts return 422 `invalid refund`.

Known routes reject other methods with 405; unknown routes return 404. Input schemas are
closed and case-sensitive. Missing required fields, duplicates, unknown fields, wrong types,
null required fields, invalid UTF-8, non-integer/out-of-int32 numbers and trailing JSON return
400. Coupon may be absent or null. JSON validation diagnostics may differ from Kiro; domain
error messages and successful response values are shared. Inputs stay within non-overflowing
signed int32 arithmetic; Go arithmetic wraparound beyond this domain differs from Kiro's
runtime rejection. The baseline intentionally compares `now > expiresAt` and refunds shipping
on any positive return. Change only the behavior requested by each independent task.

Public Go contracts use exported PascalCase fields/functions, `*string` for optional coupon,
and ordinary `error` results. `QuoteFor`/`StatusAt` avoid Go type/function name collisions.
Aliases or wrappers that preserve compatibility are permitted. Preparation and scripted
controls are not model benchmark results. The complete matched pilot protocol lives in
`benchmarks/maintenance/GO_PILOT.md` in the source repository.
