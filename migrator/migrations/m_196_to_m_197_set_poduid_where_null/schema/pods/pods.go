// This file copied from pkg/postgres/schema/.

package schema

import (
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTablePodsStmt holds the create statement for table `pods`.
	CreateTablePodsStmt = &postgres.CreateStmts{
		GormModel: (*Pods)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*PodsLiveInstances)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}

	// PodsSchema is the go schema for table `pods`.
	PodsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.VulnerabilityRequest)(nil)), "vulnerability_requests")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_PODS, "pod", (*storage.Pod)(nil)))
		schema.ScopingResource = resources.Deployment
		return schema
	}()
)

const (
	// PodsTableName is the name of the pods table
	PodsTableName = "pods"
	// PodsLiveInstancesTableName specifies the name of the table in postgres.
	PodsLiveInstancesTableName = "pods_live_instances"
)

// Pods holds the Gorm model for Postgres table `pods`.
type Pods struct {
	ID           string `gorm:"column:id;type:uuid;primaryKey"`
	Name         string `gorm:"column:name;type:varchar"`
	DeploymentID string `gorm:"column:deploymentid;type:uuid"`
	Namespace    string `gorm:"column:namespace;type:varchar;index:pods_sac_filter,type:btree"`
	ClusterID    string `gorm:"column:clusterid;type:uuid;index:pods_sac_filter,type:btree"`
	Serialized   []byte `gorm:"column:serialized;type:bytea"`
}

// PodsLiveInstances holds the Gorm model for Postgres table `pods_live_instances`.
type PodsLiveInstances struct {
	PodsID      string `gorm:"column:pods_id;type:uuid;primaryKey"`
	Idx         int    `gorm:"column:idx;type:integer;primaryKey;index:podsliveinstances_idx,type:btree"`
	ImageDigest string `gorm:"column:imagedigest;type:varchar"`
	PodsRef     Pods   `gorm:"foreignKey:pods_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
