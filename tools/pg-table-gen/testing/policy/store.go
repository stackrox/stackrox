package postgres

import (
	"database/sql"

	jsonpb "github.com/gogo/protobuf/jsonpb"
	storage "github.com/stackrox/rox/generated/storage"
)

const createTableQuery = "create table if not exists Policy (Id varchar, ProtoJSONName varchar, Description varchar, Rationale varchar, Remediation varchar, Disabled bool, Categories jsonb, LifecycleStages jsonb, EventSource numeric, Whitelists jsonb, Exclusions jsonb, Scope jsonb, Severity numeric, EnforcementActions jsonb, Notifiers jsonb, PolicyVersion varchar, PolicySections jsonb, MitreAttackVectors jsonb, CriteriaLocked bool, MitreVectorsLocked bool, IsDefault bool, Fields_SetImageAgeDays jsonb, Fields_Cve varchar, Fields_SetScanAgeDays jsonb, Fields_SetNoScanExists jsonb, Fields_Command varchar, Fields_Args varchar, Fields_Directory varchar, Fields_User varchar, Fields_SetPrivileged jsonb, Fields_DropCapabilities jsonb, Fields_AddCapabilities jsonb, Fields_SetReadOnlyRootFs jsonb, Fields_FixedBy varchar, Fields_SetWhitelist jsonb, Fields_ImageName_Registry varchar, Fields_ImageName_Remote varchar, Fields_ImageName_Tag varchar, Fields_LineRule_Instruction varchar, Fields_LineRule_Value varchar, Fields_Cvss_Op numeric, Fields_Cvss_Value numeric, Fields_Component_Name varchar, Fields_Component_Version varchar, Fields_Env_Key varchar, Fields_Env_Value varchar, Fields_Env_EnvVarSource numeric, Fields_VolumePolicy_Name varchar, Fields_VolumePolicy_Source varchar, Fields_VolumePolicy_Destination varchar, Fields_VolumePolicy_SetReadOnly jsonb, Fields_VolumePolicy_Type varchar, Fields_PortPolicy_Port numeric, Fields_PortPolicy_Protocol varchar, Fields_RequiredLabel_Key varchar, Fields_RequiredLabel_Value varchar, Fields_RequiredLabel_EnvVarSource numeric, Fields_RequiredAnnotation_Key varchar, Fields_RequiredAnnotation_Value varchar, Fields_RequiredAnnotation_EnvVarSource numeric, Fields_DisallowedAnnotation_Key varchar, Fields_DisallowedAnnotation_Value varchar, Fields_DisallowedAnnotation_EnvVarSource numeric, Fields_ContainerResourcePolicy_CpuResourceRequest_Op numeric, Fields_ContainerResourcePolicy_CpuResourceRequest_Value numeric, Fields_ContainerResourcePolicy_CpuResourceLimit_Op numeric, Fields_ContainerResourcePolicy_CpuResourceLimit_Value numeric, Fields_ContainerResourcePolicy_MemoryResourceRequest_Op numeric, Fields_ContainerResourcePolicy_MemoryResourceRequest_Value numeric, Fields_ContainerResourcePolicy_MemoryResourceLimit_Op numeric, Fields_ContainerResourcePolicy_MemoryResourceLimit_Value numeric, Fields_ProcessPolicy_Name varchar, Fields_ProcessPolicy_Args varchar, Fields_ProcessPolicy_Ancestor varchar, Fields_ProcessPolicy_Uid varchar, Fields_PortExposurePolicy_ExposureLevels jsonb, Fields_PermissionPolicy_PermissionLevel numeric, Fields_HostMountPolicy_SetReadOnly jsonb, Fields_RequiredImageLabel_Key varchar, Fields_RequiredImageLabel_Value varchar, Fields_RequiredImageLabel_EnvVarSource numeric, Fields_DisallowedImageLabel_Key varchar, Fields_DisallowedImageLabel_Value varchar, Fields_DisallowedImageLabel_EnvVarSource numeric, LastUpdated_Seconds numeric, LastUpdated_Nanos numeric)"

func CreateTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}

var marshaler = (&jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true})

func Upsert(policy *storage.Policy) error {
	var6, err := marshaler.MarshalToString(GetCategories())
	var7, err := marshaler.MarshalToString(GetLifecycleStages())
	var9, err := marshaler.MarshalToString(GetWhitelists())
	var10, err := marshaler.MarshalToString(GetExclusions())
	var11, err := marshaler.MarshalToString(GetScope())
	var13, err := marshaler.MarshalToString(GetEnforcementActions())
	var14, err := marshaler.MarshalToString(GetNotifiers())
	var16, err := marshaler.MarshalToString(GetPolicySections())
	var17, err := marshaler.MarshalToString(GetMitreAttackVectors())
	var21, err := marshaler.MarshalToString(GetFields().GetSetImageAgeDays())
	var23, err := marshaler.MarshalToString(GetFields().GetSetScanAgeDays())
	var24, err := marshaler.MarshalToString(GetFields().GetSetNoScanExists())
	var29, err := marshaler.MarshalToString(GetFields().GetSetPrivileged())
	var30, err := marshaler.MarshalToString(GetFields().GetDropCapabilities())
	var31, err := marshaler.MarshalToString(GetFields().GetAddCapabilities())
	var32, err := marshaler.MarshalToString(GetFields().GetSetReadOnlyRootFs())
	var34, err := marshaler.MarshalToString(GetFields().GetSetWhitelist())
	var50, err := marshaler.MarshalToString(GetFields().GetVolumePolicy().GetSetReadOnly())
	var75, err := marshaler.MarshalToString(GetFields().GetPortExposurePolicy().GetExposureLevels())
	var77, err := marshaler.MarshalToString(GetFields().GetHostMountPolicy().GetSetReadOnly())

	var32, err := marshaler.MarshalToString(policy.GetFields().GetSetReadOnlyRootFs())

}
