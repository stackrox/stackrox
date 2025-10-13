package schema

import (
	"reflect"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableNetworkFlowsStmt holds the create statement for table `network_flows`.
	// The flow store only deals with the identifying information, so this table has been shrunk accordingly
	// The rest of the data is looked up as the graph is built from other sources.
	// Serial flow_id allows for inserts and no updates to speed up writes dramatically
	CreateTableNetworkFlowsStmt = &postgres.CreateStmts{
		GormModel: (*NetworkFlows)(nil),
		PostStmts: []string{
			"CREATE INDEX IF NOT EXISTS network_flows_lastseentimestamp ON network_flows USING brin (lastseentimestamp) WITH (pages_per_range='32')",
		},
	}

	// NetworkFlowsSchema is the go schema for table `nodes`.
	NetworkFlowsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.NetworkFlow)(nil)), "network_flows")
		return schema
	}()
)

const (
	// NetworkFlowsTableName holds the database table name
	NetworkFlowsTableName = "network_flows"
)

// NetworkFlows holds the Gorm model for Postgres table `network_flows`.
type NetworkFlows struct {
	FlowID             string                         `gorm:"column:flow_id;type:bigserial;primaryKey"`
	PropsSrcEntityType storage.NetworkEntityInfo_Type `gorm:"column:props_srcentity_type;type:integer;index:network_flows_src,type:btree"`
	PropsSrcEntityID   string                         `gorm:"column:props_srcentity_id;type:varchar;index:network_flows_src,type:btree"`
	PropsDstEntityType storage.NetworkEntityInfo_Type `gorm:"column:props_dstentity_type;type:integer;index:network_flows_dst,type:btree"`
	PropsDstEntityID   string                         `gorm:"column:props_dstentity_id;type:varchar;index:network_flows_dst,type:btree"`
	PropsDstPort       uint32                         `gorm:"column:props_dstport;type:integer"`
	PropsL4Protocol    storage.L4Protocol             `gorm:"column:props_l4protocol;type:integer"`
	LastSeenTimestamp  *time.Time                     `gorm:"column:lastseentimestamp;type:timestamp"`
	ClusterID          string                         `gorm:"column:clusterid;type:varchar;index:network_flows_cluster,type:btree;index:network_flows_src,type:btree;index:network_flows_dst,type:btree"`
}
