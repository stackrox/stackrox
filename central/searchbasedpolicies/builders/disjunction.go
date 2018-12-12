package builders

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type disjunctionFieldQueryBuilder struct {
	qbs []searchbasedpolicies.PolicyQueryBuilder
}

func (d *disjunctionFieldQueryBuilder) Query(fields *storage.PolicyFields,
	optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.Query, searchbasedpolicies.ViolationPrinter, error) {

	disjuncts, printers, err := presentQueriesAndPrinters(d.qbs, fields, optionsMap)
	if err != nil {
		return nil, nil, err
	}

	if len(disjuncts) == 0 {
		return nil, nil, nil
	}

	return search.DisjunctionQuery(disjuncts...), concatenatingPrinter(printers), nil
}

func (*disjunctionFieldQueryBuilder) Name() string {
	return "disjunction"
}

// NewDisjunctionQueryBuilder returns a new query builder that queries for the disjunction of the passed query builders.
func NewDisjunctionQueryBuilder(qbs ...searchbasedpolicies.PolicyQueryBuilder) searchbasedpolicies.PolicyQueryBuilder {
	return &disjunctionFieldQueryBuilder{qbs: qbs}
}
