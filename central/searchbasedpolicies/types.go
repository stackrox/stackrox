package searchbasedpolicies

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// A PolicyQueryBuilder can build a query for (a part of) a policy.
type PolicyQueryBuilder interface {
	// Query returns a query matching the field(s) that this particular query builder cares about.
	// It may return all nil values -- this is allowed, and means that the given policy fields
	// don't have a value for any field that this particular query builder cares about.
	Query(*storage.PolicyFields, map[search.FieldLabel]*v1.SearchField) (*v1.Query, ViolationPrinter, error)
	// Name returns a human-friendly name for this fieldQueryBuilder.
	Name() string
}

// Violations represents a list of violation sub-objects.
type Violations struct {
	ProcessViolation *storage.Alert_ProcessViolation
	AlertViolations  []*storage.Alert_Violation
}

// A ViolationPrinter knows how to print violation messages from a search result.
type ViolationPrinter func(search.Result, ProcessIndicatorGetter) Violations

// A ProcessIndicatorGetter knows how to retrieve process indicators given its id.
type ProcessIndicatorGetter interface {
	GetProcessIndicator(id string) (*storage.ProcessIndicator, bool, error)
}

// Matcher matches objects against a policy.
type Matcher interface {
	// Match matches the policy against all objects, returning a map from object ID to violations.
	Match(searcher search.Searcher) (map[string]Violations, error)
	// MatchOne matches the policy against the object with the given id.
	MatchOne(searcher search.Searcher, id string) (Violations, error)
	// MatchMany mathes the policy against just the objects with the given ids.
	MatchMany(searcher search.Searcher, ids ...string) (map[string]Violations, error)
}
