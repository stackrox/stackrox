package matcher

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher allows you to search objects.
type Searcher interface {
	Search(q *v1.Query) ([]search.Result, error)
}

// ForPolicy returns a matcher for the given policy.
func ForPolicy(policy *v1.Policy, optionsMap map[search.FieldLabel]*v1.SearchField) (Matcher, error) {
	if policy.GetName() == "" {
		return nil, fmt.Errorf("policy %+v doesn't have a name", policy)
	}
	if policy.GetFields() == nil {
		return nil, fmt.Errorf("policy %+v has no fields specified", policy)
	}

	qb := builders.NewConjunctionQueryBuilder(fields.Registry...)
	q, v, err := qb.Query(policy.GetFields(), optionsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct matcher for policy %s: qb: %s, %s", policy.GetName(), qb.Name(), err)
	}
	if q == nil || v == nil {
		return nil, fmt.Errorf("failed to construct matcher for policy %+v: no fields specified", policy)
	}
	return &matcherImpl{
		q:                q,
		violationPrinter: v,
		policyName:       policy.GetName(),
	}, nil
}

// Matcher matches objects against a policy.
type Matcher interface {
	// Match matches the policy against all objects, returning a map from object ID to violations.
	Match(searcher Searcher) (map[string][]*v1.Alert_Violation, error)
	// MatchOne matches the policy against the object with the given id.
	MatchOne(searcher Searcher, fieldLabel search.FieldLabel, id string) ([]*v1.Alert_Violation, error)
}
