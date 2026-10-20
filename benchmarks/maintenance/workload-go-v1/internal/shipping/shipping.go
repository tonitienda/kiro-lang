package shipping

import "fmt"

func Price(delivery string) (int32, error) {
	switch delivery {
	case "standard":
		return 300, nil
	case "express":
		return 800, nil
	case "pickup":
		return 0, nil
	}
	return 0, fmt.Errorf("unknown delivery: %s", delivery)
}
