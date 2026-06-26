// Frozen pre-migration GORM schema for pods.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTablePodsStmt holds the create statement for table `pods`.
	CreateTablePodsStmt = &postgres.CreateStmts{
		GormModel: (*Pods)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*PodsLiveInstances)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
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

// TableName returns the table name for GORM.
func (Pods) TableName() string { return "pods" }

// PodsLiveInstances holds the Gorm model for Postgres table `pods_live_instances`.
type PodsLiveInstances struct {
	PodsID      string `gorm:"column:pods_id;type:uuid;primaryKey"`
	Idx         int    `gorm:"column:idx;type:integer;primaryKey;index:podsliveinstances_idx,type:btree"`
	ImageDigest string `gorm:"column:imagedigest;type:varchar"`
	PodsRef     Pods   `gorm:"foreignKey:pods_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (PodsLiveInstances) TableName() string { return "pods_live_instances" }
