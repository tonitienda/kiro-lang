package checkout

import (
	"benchmark.local/checkout/internal/inventory"
	"benchmark.local/checkout/internal/pricing"
	"fmt"
)

// Error preserves the ordinary Go error contract for callers.
type Error = error

type Authorization struct {
	Method    string `json:"method"`
	FeeCents  int32  `json:"fee_cents"`
	Reference string `json:"reference"`
}
type Receipt struct {
	SKU              string `json:"sku"`
	Units            int32  `json:"units"`
	RemainingStock   int32  `json:"remaining_stock"`
	SubtotalCents    int32  `json:"subtotal_cents"`
	DiscountCents    int32  `json:"discount_cents"`
	ShippingCents    int32  `json:"shipping_cents"`
	FeeCents         int32  `json:"fee_cents"`
	TotalCents       int32  `json:"total_cents"`
	PaymentReference string `json:"payment_reference"`
}

func PaymentKnown(method string) bool { return method == "card" || method == "bank" }
func PaymentFee(method string) int32 {
	if method == "card" {
		return 30
	}
	return 0
}
func PaymentReference(method string, amount int32) string {
	return fmt.Sprintf("%s:%d", method, amount)
}
func Authorize(method string, amount int32) (Authorization, error) {
	if !PaymentKnown(method) {
		return Authorization{}, fmt.Errorf("unknown payment: %s", method)
	}
	return Authorization{method, PaymentFee(method), PaymentReference(method, amount)}, nil
}
func Stock(units, available int32) (int32, error) { return inventory.Remaining(units, available) }
func Priced(sku string, units int32, delivery string, coupon *string) (pricing.Quote, error) {
	return pricing.QuoteFor(sku, units, delivery, coupon)
}
func Place(sku string, units, available int32, delivery string, coupon *string, payment string) (Receipt, error) {
	remaining, err := Stock(units, available)
	if err != nil {
		return Receipt{}, err
	}
	quote, err := Priced(sku, units, delivery, coupon)
	if err != nil {
		return Receipt{}, err
	}
	authorization, err := Authorize(payment, quote.AmountCents)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{sku, units, remaining, quote.SubtotalCents, quote.DiscountCents, quote.ShippingCents, authorization.FeeCents, quote.AmountCents + authorization.FeeCents, authorization.Reference}, nil
}
