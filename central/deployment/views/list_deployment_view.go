package views

import (
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
	ID          string     `db:"deployment_id"`
	Hash        uint64     `db:"deployment_hash"`
	Name        string     `db:"deployment"`
	ClusterName string     `db:"cluster"`
	ClusterID   string     `db:"cluster_id"`
	Namespace   string     `db:"namespace"`
	Created     *time.Time `db:"created"`
	// Priority is NOT selected from DB - it's computed by the ranker
}

// ToListDeployment converts the view to a storage.ListDeployment proto.
func (v *ListDeploymentView) ToListDeployment() *storage.ListDeployment {
	return &storage.ListDeployment{
		Id:        v.ID,
		Hash:      v.Hash,
		Name:      v.Name,
		Cluster:   v.ClusterName,
		ClusterId: v.ClusterID,
		Namespace: v.Namespace,
		Created:   protocompat.ConvertTimeToTimestampOrNil(v.Created),
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
	}
}
