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
