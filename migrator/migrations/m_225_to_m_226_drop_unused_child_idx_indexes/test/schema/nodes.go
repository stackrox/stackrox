// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableNodesStmt holds the create statement for table `nodes`.
	CreateTableNodesStmt = &postgres.CreateStmts{
		GormModel: (*Nodes)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*NodesTaints)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// NodesTableName specifies the name of the table in postgres.
	NodesTableName = "nodes"
	// NodesTaintsTableName specifies the name of the table in postgres.
	NodesTaintsTableName = "nodes_taints"
)

// Nodes holds the Gorm model for Postgres table `nodes`.
type Nodes struct {
	ID                      string            `gorm:"column:id;type:uuid;primaryKey"`
	Name                    string            `gorm:"column:name;type:varchar"`
	ClusterID               string            `gorm:"column:clusterid;type:uuid;index:nodes_sac_filter,type:hash"`
	ClusterName             string            `gorm:"column:clustername;type:varchar"`
	Labels                  map[string]string `gorm:"column:labels;type:jsonb"`
	Annotations             map[string]string `gorm:"column:annotations;type:jsonb"`
	JoinedAt                *time.Time        `gorm:"column:joinedat;type:timestamp"`
	ContainerRuntimeVersion string            `gorm:"column:containerruntime_version;type:varchar"`
	OsImage                 string            `gorm:"column:osimage;type:varchar"`
	LastUpdated             *time.Time        `gorm:"column:lastupdated;type:timestamp"`
	ScanScanTime            *time.Time        `gorm:"column:scan_scantime;type:timestamp"`
	Components              int32             `gorm:"column:components;type:integer"`
	Cves                    int32             `gorm:"column:cves;type:integer"`
	FixableCves             int32             `gorm:"column:fixablecves;type:integer"`
	Priority                int64             `gorm:"column:priority;type:bigint"`
	RiskScore               float32           `gorm:"column:riskscore;type:numeric"`
	TopCvss                 float32           `gorm:"column:topcvss;type:numeric"`
	Serialized              []byte            `gorm:"column:serialized;type:bytea"`
}

// NodesTaints holds the Gorm model for Postgres table `nodes_taints`.
type NodesTaints struct {
	NodesID     string              `gorm:"column:nodes_id;type:uuid;primaryKey"`
	Idx         int                 `gorm:"column:idx;type:integer;primaryKey;index:nodestaints_idx,type:btree"`
	Key         string              `gorm:"column:key;type:varchar"`
	Value       string              `gorm:"column:value;type:varchar"`
	TaintEffect storage.TaintEffect `gorm:"column:tainteffect;type:integer"`
	NodesRef    Nodes               `gorm:"foreignKey:nodes_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
