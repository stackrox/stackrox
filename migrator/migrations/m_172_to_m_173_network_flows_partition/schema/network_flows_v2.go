// Originally copied from pkg/postgres/schema/network_flows_v2.go

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableNetworkFlowsStmt holds the create statement for table `network_flows`.
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
)

const (
	// NetworkFlowsTableName holds the database table name
	NetworkFlowsTableName = "network_flows_v2"
)
