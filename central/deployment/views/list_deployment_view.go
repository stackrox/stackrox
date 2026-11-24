package views

import (
	"database/sql"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

// ListDeploymentView represents deployment data for list responses.
// This view is used to populate ListDeployment protos from database queries.
// The db tags use search field labels (lowercase with underscores), not database column names.
type ListDeploymentView struct {
	ID                         string          `db:"deployment_id"`
	Hash                       uint64          `db:"deployment_hash"`
	Name                       string          `db:"deployment"`
	ClusterName                string          `db:"cluster"`
	ClusterID                  string          `db:"cluster_id"`
	Namespace                  string          `db:"namespace"`
	Created                    *time.Time      `db:"created"`
	RiskPriorityClassification sql.NullInt32   `db:"risk_priority_classification"`
	RiskScore                  float32         `db:"deployment_risk_score"`
	ProposedRiskScore          sql.NullFloat64 `db:"proposed_deployment_risk_score"`
	RiskActivityClassification sql.NullInt32   `db:"risk_activity_classification"`
	RiskActivitySortOrder      sql.NullInt32   `db:"risk_activity_sort_order"`
	RiskExposureClassification sql.NullInt32   `db:"risk_exposure_classification"`
	RiskExposureSortOrder      sql.NullInt32   `db:"risk_exposure_sort_order"`
	RiskPostureClassification  sql.NullInt32   `db:"risk_posture_classification"`
	RiskPostureSortOrder       sql.NullInt32   `db:"risk_posture_sort_order"`
	// Priority is NOT selected from DB - it's computed by the ranker
}

// ToListDeployment converts the view to a storage.ListDeployment proto.
func (v *ListDeploymentView) ToListDeployment() *storage.ListDeployment {
	classification := storage.RiskPriorityClassification_RISK_PRIORITY_UNKNOWN
	if v.RiskPriorityClassification.Valid {
		classification = storage.RiskPriorityClassification(v.RiskPriorityClassification.Int32)
	}
	proposedRiskScore := float32(0)
	if v.ProposedRiskScore.Valid {
		proposedRiskScore = float32(v.ProposedRiskScore.Float64)
	}
	activityClassification := storage.RiskActivityClassification_RISK_ACTIVITY_UNKNOWN
	if v.RiskActivityClassification.Valid {
		activityClassification = storage.RiskActivityClassification(v.RiskActivityClassification.Int32)
	}
	activitySortOrder := int32(0)
	if v.RiskActivitySortOrder.Valid {
		activitySortOrder = v.RiskActivitySortOrder.Int32
	}
	exposureClassification := storage.RiskExposureClassification_RISK_EXPOSURE_UNKNOWN
	if v.RiskExposureClassification.Valid {
		exposureClassification = storage.RiskExposureClassification(v.RiskExposureClassification.Int32)
	}
	exposureSortOrder := int32(0)
	if v.RiskExposureSortOrder.Valid {
		exposureSortOrder = v.RiskExposureSortOrder.Int32
	}
	postureClassification := storage.RiskPostureClassification_RISK_POSTURE_UNKNOWN
	if v.RiskPostureClassification.Valid {
		postureClassification = storage.RiskPostureClassification(v.RiskPostureClassification.Int32)
	}
	postureSortOrder := int32(0)
	if v.RiskPostureSortOrder.Valid {
		postureSortOrder = v.RiskPostureSortOrder.Int32
	}
	return &storage.ListDeployment{
		Id:                         v.ID,
		Hash:                       v.Hash,
		Name:                       v.Name,
		Cluster:                    v.ClusterName,
		ClusterId:                  v.ClusterID,
		Namespace:                  v.Namespace,
		Created:                    protocompat.ConvertTimeToTimestampOrNil(v.Created),
		RiskPriorityClassification: classification,
		RiskScore:                  v.RiskScore,
		ProposedRiskScore:          proposedRiskScore,
		RiskActivityClassification: activityClassification,
		RiskActivitySortOrder:      activitySortOrder,
		RiskExposureClassification: exposureClassification,
		RiskExposureSortOrder:      exposureSortOrder,
		RiskPostureClassification:  postureClassification,
		RiskPostureSortOrder:       postureSortOrder,
		// Priority is set by updateListDeploymentPriority in the datastore layer
	}
}

// ListDeploymentViewSelects returns the query selects needed for ListDeploymentView.
func ListDeploymentViewSelects() []*v1.QuerySelect {
	return []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Proto(),
		search.NewQuerySelect(search.DeploymentHash).Proto(),
		search.NewQuerySelect(search.DeploymentName).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Namespace).Proto(),
		search.NewQuerySelect(search.Created).Proto(),
		search.NewQuerySelect(search.RiskPriorityClassification).Proto(),
		search.NewQuerySelect(search.DeploymentRiskScore).Proto(),
		search.NewQuerySelect(search.ProposedDeploymentRiskScore).Proto(),
		search.NewQuerySelect(search.RiskActivityClassification).Proto(),
		search.NewQuerySelect(search.RiskActivitySortOrder).Proto(),
		search.NewQuerySelect(search.RiskExposureClassification).Proto(),
		search.NewQuerySelect(search.RiskExposureSortOrder).Proto(),
		search.NewQuerySelect(search.RiskPostureClassification).Proto(),
		search.NewQuerySelect(search.RiskPostureSortOrder).Proto(),
	}
}
