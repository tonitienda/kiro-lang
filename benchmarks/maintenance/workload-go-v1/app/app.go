package app

import (
	"benchmark.local/checkout/internal/checkout"
	"benchmark.local/checkout/internal/refunds"
	"benchmark.local/checkout/internal/reservations"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"unicode/utf8"
)

type CheckoutRequest struct {
	SKU       string  `json:"sku"`
	Units     int32   `json:"units"`
	Available int32   `json:"available"`
	Delivery  string  `json:"delivery"`
	Coupon    *string `json:"coupon"`
	Payment   string  `json:"payment"`
}
type ReservationRequest struct {
	Now       int32 `json:"now"`
	ExpiresAt int32 `json:"expires_at"`
}
type RefundRequest struct {
	UnitCents     int32 `json:"unit_cents"`
	Units         int32 `json:"units"`
	ReturnedUnits int32 `json:"returned_units"`
	ShippingCents int32 `json:"shipping_cents"`
}

// decode enforces closed, case-sensitive schemas before decoding Go values.
// Token inspection retains duplicate keys, unlike unmarshalling into a map.
func decode(body io.Reader, target any, required []string, optional string) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if !utf8.Valid(raw) {
		return errors.New("invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("expected JSON object")
	}
	fields := make(map[string]json.RawMessage)
	allowed := make(map[string]bool)
	for _, key := range required {
		allowed[key] = true
	}
	if optional != "" {
		allowed[optional] = true
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("expected field name")
		}
		if !allowed[key] {
			return errors.New("unknown field: " + key)
		}
		if _, exists := fields[key]; exists {
			return errors.New("duplicate field: " + key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		if string(value) == "null" && key != optional {
			return errors.New("null required field: " + key)
		}
		fields[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("trailing JSON")
	}
	for _, key := range required {
		if _, exists := fields[key]; !exists {
			return errors.New("missing field: " + key)
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}
func text(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, _ = io.WriteString(w, body)
}
func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		text(w, 422, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		if r.Method == "GET" {
			text(w, 200, "ok")
		} else {
			text(w, 405, "method not allowed")
		}
		return
	}
	switch r.URL.Path {
	case "/checkout", "/reservation", "/refund":
	default:
		text(w, 404, "not found")
		return
	}
	if r.Method != "POST" {
		text(w, 405, "method not allowed")
		return
	}
	switch r.URL.Path {
	case "/checkout":
		var input CheckoutRequest
		if err := decode(r.Body, &input, []string{"sku", "units", "available", "delivery", "payment"}, "coupon"); err != nil {
			text(w, 400, err.Error())
			return
		}
		receipt, err := checkout.Place(input.SKU, input.Units, input.Available, input.Delivery, input.Coupon, input.Payment)
		respond(w, receipt, err)
	case "/reservation":
		var input ReservationRequest
		if err := decode(r.Body, &input, []string{"now", "expires_at"}, ""); err != nil {
			text(w, 400, err.Error())
			return
		}
		respond(w, reservations.StatusAt(input.Now, input.ExpiresAt), nil)
	case "/refund":
		var input RefundRequest
		if err := decode(r.Body, &input, []string{"unit_cents", "units", "returned_units", "shipping_cents"}, ""); err != nil {
			text(w, 400, err.Error())
			return
		}
		refund, err := refunds.Calculate(input.UnitCents, input.Units, input.ReturnedUnits, input.ShippingCents)
		respond(w, refund, err)
	}
}
