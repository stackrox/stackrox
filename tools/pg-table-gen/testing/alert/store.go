package postgres

import (
	"database/sql"
	jsonpb "github.com/gogo/protobuf/jsonpb"
	storage "github.com/stackrox/rox/generated/storage"
)

const createTableQuery = "create table if not exists Alert (Id varchar, LifecycleStage numeric, Entity jsonb, Violations jsonb, State numeric, Tags jsonb, Policy_Id varchar, Policy_Name varchar, Policy_Description varchar, Policy_Rationale varchar, Policy_Remediation varchar, Policy_Disabled bool, Policy_Categories jsonb, Policy_LifecycleStages jsonb, Policy_EventSource numeric, Policy_Whitelists jsonb, Policy_Exclusions jsonb, Policy_Scope jsonb, Policy_Severity numeric, Policy_EnforcementActions jsonb, Policy_Notifiers jsonb, Policy_PolicyVersion varchar, Policy_PolicySections jsonb, Policy_MitreAttackVectors jsonb, Policy_CriteriaLocked bool, Policy_MitreVectorsLocked bool, Policy_IsDefault bool, Policy_Fields_SetImageAgeDays jsonb, Policy_Fields_Cve varchar, Policy_Fields_SetScanAgeDays jsonb, Policy_Fields_SetNoScanExists jsonb, Policy_Fields_Command varchar, Policy_Fields_Args varchar, Policy_Fields_Directory varchar, Policy_Fields_User varchar, Policy_Fields_SetPrivileged jsonb, Policy_Fields_DropCapabilities jsonb, Policy_Fields_AddCapabilities jsonb, Policy_Fields_SetReadOnlyRootFs jsonb, Policy_Fields_FixedBy varchar, Policy_Fields_SetWhitelist jsonb, Policy_Fields_ImageName_Registry varchar, Policy_Fields_ImageName_Remote varchar, Policy_Fields_ImageName_Tag varchar, Policy_Fields_LineRule_Instruction varchar, Policy_Fields_LineRule_Value varchar, Policy_Fields_Cvss_Op numeric, Policy_Fields_Cvss_Value numeric, Policy_Fields_Component_Name varchar, Policy_Fields_Component_Version varchar, Policy_Fields_Env_Key varchar, Policy_Fields_Env_Value varchar, Policy_Fields_Env_EnvVarSource numeric, Policy_Fields_VolumePolicy_Name varchar, Policy_Fields_VolumePolicy_Source varchar, Policy_Fields_VolumePolicy_Destination varchar, Policy_Fields_VolumePolicy_SetReadOnly jsonb, Policy_Fields_VolumePolicy_Type varchar, Policy_Fields_PortPolicy_Port numeric, Policy_Fields_PortPolicy_Protocol varchar, Policy_Fields_RequiredLabel_Key varchar, Policy_Fields_RequiredLabel_Value varchar, Policy_Fields_RequiredLabel_EnvVarSource numeric, Policy_Fields_RequiredAnnotation_Key varchar, Policy_Fields_RequiredAnnotation_Value varchar, Policy_Fields_RequiredAnnotation_EnvVarSource numeric, Policy_Fields_DisallowedAnnotation_Key varchar, Policy_Fields_DisallowedAnnotation_Value varchar, Policy_Fields_DisallowedAnnotation_EnvVarSource numeric, Policy_Fields_ContainerResourcePolicy_CpuResourceRequest_Op numeric, Policy_Fields_ContainerResourcePolicy_CpuResourceRequest_Value numeric, Policy_Fields_ContainerResourcePolicy_CpuResourceLimit_Op numeric, Policy_Fields_ContainerResourcePolicy_CpuResourceLimit_Value numeric, Policy_Fields_ContainerResourcePolicy_MemoryResourceRequest_Op numeric, Policy_Fields_ContainerResourcePolicy_MemoryResourceRequest_Value numeric, Policy_Fields_ContainerResourcePolicy_MemoryResourceLimit_Op numeric, Policy_Fields_ContainerResourcePolicy_MemoryResourceLimit_Value numeric, Policy_Fields_ProcessPolicy_Name varchar, Policy_Fields_ProcessPolicy_Args varchar, Policy_Fields_ProcessPolicy_Ancestor varchar, Policy_Fields_ProcessPolicy_Uid varchar, Policy_Fields_PortExposurePolicy_ExposureLevels jsonb, Policy_Fields_PermissionPolicy_PermissionLevel numeric, Policy_Fields_HostMountPolicy_SetReadOnly jsonb, Policy_Fields_RequiredImageLabel_Key varchar, Policy_Fields_RequiredImageLabel_Value varchar, Policy_Fields_RequiredImageLabel_EnvVarSource numeric, Policy_Fields_DisallowedImageLabel_Key varchar, Policy_Fields_DisallowedImageLabel_Value varchar, Policy_Fields_DisallowedImageLabel_EnvVarSource numeric, Policy_LastUpdated_Seconds numeric, Policy_LastUpdated_Nanos numeric, ProcessViolation_Message varchar, ProcessViolation_Processes jsonb, Enforcement_Action numeric, Enforcement_Message varchar, Time_Seconds numeric, Time_Nanos numeric, FirstOccurred_Seconds numeric, FirstOccurred_Nanos numeric, ResolvedAt_Seconds numeric, ResolvedAt_Nanos numeric, SnoozeTill_Seconds numeric, SnoozeTill_Nanos numeric)"

func CreateTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}

var marshaler = (&jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true})

func Upsert(alert *storage.Alert) error {
	var2, err := marshaler.MarshalToString(GetEntity())
	var3, err := marshaler.MarshalToString(GetViolations())
	var5, err := marshaler.MarshalToString(GetTags())
	var12, err := marshaler.MarshalToString(GetPolicy().GetCategories())
	var13, err := marshaler.MarshalToString(GetPolicy().GetLifecycleStages())
	var15, err := marshaler.MarshalToString(GetPolicy().GetWhitelists())
	var16, err := marshaler.MarshalToString(GetPolicy().GetExclusions())
	var17, err := marshaler.MarshalToString(GetPolicy().GetScope())
	var19, err := marshaler.MarshalToString(GetPolicy().GetEnforcementActions())
	var20, err := marshaler.MarshalToString(GetPolicy().GetNotifiers())
	var22, err := marshaler.MarshalToString(GetPolicy().GetPolicySections())
	var23, err := marshaler.MarshalToString(GetPolicy().GetMitreAttackVectors())
	var27, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetImageAgeDays())
	var29, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetScanAgeDays())
	var30, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetNoScanExists())
	var35, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetPrivileged())
	var36, err := marshaler.MarshalToString(GetPolicy().GetFields().GetDropCapabilities())
	var37, err := marshaler.MarshalToString(GetPolicy().GetFields().GetAddCapabilities())
	var38, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetReadOnlyRootFs())
	var40, err := marshaler.MarshalToString(GetPolicy().GetFields().GetSetWhitelist())
	var56, err := marshaler.MarshalToString(GetPolicy().GetFields().GetVolumePolicy().GetSetReadOnly())
	var81, err := marshaler.MarshalToString(GetPolicy().GetFields().GetPortExposurePolicy().GetExposureLevels())
	var83, err := marshaler.MarshalToString(GetPolicy().GetFields().GetHostMountPolicy().GetSetReadOnly())
	var93, err := marshaler.MarshalToString(GetProcessViolation().GetProcesses())
}
