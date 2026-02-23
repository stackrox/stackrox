package views

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// PolicyNameAndSeverity is a lightweight projection of alert data containing only
// the policy name and severity. Used by risk scoring to avoid deserializing full
// alert protobuf blobs when only these two fields are needed.
type PolicyNameAndSeverity struct {
	PolicyName string `db:"policy"`
	Severity   int    `db:"severity"`
}

// GetPolicyName returns the policy name.
func (p *PolicyNameAndSeverity) GetPolicyName() string {
	return p.PolicyName
}

// GetSeverity returns the severity as a storage.Severity enum value.
func (p *PolicyNameAndSeverity) GetSeverity() storage.Severity {
	return storage.Severity(p.Severity)
}

// PolicySeverityCounts holds the count of distinct policies per severity level.
// Used by GraphQL resolvers that only need aggregate policy counts, avoiding
// deserialization of individual alert protobuf blobs.
type PolicySeverityCounts struct {
	LowCount      int `db:"low_policy_count"`
	MediumCount   int `db:"medium_policy_count"`
	HighCount     int `db:"high_policy_count"`
	CriticalCount int `db:"critical_policy_count"`
}

// WithPolicySeverityCountQuery augments a query with filtered COUNT(DISTINCT policy_id)
// selects, one per severity level. Returns a single row with 4 integer columns.
func WithPolicySeverityCountQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.PolicyID).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("low_policy_count",
				search.NewQueryBuilder().
					AddExactMatches(search.Severity, storage.Severity_LOW_SEVERITY.String()).
					ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.PolicyID).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("medium_policy_count",
				search.NewQueryBuilder().
					AddExactMatches(search.Severity, storage.Severity_MEDIUM_SEVERITY.String()).
					ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.PolicyID).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("high_policy_count",
				search.NewQueryBuilder().
					AddExactMatches(search.Severity, storage.Severity_HIGH_SEVERITY.String()).
					ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.PolicyID).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("critical_policy_count",
				search.NewQueryBuilder().
					AddExactMatches(search.Severity, storage.Severity_CRITICAL_SEVERITY.String()).
					ProtoQuery(),
			).Proto(),
	}
	return cloned
}
