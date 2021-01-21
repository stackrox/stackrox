package stringutils

import (
	"strconv"
	"strings"
)

// JoinNonEmpty joins those elements that are not empty, using `joiner` as a joining string.
// E.g., JoinNonEmpty("&", "foo", "", "bar", "") will return "foo&bar".
func JoinNonEmpty(joiner string, elems ...string) string {
	var sb strings.Builder
	for _, elem := range elems {
		if elem == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(joiner)
		}
		sb.WriteString(elem)
	}
	return sb.String()
}

// JoinInt32 joins those integer elements, using `joiner` as a joining string.
func JoinInt32(joiner string, elems ...int32) string {
	var sb strings.Builder
	for _, elem := range elems {
		s := strconv.Itoa(int(elem))
		if sb.Len() > 0 {
			sb.WriteString(joiner)
		}
		sb.WriteString(s)
	}
	return sb.String()
}
