package matcher

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type matcherImpl struct {
	q                *v1.Query
	policyName       string
	violationPrinter searchbasedpolicies.ViolationPrinter
}

func (m *matcherImpl) MatchMany(ctx context.Context, searcher search.Searcher, ids ...string) (map[string]searchbasedpolicies.Violations, error) {
	return m.violationsMapFromQuery(ctx, searcher, search.ConjunctionQuery(search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery(), m.q))
}

func (m *matcherImpl) errorPrefixForMatchOne(id string) string {
	return fmt.Sprintf("matching policy %s against %s", m.policyName, id)
}

func (m *matcherImpl) MatchOne(ctx context.Context, searcher search.Searcher, id string) (violations searchbasedpolicies.Violations, err error) {
	q := search.ConjunctionQuery(search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(), m.q)
	results, err := searcher.Search(ctx, q)
	if err != nil {
		return
	}
	if len(results) == 0 {
		return
	}
	if len(results) > 1 {
		err = fmt.Errorf("%s: got more than one result: %+v", m.errorPrefixForMatchOne(id), results)
		return
	}
	result := results[0]
	if result.ID != id {
		err = fmt.Errorf("%s: id of result %+v did not match passed id", m.errorPrefixForMatchOne(id), result)
		return
	}

	violations = m.violationPrinter(ctx, result)
	if violationsEmpty(violations) {
		err = fmt.Errorf("%s: result matched query but couldn't find any violation messages: %+v", m.errorPrefixForMatchOne(id), result)
		return
	}
	return violations, nil
}

func (m *matcherImpl) Match(ctx context.Context, searcher search.Searcher) (map[string]searchbasedpolicies.Violations, error) {
	return m.violationsMapFromQuery(ctx, searcher, m.q)
}

func (m *matcherImpl) violationsMapFromQuery(ctx context.Context, searcher search.Searcher, q *v1.Query) (map[string]searchbasedpolicies.Violations, error) {
	results, err := searcher.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	violationsMap := make(map[string]searchbasedpolicies.Violations, len(results))
	for _, result := range results {
		if result.ID == "" {
			return nil, fmt.Errorf("matching policy %s: got empty result id: %+v", m.policyName, result)
		}

		violations := m.violationPrinter(ctx, result)
		if violationsEmpty(violations) {
			return nil, fmt.Errorf("matching policy %s: result matched query but couldn't find any violation messages: %+v", m.policyName, result)
		}
		violationsMap[result.ID] = violations
	}
	return violationsMap, nil
}

func violationsEmpty(violations searchbasedpolicies.Violations) bool {
	return len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil
}
