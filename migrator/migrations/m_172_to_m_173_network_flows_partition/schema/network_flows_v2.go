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
		PostStmts: []string{},
	}

	ParentIndexStmts = []string{
		// Using ON ONLY for migrations so that child indexes won't impact moving data.  Will add them as part
		// of the migration
		"CREATE INDEX IF NOT EXISTS network_flows_src_v2 ON network_flows_v2 USING hash(props_srcentity_Id)",
		"CREATE INDEX IF NOT EXISTS network_flows_dst_v2 ON network_flows_v2 USING hash(props_dstentity_Id)",
		"CREATE INDEX IF NOT EXISTS network_flows_lastseentimestamp_v2 ON network_flows_v2 USING brin (lastseentimestamp)",
	}

	PartitionIndexes = []PartitionIndex{
		{
			IndexField: "props_srcentity_Id",
			IndexType:  "hash",
			IndexName:  "network_flows_v2_%s_props_srcentity_id_idx",
			ParentName: "network_flows_src_v2",
		},
		{
			IndexField: "props_dstentity_Id",
			IndexType:  "hash",
			IndexName:  "network_flows_v2_%s_props_dstentity_id_idx",
			ParentName: "network_flows_dst_v2",
		},
		//{
		//	IndexField: "clusterid",
		//	IndexType:  "hash",
		//	IndexName:  "network_flows_v2_%s_clusterid_idx",
		//	ParentName: "network_flows_cluster_v2",
		//},
		{
			IndexField: "lastseentimestamp",
			IndexType:  "brin",
			IndexName:  "network_flows_v2_%s_lastseentimestamp_idx",
			ParentName: "network_flows_lastseentimestamp_v2",
		},
	}
)

const (
	// NetworkFlowsTableName holds the database table name
	NetworkFlowsTableName = "network_flows_v2"
)

type PartitionIndex struct {
	IndexField string
	IndexType  string
	IndexName  string
	ParentName string
}
