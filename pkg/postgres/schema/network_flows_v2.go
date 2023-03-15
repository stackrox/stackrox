package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableNetworkFlowsStmt holds the create statement for table `network_flows_v2`.
	// The flow store only deals with the identifying information, so this table has been shrunk accordingly
	// The rest of the data is looked up as the graph is built from other sources.
	// Serial flow_id allows for inserts and no updates to speed up writes dramatically.
	// Network flows is a partitioned table which is not supported by Gorm, as such, network flows
	// do not utilize the gorm model.  The individual partitions are created on demand and managed in the
	// store as necessary.
	CreateTableNetworkFlowsStmt = &postgres.CreateStmts{
		GormModel: nil,
		PartitionCreate: `CREATE TABLE IF NOT EXISTS network_flows_v2 (
					Flow_id bigserial,
					Props_SrcEntity_Type integer,
					Props_SrcEntity_Id varchar,
					Props_DstEntity_Type integer,
					Props_DstEntity_Id varchar,
					Props_DstPort integer,
					Props_L4Protocol integer,
					LastSeenTimestamp timestamp,
					ClusterId varchar,
					PRIMARY KEY(ClusterId, Flow_id)
			) PARTITION BY LIST (ClusterId)`,
		Partition: true,
		PostStmts: []string{
			"CREATE INDEX IF NOT EXISTS network_flows_src_v2 ON network_flows_v2 USING hash(props_srcentity_Id)",
			"CREATE INDEX IF NOT EXISTS network_flows_dst_v2 ON network_flows_v2 USING hash(props_dstentity_Id)",
			"CREATE INDEX IF NOT EXISTS network_flows_lastseentimestamp_v2 ON network_flows_v2 USING brin (lastseentimestamp)",
		},
	}

	// NetworkFlowsSchema is the go schema for table `nodes`.
	NetworkFlowsSchema = func() *walker.Schema {
		schema := GetSchemaForTable("network_flows_v2")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.NetworkFlow)(nil)), "network_flows_v2")
		RegisterTable(schema, CreateTableNetworkFlowsStmt)
		return schema
	}()
)

const (
	// NetworkFlowsTableName holds the database table name
	NetworkFlowsTableName = "network_flows_v2"
)
