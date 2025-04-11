package common

import (
	"strings"

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
			Filter(search.CriticalSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.FixableCriticalSeverityCount.Alias(),
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
			Filter(search.ImportantSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.FixableImportantSeverityCount.Alias(),
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
			Filter(search.ModerateSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.FixableModerateSeverityCount.Alias(),
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
			Filter(search.LowSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.FixableLowSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.UnknownSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter(search.FixableUnknownSeverityCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
	)
	return cloned
}

// IsSortBySeverityCounts returns true if the query has sort options requesting to sort by CVE severity counts
func IsSortBySeverityCounts(q *v1.Query) bool {
	severitySortFields := []string{
		search.CriticalSeverityCount.String(),
		search.FixableCriticalSeverityCount.String(),
		search.ImportantSeverityCount.String(),
		search.FixableImportantSeverityCount.String(),
		search.ModerateSeverityCount.String(),
		search.FixableModerateSeverityCount.String(),
		search.LowSeverityCount.String(),
		search.FixableLowSeverityCount.String(),
		search.UnknownSeverityCount.String(),
		search.FixableUnknownSeverityCount.String(),
	}

	// Check to see if a severity sort is in the query
	for _, severity := range severitySortFields {
		if strings.Contains(q.GetPagination().String(), severity) {
			return true
		}
	}

	return false
}
