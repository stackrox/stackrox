package conv

import "strconv"

// FormatBool returns array of "true" or "false" according to the value of b.
func FormatBool(b ...bool) []string {
	bools := make([]string, 0, len(b))
	for _, val := range b {
		bools = append(bools, strconv.FormatBool(val))
	}
	return bools
}
