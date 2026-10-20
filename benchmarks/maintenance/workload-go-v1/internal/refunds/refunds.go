package refunds

import "errors"

type Refund struct {
	RefundCents    int32 `json:"refund_cents"`
	RemainingCents int32 `json:"remaining_cents"`
}

func Valid(unitCents, units, returnedUnits, shippingCents int32) bool {
	return unitCents >= 0 && units > 0 && returnedUnits >= 0 && returnedUnits <= units && shippingCents >= 0
}
func ShippingRefund(returnedUnits, units, shippingCents int32) int32 {
	if returnedUnits == 0 {
		return 0
	}
	return shippingCents
}
func Calculate(unitCents, units, returnedUnits, shippingCents int32) (Refund, error) {
	if !Valid(unitCents, units, returnedUnits, shippingCents) {
		return Refund{}, errors.New("invalid refund")
	}
	paid := unitCents*units + shippingCents
	returned := unitCents*returnedUnits + ShippingRefund(returnedUnits, units, shippingCents)
	return Refund{returned, paid - returned}, nil
}
