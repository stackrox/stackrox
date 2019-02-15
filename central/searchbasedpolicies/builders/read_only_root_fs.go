package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ReadOnlyRootFSQueryBuilder checks for read only root fs in containers.
type ReadOnlyRootFSQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (p ReadOnlyRootFSQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	// We don't match on readonlyrootfs = true, because that seems pointless.
	if fields.GetSetReadOnlyRootFs() == nil || fields.GetReadOnlyRootFs() {
		return
	}
	searchField, err := getSearchField(search.ReadOnlyRootFilesystem, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", p.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddBoolsHighlighted(search.ReadOnlyRootFilesystem, false).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		if match != "false" {
			return ""
		}
		return "Container using read-write root filesystem found"
	})
	return
}

// Name implements the PolicyQueryBuilder interface.
func (ReadOnlyRootFSQueryBuilder) Name() string {
	return "Query builder for read-write filesystem containers"
}
