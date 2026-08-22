package schema

import (
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
					UpdatedAt timestamp,
					PRIMARY KEY(ClusterId, Flow_id)
			) PARTITION BY LIST (ClusterId)`,
		Partition: true,
		PostStmts: []string{
			"CREATE INDEX IF NOT EXISTS network_flows_src_v2 ON network_flows_v2 USING btree(props_srcentity_Id)",
			"CREATE INDEX IF NOT EXISTS network_flows_dst_v2 ON network_flows_v2 USING btree(props_dstentity_Id)",
			"CREATE INDEX IF NOT EXISTS network_flows_lastseentimestamp_v2 ON network_flows_v2 USING brin (lastseentimestamp)",
			"CREATE INDEX IF NOT EXISTS network_flows_updatedat_v2 ON network_flows_v2 USING brin (updatedat)",
		},
	}

	// NetworkFlowsSchema is the go schema for table `nodes`.
	NetworkFlowsSchema = func() *walker.Schema {
		schema := GetSchemaForTable("network_flows_v2")
		if schema != nil {
			return schema
		}
		schema = &walker.Schema{
			Table:    "network_flows_v2",
			Type:     "*storage.NetworkFlow",
			TypeName: "NetworkFlow",
		}
		schema.Fields = []walker.Field{
			{Schema: schema, Name: "Id", ProtoBufName: "id", ColumnName: "Props_SrcEntity_Id", Type: "string", DataType: postgres.String, SQLType: "varchar", ModelType: "string", ObjectGetter: walker.MakeObjectGetter("GetProps().GetSrcEntity().GetId()", false), Options: walker.PostgresOptions{PrimaryKey: true}},
			{Schema: schema, Name: "Cidr", ProtoBufName: "cidr", ColumnName: "Props_SrcEntity_ExternalSource_Cidr", Type: "string", DataType: postgres.String, SQLType: "cidr", ModelType: "string", ObjectGetter: walker.MakeObjectGetter("GetProps().GetSrcEntity().GetExternalSource().GetCidr()", false), Options: walker.PostgresOptions{ColumnType: "cidr"}},
			{Schema: schema, Name: "Default", ProtoBufName: "default", ColumnName: "Props_SrcEntity_ExternalSource_Default", Type: "bool", DataType: postgres.Bool, SQLType: "bool", ModelType: "bool", ObjectGetter: walker.MakeObjectGetter("GetProps().GetSrcEntity().GetExternalSource().GetDefault()", false)},
			{Schema: schema, Name: "Discovered", ProtoBufName: "discovered", ColumnName: "Props_SrcEntity_ExternalSource_Discovered", Type: "bool", DataType: postgres.Bool, SQLType: "bool", ModelType: "bool", ObjectGetter: walker.MakeObjectGetter("GetProps().GetSrcEntity().GetExternalSource().GetDiscovered()", false)},
			{Schema: schema, Name: "Id", ProtoBufName: "id", ColumnName: "Props_DstEntity_Id", Type: "string", DataType: postgres.String, SQLType: "varchar", ModelType: "string", ObjectGetter: walker.MakeObjectGetter("GetProps().GetDstEntity().GetId()", false), Options: walker.PostgresOptions{PrimaryKey: true}},
			{Schema: schema, Name: "Cidr", ProtoBufName: "cidr", ColumnName: "Props_DstEntity_ExternalSource_Cidr", Type: "string", DataType: postgres.String, SQLType: "cidr", ModelType: "string", ObjectGetter: walker.MakeObjectGetter("GetProps().GetDstEntity().GetExternalSource().GetCidr()", false), Options: walker.PostgresOptions{ColumnType: "cidr"}},
			{Schema: schema, Name: "Default", ProtoBufName: "default", ColumnName: "Props_DstEntity_ExternalSource_Default", Type: "bool", DataType: postgres.Bool, SQLType: "bool", ModelType: "bool", ObjectGetter: walker.MakeObjectGetter("GetProps().GetDstEntity().GetExternalSource().GetDefault()", false)},
			{Schema: schema, Name: "Discovered", ProtoBufName: "discovered", ColumnName: "Props_DstEntity_ExternalSource_Discovered", Type: "bool", DataType: postgres.Bool, SQLType: "bool", ModelType: "bool", ObjectGetter: walker.MakeObjectGetter("GetProps().GetDstEntity().GetExternalSource().GetDiscovered()", false)},
			{Schema: schema, Name: "DstPort", ProtoBufName: "dst_port", ColumnName: "Props_DstPort", Type: "uint32", DataType: postgres.BigInteger, SQLType: "bigint", ModelType: "uint32", ObjectGetter: walker.MakeObjectGetter("GetProps().GetDstPort()", false), Options: walker.PostgresOptions{PrimaryKey: true}},
			{Schema: schema, Name: "L4Protocol", ProtoBufName: "l4protocol", ColumnName: "Props_L4Protocol", Type: "storage.L4Protocol", DataType: postgres.Enum, SQLType: "integer", ModelType: "storage.L4Protocol", ObjectGetter: walker.MakeObjectGetter("GetProps().GetL4Protocol()", false), Options: walker.PostgresOptions{PrimaryKey: true}},
			{Schema: schema, Name: "ClusterId", ProtoBufName: "cluster_id", ColumnName: "ClusterId", Type: "string", DataType: postgres.String, SQLType: "uuid", ModelType: "string", ObjectGetter: walker.MakeObjectGetter("GetClusterId()", false), Options: walker.PostgresOptions{PrimaryKey: true, ColumnType: "uuid"}},
			{Schema: schema, Name: "serialized", ProtoBufName: "", ColumnName: "serialized", Type: "[]byte", DataType: postgres.DataType(""), SQLType: "bytea", ModelType: "[]byte", ObjectGetter: walker.MakeObjectGetter("serialized", true)},
		}
		RegisterTable(schema, CreateTableNetworkFlowsStmt)
		return schema
	}()
)

const (
	// NetworkFlowsTableName holds the database table name
	NetworkFlowsTableName = "network_flows_v2"
)
