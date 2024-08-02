package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// WithCountQuery returns a query to count the number of distinct values of the given field
func WithCountQuery(q *v1.Query, field search.FieldLabel) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(field).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}

func WithCountBySeverityAndFixabilityQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("critical_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_critical_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("important_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_important_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("moderate_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_moderate_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("low_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_low_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
	)
	return cloned
}
