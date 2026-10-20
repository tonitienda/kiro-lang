"""Trusted scripted Go fixture patches; never model-performance trials."""
from pathlib import Path
import re


def replace_function(text, name, replacement):
    pattern = rf"(?ms)^func {re.escape(name)}\(.*?(?=^func |\Z)"
    changed, count = re.subn(pattern, replacement.strip() + "\n", text)
    if count != 1: raise ValueError(f"expected one {name}, got {count}")
    return changed


def patch(project, task, negative=False):
    root = Path(project)
    def edit(path, change):
        file = root / path
        file.write_text(change(file.read_text()))
    if task == "discount-policy":
        comparator = ">" if negative else ">="
        edit("internal/pricing/pricing.go", lambda text: replace_function(text, "Discount", '''func Discount(subtotal int32, coupon *string) (int32, error) {
 if coupon == nil || *coupon == "SHIPFREE" { return 0, nil }
 if *coupon == "SAVE10" {
  if subtotal COMPARATOR 5000 { return subtotal / 10, nil }
  return 0, nil
 }
 return 0, fmt.Errorf("unknown coupon: %s", *coupon)
}'''.replace("COMPARATOR", comparator)))
    elif task == "reservation-expiry":
        edit("internal/reservations/reservations.go", lambda text: text.replace("now > expiresAt", "expiresAt < now" if negative else "now >= expiresAt"))
    elif task == "partial-refund":
        edit("internal/refunds/refunds.go", lambda text: replace_function(text, "ShippingRefund", '''func ShippingRefund(returnedUnits, units, shippingCents int32) int32 {
 if returnedUnits == units { return shippingCents }; return 0
}''' if not negative else '''func ShippingRefund(returnedUnits, units, shippingCents int32) int32 {
 if returnedUnits > 0 { return shippingCents }; return 0
}'''))
    elif task == "payment-declined":
        edit("internal/checkout/checkout.go", lambda text: text.replace('import (', 'import (\n "errors"', 1).replace("type Error = error", 'type Error = error\nvar ErrPaymentDeclined = errors.New("payment declined")', 1).replace("func Authorize(method string, amount int32) (Authorization, error) {", 'func Authorize(method string, amount int32) (Authorization, error) {\n if method == "declined" {return Authorization{}, ErrPaymentDeclined}', 1))
        status = 422 if negative else 402
        edit("app/app.go", lambda text: text.replace("func respond(w http.ResponseWriter, value any, err error) {", f'''func respond(w http.ResponseWriter, value any, err error) {{
 if errors.Is(err, checkout.ErrPaymentDeclined) {{ text(w, {status}, "payment declined"); return }}''', 1))
    elif task == "payment-boundary":
        dest = root / "internal/payments/payments.go"
        dest.parent.mkdir(parents=True, exist_ok=True)
        if negative:
            dest.write_text("package payments\nfunc Unused() {}\n")
            return
        path = root / "internal/checkout/checkout.go"
        text = path.read_text()
        start, end = text.index("func PaymentKnown("), text.index("func Stock(")
        policy = text[start:end]
        dest.write_text('package payments\nimport "fmt"\ntype Authorization struct {Method string; FeeCents int32; Reference string}\n' + policy)
        adapter = '''func Authorize(method string, amount int32) (Authorization, error) {
 authorization, err := payments.Authorize(method, amount)
 if err != nil {return Authorization{}, err}
 return Authorization{authorization.Method, authorization.FeeCents, authorization.Reference}, nil
}
'''
        text = text[:start] + adapter + text[end:]
        text = text.replace('\n\t"fmt"', '').replace('import (', 'import (\n "benchmark.local/checkout/internal/payments"', 1)
        path.write_text(text)
    else: raise ValueError("unknown control task")
