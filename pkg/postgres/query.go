package postgres

import (
	"strconv"
	"strings"
)

func GetValues(start, end int) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i := start; i < end; i++ {
		sb.WriteString("$" + strconv.Itoa(i))
		if i != end-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}
