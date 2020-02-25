package predicate

import (
	"strconv"
)

func parseInt(value string) (int64, error) {
	i64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return i64, nil
}

func parseUint(value string) (uint64, error) {
	ui64, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return ui64, nil
}

func parseFloat(value string) (float64, error) {
	f64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0, err
	}
	return f64, nil
}
