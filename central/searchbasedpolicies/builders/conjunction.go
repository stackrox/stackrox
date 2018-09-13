package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type conjunctionFieldQueryBuilder struct {
	qbs []searchbasedpolicies.PolicyQueryBuilder
}

func (c *conjunctionFieldQueryBuilder) Query(fields *v1.PolicyFields,
	optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.Query, searchbasedpolicies.ViolationPrinter, error) {
	var conjuncts []*v1.Query
	var matchers []searchbasedpolicies.ViolationPrinter
	for _, qb := range c.qbs {
		conjunct, matcher, err := qb.Query(fields, optionsMap)
		if err != nil {
			return nil, nil, err
		}
		if conjunct == nil {
			continue
		}
		if matcher == nil {
			return nil, nil, fmt.Errorf("query builder %+v (%s) returned non-nil query but nil matcher", qb, qb.Name())
		}
		conjuncts = append(conjuncts, conjunct)
		matchers = append(matchers, matcher)
	}

	if len(conjuncts) == 0 {
		return nil, nil, nil
	}

	concatenatingMatcher := func(result search.Result) (violations []*v1.Alert_Violation) {
		for _, m := range matchers {
			violations = append(violations, m(result)...)
		}
		return
	}

	return search.ConjunctionQuery(conjuncts...), concatenatingMatcher, nil
}

func (c *conjunctionFieldQueryBuilder) Name() string {
	return "conjunction"
}

// NewConjunctionQueryBuilder returns a new query builder that queries for the conjunction of the passed query builders.
func NewConjunctionQueryBuilder(qbs ...searchbasedpolicies.PolicyQueryBuilder) searchbasedpolicies.PolicyQueryBuilder {
	return &conjunctionFieldQueryBuilder{qbs: qbs}
}
