package views

import (
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// DeploymentIDResult is a lightweight projection of alert data containing only
// the deployment ID. Used by failingDeployments to avoid deserializing full
// alert protobuf blobs when only the deployment ID is needed.
// DeploymentID is a pointer to handle NULL values from resource/image alerts.
type DeploymentIDResult struct {
	DeploymentID *string `db:"deployment_id"`
}

// GetDeploymentID returns the deployment ID, or an empty string if NULL.
func (d *DeploymentIDResult) GetDeploymentID() string {
	if d.DeploymentID == nil {
		return ""
	}
	return *d.DeploymentID
}

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

// AlertMatcher provides the fields needed to match alerts by policy and entity
// without deserializing full protobuf blobs. Both AlertMatchKey (lightweight
// DB projection) and *storage.Alert (via an adapter) can implement this.
type AlertMatcher interface {
	GetId() string
	GetPolicyId() string
	GetState() storage.ViolationState
	GetLifecycleStage() storage.LifecycleStage
	HasDeployment() bool
	GetDeploymentId() string
	IsDeploymentInactive() bool
	HasResource() bool
	GetResourceType() storage.Alert_Resource_ResourceType
	GetResourceName() string
	HasNode() bool
	GetNodeId() string
	GetNodeName() string
	GetClusterId() string
	GetNamespace() string
}

// AlertMatchKey is a lightweight projection of alert data containing only the
// fields needed by mergeManyAlerts to match incoming alerts against previous
// alerts and determine resolution/inactive status. Avoids TOAST I/O by
// reading only inline columns.
type AlertMatchKey struct {
	ID                 string  `db:"alert_id"`
	PolicyID           string  `db:"policy_id"`
	State              int     `db:"violation_state"`
	LifecycleStage     int     `db:"lifecycle_stage"`
	DeploymentID       *string `db:"deployment_id"`
	DeploymentInactive *bool   `db:"inactive_deployment"`
	ResourceType       *int    `db:"resource_type"`
	ResourceName       *string `db:"resource"`
	ClusterID          *string `db:"cluster_id"`
	Namespace          *string `db:"namespace"`
	NodeID             *string `db:"node_id"`
	NodeName           *string `db:"node"`
}

func (k *AlertMatchKey) GetId() string                    { return k.ID }
func (k *AlertMatchKey) GetPolicyId() string              { return k.PolicyID }
func (k *AlertMatchKey) GetState() storage.ViolationState { return storage.ViolationState(k.State) }
func (k *AlertMatchKey) GetLifecycleStage() storage.LifecycleStage {
	return storage.LifecycleStage(k.LifecycleStage)
}

func (k *AlertMatchKey) HasDeployment() bool {
	return k.DeploymentID != nil && *k.DeploymentID != ""
}
func (k *AlertMatchKey) GetDeploymentId() string {
	if k.DeploymentID == nil {
		return ""
	}
	return *k.DeploymentID
}
func (k *AlertMatchKey) IsDeploymentInactive() bool {
	return k.DeploymentInactive != nil && *k.DeploymentInactive
}
func (k *AlertMatchKey) HasResource() bool {
	return k.ResourceName != nil && *k.ResourceName != ""
}
func (k *AlertMatchKey) GetResourceType() storage.Alert_Resource_ResourceType {
	if k.ResourceType == nil {
		return 0
	}
	return storage.Alert_Resource_ResourceType(*k.ResourceType)
}
func (k *AlertMatchKey) GetResourceName() string {
	if k.ResourceName == nil {
		return ""
	}
	return *k.ResourceName
}
func (k *AlertMatchKey) HasNode() bool {
	return k.NodeID != nil && *k.NodeID != ""
}
func (k *AlertMatchKey) GetNodeId() string {
	if k.NodeID == nil {
		return ""
	}
	return *k.NodeID
}
func (k *AlertMatchKey) GetNodeName() string {
	if k.NodeName == nil {
		return ""
	}
	return *k.NodeName
}
func (k *AlertMatchKey) GetClusterId() string {
	if k.ClusterID == nil {
		return ""
	}
	return *k.ClusterID
}
func (k *AlertMatchKey) GetNamespace() string {
	if k.Namespace == nil {
		return ""
	}
	return *k.Namespace
}

// WithAlertMatchKeyQuery augments a query with SELECT on the 12 inline columns
// needed for alert matching: alert_id, policy_id, violation_state, lifecycle_stage,
// deployment_id, inactive_deployment, resource_type, resource, cluster_id,
// namespace, node_id, and node.
func WithAlertMatchKeyQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.AlertID).Proto(),
		search.NewQuerySelect(search.PolicyID).Proto(),
		search.NewQuerySelect(search.ViolationState).Proto(),
		search.NewQuerySelect(search.LifecycleStage).Proto(),
		search.NewQuerySelect(search.DeploymentID).Proto(),
		search.NewQuerySelect(search.Inactive).Proto(),
		search.NewQuerySelect(search.ResourceType).Proto(),
		search.NewQuerySelect(search.ResourceName).Proto(),
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Namespace).Proto(),
		search.NewQuerySelect(search.NodeID).Proto(),
		search.NewQuerySelect(search.Node).Proto(),
	}
	return cloned
}
