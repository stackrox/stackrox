package matcher

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
)

var (
	imageFactory      = predicate.NewFactory("image", (*storage.Image)(nil))
	deploymentFactory = predicate.NewFactory("deployment", (*storage.Deployment)(nil))
	processFactory    = predicate.NewFactory("process_indicator", (*storage.ProcessIndicator)(nil))
)

// Builder builds matchers.
//go:generate mockgen-wrapper
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
	if scopeQuery := policyutils.ScopeToQuery(policy.GetScope()); scopeQuery != nil {
		q = search.NewConjunctionQuery(scopeQuery, q)
	}

	// Generate the deployment and image predicate
	imgPredicate, err := imageFactory.GeneratePredicate(q)
	if err != nil {
		return nil, err
	}
	deploymentPredicate, err := deploymentFactory.GeneratePredicate(q)
	if err != nil {
		return nil, err
	}

	processPredicate, err := processFactory.GeneratePredicate(q)
	if err != nil {
		return nil, err
	}

	return &matcherImpl{
		q:                   q,
		imagePredicate:      imgPredicate,
		deploymentPredicate: deploymentPredicate,
		processPredicate:    processPredicate,

		violationPrinter: v,
		policyName:       policy.GetName(),
	}, nil
}
