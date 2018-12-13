package matcher

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type matcherImpl struct {
	q                *v1.Query
	policyName       string
	violationPrinter searchbasedpolicies.ViolationPrinter
	processGetter    searchbasedpolicies.ProcessIndicatorGetter
}

func (m *matcherImpl) MatchMany(searcher searchbasedpolicies.Searcher, ids ...string) (map[string][]*storage.Alert_Violation, error) {
	return m.violationsMapFromQuery(searcher, search.ConjunctionQuery(search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery(), m.q))
}

func (m *matcherImpl) errorPrefixForMatchOne(id string) string {
	return fmt.Sprintf("matching policy %s against %s", m.policyName, id)
}

func (m *matcherImpl) MatchOne(searcher searchbasedpolicies.Searcher, id string) ([]*storage.Alert_Violation, error) {
	q := search.ConjunctionQuery(search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(), m.q)
	results, err := searcher.Search(q)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("%s: got more than one result: %+v", m.errorPrefixForMatchOne(id), results)
	}
	result := results[0]
	if result.ID != id {
		return nil, fmt.Errorf("%s: id of result %+v did not match passed id", m.errorPrefixForMatchOne(id), result)
	}

	violations := m.violationPrinter(result, m.processGetter)
	if len(violations) == 0 {
		return nil, fmt.Errorf("%s: result matched query but couldn't find any violation messages: %+v", m.errorPrefixForMatchOne(id), result)
	}
	return violations, nil
}

func (m *matcherImpl) Match(searcher searchbasedpolicies.Searcher) (map[string][]*storage.Alert_Violation, error) {
	return m.violationsMapFromQuery(searcher, m.q)
}

func (m *matcherImpl) violationsMapFromQuery(searcher searchbasedpolicies.Searcher, q *v1.Query) (map[string][]*storage.Alert_Violation, error) {
	results, err := searcher.Search(q)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	violationsMap := make(map[string][]*storage.Alert_Violation, len(results))
	for _, result := range results {
		if result.ID == "" {
			return nil, fmt.Errorf("matching policy %s: got empty result id: %+v", m.policyName, result)
		}

		violations := m.violationPrinter(result, m.processGetter)
		if len(violations) == 0 {
			return nil, fmt.Errorf("matching policy %s: result matched query but couldn't find any violation messages: %+v", m.policyName, result)
		}
		violationsMap[result.ID] = violations
	}
	return violationsMap, nil
}
