"""Independent HTTP expectations. Never mounted into an implementing agent's container.

This is public reproducibility source, not secret data. Held-out status requires the trial
broker/container boundary, no network, and a fresh agent that has not read this repository.
"""
import itertools
import json
from pathlib import Path
import re

PRICES = {"book": 1500, "poster": 700, "mug": 1200, "notebook": 500}
SHIPPING = {"standard": 300, "express": 800, "pickup": 0}
TASKS = ("discount-policy", "reservation-expiry", "partial-refund", "payment-boundary", "payment-declined")


def checkout_expected(body, task):
    sku, units, available = body["sku"], body["units"], body["available"]
    delivery, coupon, method = body["delivery"], body.get("coupon"), body["payment"]
    if not 1 <= units <= 100:
        return 422, f"invalid quantity: {units}"
    if available < units:
        return 422, f"insufficient stock: {units}/{available}"
    if sku not in PRICES:
        return 422, f"unknown product: {sku}"
    if delivery not in SHIPPING:
        return 422, f"unknown delivery: {delivery}"
    allowed = (None, "SHIPFREE", "SAVE10") if task == "discount-policy" else (None, "SHIPFREE")
    if coupon not in allowed:
        return 422, f"unknown coupon: {coupon}"
    if method == "declined" and task == "payment-declined":
        return 402, "payment declined"
    if method not in ("card", "bank"):
        return 422, f"unknown payment: {method}"
    subtotal = PRICES[sku] * units
    discount = subtotal // 10 if task == "discount-policy" and coupon == "SAVE10" and subtotal >= 5000 else 0
    shipping = 0 if coupon == "SHIPFREE" else SHIPPING[delivery]
    fee = 30 if method == "card" else 0
    amount = subtotal - discount + shipping
    return 200, dict(sku=sku, units=units, remaining_stock=available - units,
                    subtotal_cents=subtotal, discount_cents=discount, shipping_cents=shipping,
                    fee_cents=fee, total_cents=amount + fee, payment_reference=f"{method}:{amount}")


def cases(task):
    if task not in (*TASKS, "baseline"):
        raise ValueError("unknown task")
    out = []
    def add(group, path, body, status, expected, method="POST"):
        out.append(dict(id=f"{group}-{len(out):04d}", group=group, path=path, method=method,
                        body=json.dumps(body, separators=(",", ":")), status=status, expected=expected))
    for sku, units, delivery, method, coupon in itertools.product(PRICES, (1, 3, 10), SHIPPING, ("card", "bank"), (None, "SHIPFREE")):
        body = dict(sku=sku, units=units, available=100, delivery=delivery, payment=method, coupon=coupon)
        add("checkout-regression", "/checkout", body, *checkout_expected(body, task))
    base = dict(sku="book", units=1, available=5, delivery="standard", coupon=None, payment="card")
    for changes in (dict(units=0), dict(units=-1), dict(units=101), dict(available=0), dict(sku="missing"),
                    dict(delivery="teleport"), dict(coupon="UNKNOWN"), dict(payment="cash"),
                    dict(units=0, sku="missing", payment="declined"), dict(delivery="teleport", payment="declined")):
        body = base | changes
        add("validation-priority", "/checkout", body, *checkout_expected(body, task))
    for sku, units, delivery, method in itertools.product(PRICES, (1, 4, 10), SHIPPING, ("card", "bank")):
        body = base | dict(sku=sku, units=units, available=100, coupon="SAVE10", delivery=delivery, payment=method)
        add("discount-policy", "/checkout", body, *checkout_expected(body, task))
    # Exact threshold and immediately below it, with the available integer prices.
    for units in (9, 10, 11):
        body = base | dict(sku="notebook", units=units, available=100, coupon="SAVE10")
        add("discount-threshold", "/checkout", body, *checkout_expected(body, task))
    for method in ("declined", "declined-other", "card", "bank"):
        body = base | dict(payment=method)
        add("payment-outcome", "/checkout", body, *checkout_expected(body, task))
    for expiry in (-100, 0, 10, 100000):
        for delta in (-1, 0, 1):
            now = expiry + delta
            active = now < expiry if task == "reservation-expiry" else now <= expiry
            add("reservation-boundary", "/reservation", dict(now=now, expires_at=expiry), 200, dict(active=active))
    for price, units, shipping in itertools.product((0, 700, 1500), (1, 2, 5), (0, 300, 800)):
        for returned in sorted({0, 1, units}):
            include_shipping = returned == units if task == "partial-refund" else returned > 0
            refund = price * returned + (shipping if include_shipping else 0)
            add("refund-matrix", "/refund", dict(unit_cents=price, units=units, returned_units=returned, shipping_cents=shipping),
                200, dict(refund_cents=refund, remaining_cents=price * units + shipping - refund))
    for changes in (dict(unit_cents=-1), dict(units=0), dict(returned_units=-1), dict(returned_units=3), dict(shipping_cents=-1)):
        body = dict(unit_cents=700, units=2, returned_units=1, shipping_cents=300) | changes
        add("refund-invalid", "/refund", body, 422, "invalid refund")
    for path in ("/checkout", "/reservation", "/refund"):
        for raw in ("{", "{}", '{"extra":1}', '{"now":1,"now":2,"expires_at":3}'):
            out.append(dict(id=f"schema-{len(out):04d}", group="schema", path=path, method="POST", body=raw, status=400, expected=None))
        add("routing", path, {}, 405, "method not allowed", method="GET")
    for path, valid in (("/checkout", base), ("/reservation", dict(now=1, expires_at=3)),
                        ("/refund", dict(unit_cents=700, units=2, returned_units=1, shipping_cents=300))):
        encoded = json.dumps(valid, separators=(",", ":"))
        key = next(iter(valid))
        invalid = [encoded[:-1] + ',"extra":1}', encoded[:-1] + ',' + json.dumps(key) + ':' + json.dumps(valid[key]) + '}',
                   encoded + '{}', json.dumps(valid | {key: []})]
        for raw in invalid:
            out.append(dict(id=f"strict-schema-{len(out):04d}", group="strict-schema", path=path,
                            method="POST", body=raw, status=400, expected=None))
    omitted = dict(base)
    omitted.pop("coupon")
    add("optional-omission", "/checkout", omitted, *checkout_expected(omitted, task))
    add("routing", "/health", {}, 200, "ok", method="GET")
    add("routing", "/health", {}, 405, "method not allowed")
    add("routing", "/missing", {}, 404, "not found")
    return out


