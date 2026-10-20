"""Scripted reference/negative fixtures for evaluator validation, NOT agent trials.

Only selftest uses these patches. They are not copied into model task workspaces.
"""
import re
from pathlib import Path


def replace_function(text, name, replacement):
    pattern = rf"(?ms)^fn {re.escape(name)}\(.*?(?=^fn |\Z)"
    changed, count = re.subn(pattern, replacement.strip() + "\n\n", text)
    if count != 1:
        raise ValueError(f"expected one {name} function, got {count}")
    return changed


def patch(project, task, negative=False):
    root = Path(project)
    def edit(path, change):
        p = root / path
        p.write_text(change(p.read_text()))
    if task == "discount-policy":
        comparator = ">" if negative else ">="
        edit("internal/pricing/main.ki", lambda s: replace_function(s, "discount", f'''
fn discount(subtotal:i32, coupon:?str) -> R[i32,Error] {{
  when (coupon) {{
    nil => {{ return Ok(value:0) }}
    "SHIPFREE" => {{ return Ok(value:0) }}
    "SAVE10" => {{
      when (subtotal {comparator} 5000) {{
        true => {{ return Ok(value:subtotal / 10) }}
        _ => {{ return Ok(value:0) }}
      }}
    }}
    _ => {{ return Err(error:Error.UnknownCoupon(coupon:coupon)) }}
  }}
}}'''))
    elif task == "reservation-expiry":
        edit("internal/reservations/main.ki", lambda s: s.replace("now > expires_at", "now >= expires_at" if not negative else "now == expires_at"))
    elif task == "partial-refund":
        edit("internal/refunds/main.ki", lambda s: replace_function(s, "shipping_refund", '''
fn shipping_refund(returned_units:i32, units:i32, shipping_cents:i32) -> i32 {
  when (returned_units == units) {
    true => { return shipping_cents }
    _ => { return 0 }
  }
}''' if not negative else '''
fn shipping_refund(returned_units:i32, units:i32, shipping_cents:i32) -> i32 { return 0 }
'''))
    elif task == "payment-declined":
        edit("internal/checkout/main.ki", lambda s: s.replace("type Error {", "type Error {\n variant PaymentDeclined {}", 1))
        p = root / "internal/checkout/main.ki"
        text = p.read_text()
        start = text.index("fn authorize(")
        pos = text.index("{", start) + 1
        text = text[:pos] + '\nwhen (method) {"declined" => {return Err(error:Error.PaymentDeclined())} _ => {}}\n' + text[pos:]
        p.write_text(text)
        status = 422 if negative else 402
        edit("app/main.ki", lambda s: s.replace("when ( error ) {", f'when ( error ) {{\n checkout.Error.PaymentDeclined() => {{return http.text(code:{status},body:"payment declined")}}', 1))
    elif task == "payment-boundary":
        if negative:
            dest = root / "internal/payments/main.ki"
            dest.parent.mkdir()
            dest.write_text('mod payments\nfn unused()->nil {return nil}\n')
            return
        p = root / "internal/checkout/main.ki"
        text = p.read_text()
        start, end = text.index("fn payment_known("), text.index("fn stock(")
        policy = text[start:end]
        dest = root / "internal/payments/main.ki"
        dest.parent.mkdir()
        dest.write_text('mod payments\ntype Error {variant UnknownPayment {method:str}}\ntype Authorization {method:str fee_cents:i32 reference:str}\n' + policy)
        adapter = '''fn authorize(method:str, amount:i32) -> R[Authorization,Error] {
 when (payments.authorize(method:method,amount:amount)) {
  Ok(value:let authorization) => {return Ok(value:Authorization{method:authorization.method,fee_cents:authorization.fee_cents,reference:authorization.reference})}
  Err(error:payments.Error.UnknownPayment(method:let method)) => {return Err(error:Error.UnknownPayment(method:method))}
 }
}
'''
        p.write_text(text[:start] + adapter + text[end:])
        edit("internal/checkout/main.ki", lambda s: s.replace("import pricing", "import pricing\nimport payments", 1))
    else:
        raise ValueError("unknown control task")
