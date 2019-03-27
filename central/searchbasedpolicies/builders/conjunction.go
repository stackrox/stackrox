package builders

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type conjunctionFieldQueryBuilder struct {
	qbs []searchbasedpolicies.PolicyQueryBuilder
}

func (c *conjunctionFieldQueryBuilder) Query(fields *storage.PolicyFields,
	optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.Query, searchbasedpolicies.ViolationPrinter, error) {

	conjuncts, printers, err := presentQueriesAndPrinters(c.qbs, fields, optionsMap)
	if err != nil {
		return nil, nil, err
	}

	if len(conjuncts) == 0 {
		return nil, nil, nil
	}

	return search.ConjunctionQuery(conjuncts...), concatenatingPrinter(printers), nil
}

func (c *conjunctionFieldQueryBuilder) Name() string {
	return "conjunction"
}

// NewConjunctionQueryBuilder returns a new query builder that queries for the conjunction of the passed query builders.
func NewConjunctionQueryBuilder(qbs ...searchbasedpolicies.PolicyQueryBuilder) searchbasedpolicies.PolicyQueryBuilder {
	return &conjunctionFieldQueryBuilder{qbs: qbs}
}
