package readable

import (
	"fmt"
	"strconv"
	"strings"
)

// Float formats a float, returning at most maxPlaces points after
// the decimal place, and aggressively trimming trailing zeros.
// Passing maxPlaces < 0 results in no trimming.
func Float(f float64, maxPlaces int) string {
	var formatString string
	if maxPlaces < 0 {
		formatString = "%f"
	} else {
		formatString = "%." + strconv.Itoa(maxPlaces) + "f"
	}
	formatted := fmt.Sprintf(formatString, f)
	indexDot := strings.Index(formatted, ".")
	if indexDot == -1 {
		return formatted
	}
	keepUpto := len(formatted) - 1
	for ; keepUpto >= indexDot; keepUpto-- {
		if formatted[keepUpto] != '0' && formatted[keepUpto] != '.' {
			break
		}
	}
	return formatted[:keepUpto+1]
}
