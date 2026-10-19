# Quote service: first agent-maintenance example

A multi-module, deterministic starting point for investigating Kiro with coding agents.
This is a language experiment, not a production checkout system or evidence that Kiro outperforms Go.

## Run

From the repository root, with Go on `PATH`:

```bash
go build -o kiro ./cmd/kiro
./kiro fmt examples/quote_service
./kiro check examples/quote_service
./kiro test examples/quote_service
./kiro run examples/quote_service
```

The default address is `127.0.0.1:8080`; override it with `KIRO_QUOTE_ADDR`.

```bash
curl http://127.0.0.1:8080/health
curl -X POST http://127.0.0.1:8080/quotes \
  -H 'Content-Type: application/json' \
  -d '{"sku":"book","addon":"poster","delivery":"express","coupon":null}'
```

The quote has EUR subtotal 2200 cents, shipping 800 cents, total 3000 cents, and a null coupon.
Current JSON encoding also emits the runtime's `_type` field. Consumers should ignore it;
removing runtime metadata from public JSON is a proposed follow-up, not a stable API promise.

## Business rules

- One primary product and one optional add-on, one unit each. This is a bounded bundle quote, not an arbitrary cart.
- `book`: 1500 cents; `poster`: 700 cents. Either can occupy either product position.
- Delivery: `standard` costs 300 cents; `express` costs 800 cents.
- `SHIPFREE` sets shipping to zero without changing the subtotal. Unknown coupons fail.
- Use integer cents, not floating point. No taxes, payment, inventory, persistence, or external calls yet.
- Send all four request fields explicitly, using JSON null for absent add-on/coupon.

`GET /health` returns 200; `POST /quotes` returns a JSON quote; other methods on these
routes return 405; other paths return 404.

## Architecture

- `main.ki`: environment, logging and network boundaries.
- `app/`: request decoding, routing and response encoding.
- `internal/config/`: environment access.
- `internal/catalog/`: product lookup and optional add-on.
- `internal/shipping/`: delivery validation and coupon policy.
- `internal/pricing/`: pure composition and totals.
- `test/`: 14 domain/handler tests, including invalid leaf operations.

The example deliberately avoids `spawn`: these in-memory operations have no useful parallel work.
Introduce concurrency when adding actual independent I/O and testing its lifecycle semantics.

## Known limits exposed by this example

- `kiro check` does not establish full type correctness, including optional narrowing.
- `json.decode[T]` does not yet validate required fields or field types.
- The runtime turns a failed `?` into an interpreter error rather than returning an `Err`
  to a caller that can inspect it. Invalid inputs currently reach HTTP 500, not a designed
  400/422 response. The leaf error tests do **not** establish compositional error handling.
- The runtime JSON representation exposes `_type` metadata.
- The existing runtime supports less syntax than the frontend accepts; arbitrary carts
  need working collection iteration and arithmetic before they make a useful next checkpoint.
- Executables currently package a Go-hosted interpreter. `inspect go` emits stubs;
  neither path currently gives direct, typed Go lowering of these business functions.

These limits are recorded rather than hidden behind extra example-specific helpers.
See [the research plan](../../docs/research/agent_language_iteration.md) for proposed follow-ups,
comparison controls, and the article/video direction.
