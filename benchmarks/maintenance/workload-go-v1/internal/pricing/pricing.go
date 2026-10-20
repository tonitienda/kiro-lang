package pricing

import (
	"benchmark.local/checkout/internal/catalog"
	"benchmark.local/checkout/internal/shipping"
	"fmt"
)

type Quote struct {
	SubtotalCents int32 `json:"subtotal_cents"`
	DiscountCents int32 `json:"discount_cents"`
	ShippingCents int32 `json:"shipping_cents"`
	AmountCents   int32 `json:"amount_cents"`
}

func UnitPrice(sku string) (int32, error)          { return catalog.Price(sku) }
func DeliveryPrice(delivery string) (int32, error) { return shipping.Price(delivery) }
func Discount(subtotal int32, coupon *string) (int32, error) {
	if coupon == nil || *coupon == "SHIPFREE" {
		return 0, nil
	}
	return 0, fmt.Errorf("unknown coupon: %s", *coupon)
}
func ShippingDiscount(shipping int32, coupon *string) int32 {
	if coupon != nil && *coupon == "SHIPFREE" {
		return 0
	}
	return shipping
}
func QuoteFor(sku string, units int32, delivery string, coupon *string) (Quote, error) {
	unit, err := UnitPrice(sku)
	if err != nil {
		return Quote{}, err
	}
	shipping, err := DeliveryPrice(delivery)
	if err != nil {
		return Quote{}, err
	}
	subtotal := unit * units
	discount, err := Discount(subtotal, coupon)
	if err != nil {
		return Quote{}, err
	}
	charged := ShippingDiscount(shipping, coupon)
	return Quote{subtotal, discount, charged, subtotal - discount + charged}, nil
}
