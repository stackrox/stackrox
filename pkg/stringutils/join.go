package stringutils

import "strings"

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
