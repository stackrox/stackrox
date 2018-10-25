package searchbasedpolicies

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// A PolicyQueryBuilder can build a query for (a part of) a policy.
type PolicyQueryBuilder interface {
	// Query returns a query matching the field(s) that this particular query builder cares about.
	// It may return all nil values -- this is allowed, and means that the given policy fields
	// don't have a value for any field that this particular query builder cares about.
	Query(*v1.PolicyFields, map[search.FieldLabel]*v1.SearchField) (*v1.Query, ViolationPrinter, error)
	// Name returns a human-friendly name for this fieldQueryBuilder.
	Name() string
}

// A ViolationPrinter knows how to print violation messages from a search result.
type ViolationPrinter func(search.Result, ProcessIndicatorGetter) []*v1.Alert_Violation

// A ProcessIndicatorGetter knows how to retrieve process indicators given its id.
type ProcessIndicatorGetter interface {
	GetProcessIndicator(id string) (*v1.ProcessIndicator, bool, error)
}

// Searcher allows you to search objects.
type Searcher interface {
	Search(q *v1.Query) ([]search.Result, error)
}

// Matcher matches objects against a policy.
type Matcher interface {
	// Match matches the policy against all objects, returning a map from object ID to violations.
	Match(searcher Searcher) (map[string][]*v1.Alert_Violation, error)
	// MatchOne matches the policy against the object with the given id.
	MatchOne(searcher Searcher, id string) ([]*v1.Alert_Violation, error)
	// MatchMany mathes the policy against just the objects with the given ids.
	MatchMany(searcher Searcher, ids ...string) (map[string][]*v1.Alert_Violation, error)
}
