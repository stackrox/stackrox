package parse

import (
	"errors"
	"strconv"
	"strings"
)

// FriendlyParseBool checks more potential values to match to true and false
// than strconv.ParseBool
func FriendlyParseBool(value string) (bool, error) {
	if value == "" {
		return false, errors.New("empty value is neither true nor false")
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		lower := strings.ToLower(value)
		if strings.HasPrefix("true", lower) {
			return true, nil
		}
		if strings.HasPrefix("false", lower) {
			return false, nil
		}
		return false, err
	}
	return b, nil
}
