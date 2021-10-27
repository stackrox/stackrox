package postgres

import (
	"database/sql"
	jsonpb "github.com/gogo/protobuf/jsonpb"
	storage "github.com/stackrox/rox/generated/storage"
)

const createTableQuery = "create table if not exists Deployment (Id varchar, ProtoJSONName varchar, Hash numeric, Type varchar, Namespace varchar, NamespaceId varchar, OrchestratorComponent bool, Replicas numeric, Labels jsonb, PodLabels jsonb, ClusterId varchar, ClusterName varchar, Containers jsonb, Annotations jsonb, Priority numeric, Inactive bool, ImagePullSecrets jsonb, ServiceAccount varchar, ServiceAccountPermissionLevel numeric, AutomountServiceAccountToken bool, HostNetwork bool, HostPid bool, HostIpc bool, Tolerations jsonb, Ports jsonb, StateTimestamp numeric, RiskScore numeric, ProcessTags jsonb, LabelSelector_MatchLabels jsonb, LabelSelector_Requirements jsonb, Created_Seconds numeric, Created_Nanos numeric)"

func CreateTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}

var marshaler = (&jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true})

func Upsert(deployment *storage.Deployment) error {
	var12, err := marshaler.MarshalToString(GetContainers())
	var16, err := marshaler.MarshalToString(GetImagePullSecrets())
	var23, err := marshaler.MarshalToString(GetTolerations())
	var24, err := marshaler.MarshalToString(GetPorts())
	var27, err := marshaler.MarshalToString(GetProcessTags())
	var29, err := marshaler.MarshalToString(GetLabelSelector().GetRequirements())
}
