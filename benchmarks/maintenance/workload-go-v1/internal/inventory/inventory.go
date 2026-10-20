package inventory

import "fmt"

func Validate(units, available int32) error {
	if units <= 0 || units > 100 {
		return fmt.Errorf("invalid quantity: %d", units)
	}
	if available < units {
		return fmt.Errorf("insufficient stock: %d/%d", units, available)
	}
	return nil
}
func Remaining(units, available int32) (int32, error) {
	if err := Validate(units, available); err != nil {
		return 0, err
	}
	return available - units, nil
}
