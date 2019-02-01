package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ScanExistsQueryBuilder is a query for allowing users to flag unscanned images.
type ScanExistsQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (s ScanExistsQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	// We only match if the user specifies that they want to match un-scanned images.
	if !fields.GetNoScanExists() {
		return
	}

	_, err = getSearchFieldNotStored(search.ImageScanTime, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", s.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddNullField(search.ImageScanTime).ProtoQuery()

	v = func(result search.Result, _ searchbasedpolicies.ProcessIndicatorGetter) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{AlertViolations: []*storage.Alert_Violation{{Message: "Image has not been scanned"}}}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (s ScanExistsQueryBuilder) Name() string {
	return "Check whether a scan exists for an image"
}
