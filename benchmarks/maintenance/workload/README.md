# Order-workflow maintenance workload

An intentionally small in-memory HTTP service for benchmark tasks, not a real payment or
inventory system. All requests are independent; there is no database, external provider,
authentication, clock service or persistent order state. Ten modules cover catalog,
inventory validation, pricing, shipping, checkout/payment helpers, reservations, refunds,
HTTP adaptation and public tests. Amounts/timestamps are integer inputs.

Run `kiro test .` or `kiro run .`. ADDR defaults to 127.0.0.1:8080. GET /health returns
200 "ok". POST endpoints below return JSON on success. Other methods on known routes return
405; unknown routes return 404. Invalid/missing/duplicate/extra JSON fields return 400.

- `/checkout`: `{sku,units,available,delivery,coupon?,payment}`. Products: book 1500,
  poster 700, mug 1200, notebook 500 cents. Units must be 1..100 and stock must cover units.
  Standard/express/pickup delivery costs 300/800/0. SHIPFREE removes shipping. Unknown
  products/delivery/coupons/payment methods return 422 text. Card adds 30 cents; bank adds 0.
  Payment references are method:amount before fee. Stock validation precedes pricing,
  then payment validation. Receipt fields are visible in checkout.Receipt.
- `/reservation`: `{now,expires_at}` returns `{active}`. The baseline compares now > expiry.
- `/refund`: `{unit_cents,units,returned_units,shipping_cents}` returns refund_cents and
  remaining_cents. Invalid counts/negative amounts return 422 "invalid refund". The baseline
  refunds shipping on every positive return. Zero return refunds zero; a full return refunds
  all item charges and shipping. Inputs in the workload stay within i32 arithmetic bounds.

The expiry boundary and partial-refund behavior are intentional maintenance-task defects.
Preserve the baseline behavior outside the requested task; tasks are separate fresh copies,
not sequential features. Public tests deliberately omit some acceptance cases. Task specs
state the required changes; evaluator cases are supplied separately from the agent workspace.

This fixture is not an agent benchmark result. No human source-review gate is assumed.