def structure(graph, task):
    if task not in ("payment-boundary", "payment-declined"):
        return dict(required=False, passed=True, checks=[])
    nodes = {n["id"]: n for n in graph["nodes"]}
    checks = []
    def check(name, passed):
        checks.append(dict(name=name, passed=bool(passed)))
    if task == "payment-declined":
        check("distinct checkout PaymentDeclined constructor", "checkout.Error.PaymentDeclined" in nodes)
        check("authorization constructs declined outcome", any(e["from"]=="checkout.authorize" and e["to"]=="checkout.Error.PaymentDeclined" and e["kind"]=="call" for e in graph["edges"]))
    else:
        helpers = ("payment_known", "payment_fee", "payment_reference")
        check("payment helpers owned by payments", all(f"payments.{name}" in nodes and f"checkout.{name}" not in nodes for name in helpers))
        check("payments authorization owns policy", "payments.authorize" in nodes)
        calls = {(e["from"], e["to"]) for e in graph["edges"] if e["kind"] in ("call", "reference", "callback")}
        check("compatibility adapter calls payment module", ("checkout.authorize", "payments.authorize") in calls)
        for helper in helpers:
            check(f"payments.authorize uses {helper}", ("payments.authorize", f"payments.{helper}") in calls)
        forbidden = {"checkout", "inventory", "pricing", "shipping", "catalog", "app", "reservations", "refunds"}
        violations = [e for e in graph["edges"] if nodes.get(e["from"], {}).get("module") == "payments"
                      and nodes.get(e["to"], {}).get("module") in forbidden]
        check("payment module has no forbidden dependencies", not violations)
    return dict(required=True, passed=all(c["passed"] for c in checks), checks=checks)


def api_contracts(symbols):
    expected=json.loads((Path(__file__).parent/"contracts.json").read_text())
    actual={node["id"]:node for node in symbols}
    checks=[]
    def normalized(node,key):
        module=node["id"].split(".",1)[0]
        def qualify(typ):
            return re.sub(r"(?<![\w.])(Authorization|Error|Receipt|Status|Refund|Quote)(?!\w)",lambda match:module+"."+match.group(),typ)
        if key=="return_type":return qualify(node.get(key,""))
        if key=="effects":return sorted(node.get(key,[]))
        return sorted((item["name"],qualify(item["type"]),item.get("default","")) for item in node.get(key,[]))
    for wanted in expected:
        found=actual.get(wanted["id"])
        passed=found is not None and wanted["kind"]==found.get("kind")
        if passed:
            keys=("parameters","return_type","effects") if wanted["kind"]=="function" else ("fields",)
            passed=all(normalized(wanted,key)==normalized(found,key) for key in keys)
        checks.append(dict(symbol=wanted["id"],passed=passed))
    return dict(passed=all(check["passed"] for check in checks),checks=checks)
