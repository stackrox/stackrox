package tracker

import (
	"slices"
	"strings"
)

// orderedValues is a list of elements knowing their order.
type orderedValues []valueOrder

type valueOrder struct {
	int
	string
}

func (ov valueOrder) cmp(b valueOrder) int {
	return ov.int - b.int
}

// join the elements according to their order.
func (ov orderedValues) join(sep rune) string {
	slices.SortFunc(ov, valueOrder.cmp)
	sb := strings.Builder{}
	for _, value := range ov {
		if sb.Len() > 0 {
			sb.WriteRune(sep)
		}
		sb.WriteString(value.string)
	}
	return sb.String()
}
