package matcher

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Builder builds matchers.
//go:generate mockgen-wrapper Builder
type Builder interface {
	ForPolicy(policy *storage.Policy) (searchbasedpolicies.Matcher, error)
}

// NewBuilder returns a new MatcherBuilder instance using the input registry.
func NewBuilder(registry Registry, optionsMap search.OptionsMap) Builder {
	return &builderImpl{
		registry:   registry,
		optionsMap: optionsMap,
	}
}

type builderImpl struct {
	registry   Registry
	optionsMap search.OptionsMap
}

// ForPolicy returns a matcher for the given policy and options.
func (mb *builderImpl) ForPolicy(policy *storage.Policy) (searchbasedpolicies.Matcher, error) {
	if policy.GetName() == "" {
		return nil, fmt.Errorf("policy %+v doesn't have a name", policy)
	}
	if policy.GetFields() == nil {
		return nil, fmt.Errorf("policy %+v has no fields specified", policy)
	}

	qb := builders.NewConjunctionQueryBuilder(mb.registry...)
	q, v, err := qb.Query(policy.GetFields(), mb.optionsMap.Original())
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
	}, nil
}
