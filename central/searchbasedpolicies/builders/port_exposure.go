package builders

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// PortExposureQueryBuilder checks for exposed ports in containers.
type PortExposureQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (e PortExposureQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	exposureLevels := fields.GetPortExposurePolicy().GetExposureLevels()
	if len(exposureLevels) == 0 {
		return
	}

	searchField, err := getSearchField(search.ExposureLevel, optionsMap)
	if err != nil {
		err = errors.Wrap(err, e.Name())
		return
	}

	queryStrings := make([]string, 0, len(exposureLevels))
	for _, exposureLevel := range exposureLevels {
		queryStrings = append(queryStrings, exposureLevel.String())
	}

	q = search.NewQueryBuilder().AddStringsHighlighted(search.ExposureLevel, queryStrings...).ProtoQuery()
	v = func(result search.Result) searchbasedpolicies.Violations {
		matches := result.Matches[searchField.GetFieldPath()]

		if len(matches) == 0 {
			return searchbasedpolicies.Violations{}
		}

		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{Message: "Port(s) exposed externally found"},
			},
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (PortExposureQueryBuilder) Name() string {
	return "Query builder for port exposure level"
}
