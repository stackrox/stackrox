package common

import (
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// ValidateQuery checks if the query passed to a view does not have a select or group by clause
// We only support a dynamic where clause for queries passed to the view.
// The view implementations add pre-defined select and group by. Remember this is a "view".
func ValidateQuery(q *v1.Query) error {
	if len(q.GetSelects()) > 0 {
		return errors.Errorf("Unexpected select clause in query %q", q.String())
	}
	if q.GetGroupBy() != nil {
		return errors.Errorf("Unexpected group by clause in query %q", q.String())
	}
	return nil
}
