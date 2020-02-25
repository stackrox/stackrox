package builders

import (
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

// PrivilegedQueryBuilder checks for privileged containers.
type PrivilegedQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (p PrivilegedQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	// We don't match on privileged = false, because that seems pointless.
	if fields.GetSetPrivileged() == nil {
		return
	}
	if !fields.GetPrivileged() {
		return nil, nil, errors.New("Policy can only check for privileged containers")
	}
	searchField, err := getSearchField(search.Privileged, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", p.Name())
		return
	}

	q = search.NewQueryBuilder().AddBoolsHighlighted(search.Privileged, true).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		if match != "true" {
			return ""
		}
		return "Privileged container found"
	})
	return
}

// Name implements the PolicyQueryBuilder interface.
func (PrivilegedQueryBuilder) Name() string {
	return "Query builder for privileged containers"
}
