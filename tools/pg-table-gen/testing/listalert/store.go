package postgres

import (
	"database/sql"
	jsonpb "github.com/gogo/protobuf/jsonpb"
	storage "github.com/stackrox/rox/generated/storage"
)

const createTableQuery = "create table if not exists ListAlert (Id varchar, LifecycleStage numeric, State numeric, EnforcementCount numeric, Tags jsonb, EnforcementAction numeric, Entity jsonb, Time_Seconds numeric, Time_Nanos numeric, Policy_Id varchar, Policy_Name varchar, Policy_Severity numeric, Policy_Description varchar, Policy_Categories jsonb, Policy_DeveloperInternalFields_SORTName varchar, CommonEntityInfo_ClusterName varchar, CommonEntityInfo_Namespace varchar, CommonEntityInfo_ClusterId varchar, CommonEntityInfo_NamespaceId varchar, CommonEntityInfo_ResourceType numeric)"

func CreateTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}

var marshaler = (&jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true})

func Upsert(listalert *storage.ListAlert) error {
	var4, err := marshaler.MarshalToString(GetTags())
	var6, err := marshaler.MarshalToString(listalert.GetEntity())
	var13, err := marshaler.MarshalToString(GetPolicy().GetCategories())
}
