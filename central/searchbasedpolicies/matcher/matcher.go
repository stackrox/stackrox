package matcher

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ForPolicy returns a matcher for the given policy.
func ForPolicy(policy *storage.Policy, optionsMap search.OptionsMap, processGetter searchbasedpolicies.ProcessIndicatorGetter) (searchbasedpolicies.Matcher, error) {
	if policy.GetName() == "" {
		return nil, fmt.Errorf("policy %+v doesn't have a name", policy)
	}
	if policy.GetFields() == nil {
		return nil, fmt.Errorf("policy %+v has no fields specified", policy)
	}

	qb := builders.NewConjunctionQueryBuilder(fields.Registry...)
	q, v, err := qb.Query(policy.GetFields(), optionsMap.Original())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct matcher for policy %s: qb: %s", policy.GetName(), qb.Name())
	}
	if q == nil || v == nil {
		return nil, fmt.Errorf("failed to construct matcher for policy %+v: no fields specified", policy)
	}
	if scopeQuery := scopeToQuery(policy.GetScope()); scopeQuery != nil {
		q = search.NewConjunctionQuery(scopeQuery, q)
	}
	return &matcherImpl{
		q:                q,
		violationPrinter: v,
		policyName:       policy.GetName(),
		processGetter:    processGetter,
	}, nil
}
