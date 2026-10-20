package main

import (
	"benchmark.local/checkout/internal/checkout"
	"benchmark.local/checkout/internal/refunds"
	"benchmark.local/checkout/internal/reservations"
	"testing"
)

func TestCheckout(t *testing.T) {
	receipt, err := checkout.Place("book", 2, 4, "standard", nil, "card")
	if err != nil || receipt.TotalCents != 3330 || receipt.RemainingStock != 2 || receipt.PaymentReference != "card:3300" {
		t.Fatalf("%+v %v", receipt, err)
	}
}
func TestFreeShipping(t *testing.T) {
	coupon := "SHIPFREE"
	receipt, err := checkout.Place("mug", 1, 1, "express", &coupon, "bank")
	if err != nil || receipt.TotalCents != 1200 {
		t.Fatalf("%+v %v", receipt, err)
	}
}
func TestInvalidStock(t *testing.T) {
	_, err := checkout.Place("book", 2, 1, "standard", nil, "card")
	if err == nil || err.Error() != "insufficient stock: 2/1" {
		t.Fatal(err)
	}
}
func TestReservationBeforeAndAfter(t *testing.T) {
	if !reservations.StatusAt(9, 10).Active || reservations.StatusAt(11, 10).Active {
		t.Fatal("reservation status")
	}
}
func TestFullRefund(t *testing.T) {
	result, err := refunds.Calculate(1500, 2, 2, 300)
	if err != nil || result.RefundCents != 3300 || result.RemainingCents != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestZeroRefund(t *testing.T) {
	result, err := refunds.Calculate(1500, 2, 0, 300)
	if err != nil || result.RefundCents != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}
