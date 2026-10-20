package catalog

import "fmt"

func Price(sku string) (int32, error) {
	switch sku {
	case "book":
		return 1500, nil
	case "poster":
		return 700, nil
	case "mug":
		return 1200, nil
	case "notebook":
		return 500, nil
	}
	return 0, fmt.Errorf("unknown product: %s", sku)
}
func Name(sku string) (string, error) {
	switch sku {
	case "book":
		return "Book", nil
	case "poster":
		return "Poster", nil
	case "mug":
		return "Mug", nil
	case "notebook":
		return "Notebook", nil
	}
	return "", fmt.Errorf("unknown product: %s", sku)
}
