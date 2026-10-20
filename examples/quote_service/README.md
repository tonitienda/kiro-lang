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
JSON encoding now contains only data fields, without runtime `_type` metadata.

## Business rules

- One primary product and one optional add-on, one unit each. This is a bounded bundle quote, not an arbitrary cart.
- `book`: 1500 cents; `poster`: 700 cents. Either can occupy either product position.
- Delivery: `standard` costs 300 cents; `express` costs 800 cents.
- `SHIPFREE` sets shipping to zero without changing the subtotal. Unknown coupons fail.
- Use integer cents, not floating point. No taxes, payment, inventory, persistence, or external calls yet.
- JSON may omit optional add-on/coupon fields or supply null; source record constructors require all fields.

`GET /health` returns 200; `POST /quotes` returns a JSON quote; other methods on these
routes return 405; other paths return 404.

## Architecture

- `main.ki`: environment, logging and network boundaries.
- `app/`: request decoding, routing and response encoding.
- `internal/config/`: environment access.
- `internal/catalog/`: product lookup and optional add-on.
- `internal/shipping/`: delivery validation and coupon policy.
- `internal/pricing/`: pure composition and totals.
- `test/`: 20 domain/handler tests, including typed conversion, malformed input and each 422 variant.

The example deliberately avoids `spawn`: these in-memory operations have no useful parallel work.
Introduce concurrency when adding actual independent I/O and testing its lifecycle semantics.

## Enforced boundaries and remaining limits

- Catalog and shipping declare distinct error variants. Pricing converts each explicitly
  into its own closed error type. Exhaustive matches bind and use payloads.
- Typed decoding validates shape and fields. Malformed/nonconforming JSON returns HTTP 400;
  absent optional addon/coupon fields become nil.
- Each pricing error maps to HTTP 422 with its payload. An unexpected serialization Err
  still reaches the adapter's HTTP 500 boundary.
- Add an error variant and checking names the affected converter/handler. Reproduce this
  without changing the example: `python3 scripts/probe_variant_evolution.py --kiro ./kiro`.
- This example keeps its bounded two-product quote. List loops and collection helpers are
  available; cart quantities, bounded totals and overflow policy remain follow-up work.
- Checking remains incremental; constructor coverage is not arbitrary payload-domain or
  return-path analysis. Complete concurrency/effect guarantees remain unfinished.
- Executables package a Go-hosted interpreter; `inspect go` emits structural stubs.

These limits are recorded rather than hidden behind extra example-specific helpers.
See [the research plan](../../docs/research/agent_language_iteration.md) for proposed follow-ups,
comparison controls, and the article/video direction.
