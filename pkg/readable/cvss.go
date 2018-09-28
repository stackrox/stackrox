package readable

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// NumericalPolicy formats type *v1.NumericalPolicy into e.g. MAX(field) > 3
func NumericalPolicy(p *v1.NumericalPolicy, field string) string {
	var comparatorChar string
	switch p.GetOp() {
	case v1.Comparator_LESS_THAN:
		comparatorChar = "<"
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorChar = "<="
	case v1.Comparator_EQUALS:
		comparatorChar = "="
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorChar = ">="
	case v1.Comparator_GREATER_THAN:
		comparatorChar = ">"
	}
	return fmt.Sprintf("%v %v %v", field, comparatorChar, p.GetValue())
}
