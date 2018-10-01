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
type ViolationPrinter func(search.Result) []*v1.Alert_Violation

// Searcher allows you to search objects.
type Searcher interface {
	Search(q *v1.Query) ([]search.Result, error)
}

// Matcher matches objects against a policy.
type Matcher interface {
	// Match matches the policy against all objects, returning a map from object ID to violations.
	Match(searcher Searcher) (map[string][]*v1.Alert_Violation, error)
	// MatchOne matches the policy against the object with the given id.
	MatchOne(searcher Searcher, fieldLabel search.FieldLabel, id string) ([]*v1.Alert_Violation, error)
}
