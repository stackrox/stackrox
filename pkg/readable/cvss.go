package readable

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// NumericalPolicy formats type *storage.NumericalPolicy into e.g. MAX(field) > 3
func NumericalPolicy(p *storage.NumericalPolicy, field string) string {
	var comparatorChar string
	switch p.GetOp() {
	case storage.Comparator_LESS_THAN:
		comparatorChar = "<"
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		comparatorChar = "<="
	case storage.Comparator_EQUALS:
		comparatorChar = "="
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorChar = ">="
	case storage.Comparator_GREATER_THAN:
		comparatorChar = ">"
	}
	return fmt.Sprintf("%v %v %v", field, comparatorChar, p.GetValue())
}
