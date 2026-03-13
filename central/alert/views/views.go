package views

import (
	"time"

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

// AlertPolicyGroup holds the result of a GROUP BY query that counts alerts per policy.
// Used by GetAlertsGroup to avoid deserializing full alert protobuf blobs.
type AlertPolicyGroup struct {
	PolicyID    string   `db:"policy_id"`
	PolicyName  string   `db:"policy"`
	Severity    int      `db:"severity"`
	Description string   `db:"description"`
	Categories  []string `db:"category"`
	NumAlerts   int      `db:"alert_id_count"`
}

// GetPolicySeverity returns the severity as a storage.Severity enum value.
func (g *AlertPolicyGroup) GetPolicySeverity() storage.Severity {
	return storage.Severity(g.Severity)
}

// AlertTimeseriesEvent is a lightweight projection of alert data containing only
// the fields needed by GetAlertTimeseries: id, cluster name, severity, time, and state.
// Used to avoid deserializing full alert protobuf blobs.
type AlertTimeseriesEvent struct {
	AlertID     string     `db:"alert_id"`
	ClusterName string     `db:"cluster"`
	Severity    int        `db:"severity"`
	Time        *time.Time `db:"violation_time"`
	State       int        `db:"violation_state"`
}

// GetAlertID returns the alert ID.
func (e *AlertTimeseriesEvent) GetAlertID() string {
	return e.AlertID
}

// GetClusterName returns the cluster name.
func (e *AlertTimeseriesEvent) GetClusterName() string {
	return e.ClusterName
}

// GetSeverity returns the severity as a storage.Severity enum value.
func (e *AlertTimeseriesEvent) GetSeverity() storage.Severity {
	return storage.Severity(e.Severity)
}

// GetTimeMillis returns the violation time in milliseconds since epoch.
func (e *AlertTimeseriesEvent) GetTimeMillis() int64 {
	if e.Time == nil {
		return 0
	}
	return e.Time.Unix() * 1000
}

// GetState returns the violation state as a storage.ViolationState enum value.
func (e *AlertTimeseriesEvent) GetState() storage.ViolationState {
	return storage.ViolationState(e.State)
}

// WithAlertTimeseriesQuery augments a query with SELECT on the 5 fields needed
// by GetAlertTimeseries: alert_id, cluster, severity, violation_time, and violation_state.
// Results are sorted by violation time ascending.
func WithAlertTimeseriesQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.AlertID).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.Severity).Proto(),
		search.NewQuerySelect(search.ViolationTime).Proto(),
		search.NewQuerySelect(search.ViolationState).Proto(),
	}
	cloned.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.ViolationTime.String(),
			},
		},
	}
	return cloned
}

// WithAlertPolicyGroupQuery augments a query with COUNT(alert_id) and GROUP BY
// on policy_id, policy name, severity, and description. Results are sorted by
// policy name ascending.
func WithAlertPolicyGroupQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.PolicyID).Proto(),
		search.NewQuerySelect(search.PolicyName).Proto(),
		search.NewQuerySelect(search.Severity).Proto(),
		search.NewQuerySelect(search.Description).Proto(),
		search.NewQuerySelect(search.Category).Proto(),
		search.NewQuerySelect(search.AlertID).AggrFunc(aggregatefunc.Count).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.PolicyID.String(),
			search.PolicyName.String(),
			search.Severity.String(),
			search.Description.String(),
			search.Category.String(),
		},
	}
	cloned.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.PolicyName.String(),
			},
		},
	}
	return cloned
}

// WithPolicySeverityCountQuery augments a query with filtered COUNT(DISTINCT policy_id)
// selects, one per severity level. Returns a single row with 4 integer columns.
func WithPolicySeverityCountQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		buildSelectCountBySeverity("low_policy_count", storage.Severity_LOW_SEVERITY),
		buildSelectCountBySeverity("medium_policy_count", storage.Severity_MEDIUM_SEVERITY),
		buildSelectCountBySeverity("high_policy_count", storage.Severity_HIGH_SEVERITY),
		buildSelectCountBySeverity("critical_policy_count", storage.Severity_CRITICAL_SEVERITY),
	}
	return cloned
}

func buildSelectCountBySeverity(filterName string, severity storage.Severity) *v1.QuerySelect {
	return search.NewQuerySelect(search.PolicyID).
		Distinct().
		AggrFunc(aggregatefunc.Count).
		Filter(filterName,
			search.NewQueryBuilder().
				AddExactMatches(search.Severity, severity.String()).
				ProtoQuery(),
		).Proto()
}
