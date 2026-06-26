//go:build sql_integration

package m225tom226

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// Frozen pre-PR#21423 GORM models. These reproduce the old index tags so
// GORM AutoMigrate creates the standalone _idx indexes that the migration drops.
// External FK references (Roles, Notifiers, BaseImageRepositories) are stripped.
// All real columns are preserved.

// --- auth_machine_to_machine_configs ---

type oldAuthMachineToMachineConfigs struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Issuer     string `gorm:"column:issuer;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

func (oldAuthMachineToMachineConfigs) TableName() string {
	return "auth_machine_to_machine_configs"
}

type oldAuthMachineToMachineConfigsMappings struct {
	AuthMachineToMachineConfigsID  string                         `gorm:"column:auth_machine_to_machine_configs_id;type:uuid;primaryKey"`
	Idx                            int                            `gorm:"column:idx;type:integer;primaryKey;index:authmachinetomachineconfigsmappings_idx,type:btree"`
	Role                           string                         `gorm:"column:role;type:varchar"`
	AuthMachineToMachineConfigsRef oldAuthMachineToMachineConfigs `gorm:"foreignKey:auth_machine_to_machine_configs_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldAuthMachineToMachineConfigsMappings) TableName() string {
	return "auth_machine_to_machine_configs_mappings"
}

// --- base_images ---

type oldBaseImages struct {
	ID                    string     `gorm:"column:id;type:uuid;primaryKey"`
	BaseImageRepositoryID string     `gorm:"column:baseimagerepositoryid;type:varchar"`
	Repository            string     `gorm:"column:repository;type:varchar"`
	Tag                   string     `gorm:"column:tag;type:varchar"`
	ManifestDigest        string     `gorm:"column:manifestdigest;type:varchar"`
	DiscoveredAt          *time.Time `gorm:"column:discoveredat;type:timestamp"`
	Active                bool       `gorm:"column:active;type:bool"`
	FirstLayerDigest      string     `gorm:"column:firstlayerdigest;type:varchar;index:baseimages_firstlayerdigest,type:btree"`
	Serialized            []byte     `gorm:"column:serialized;type:bytea"`
}

func (oldBaseImages) TableName() string { return "base_images" }

type oldBaseImagesLayers struct {
	BaseImagesID  string        `gorm:"column:base_images_id;type:uuid;primaryKey"`
	Idx           int           `gorm:"column:idx;type:integer;primaryKey;index:baseimageslayers_idx,type:btree"`
	LayerDigest   string        `gorm:"column:layerdigest;type:varchar;uniqueIndex:base_image_id_layer"`
	Index         int32         `gorm:"column:index;type:integer"`
	BaseImagesRef oldBaseImages `gorm:"foreignKey:base_images_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldBaseImagesLayers) TableName() string { return "base_images_layers" }

// --- collections ---

type oldCollections struct {
	ID            string `gorm:"column:id;type:varchar;primaryKey"`
	Name          string `gorm:"column:name;type:varchar;unique"`
	CreatedByName string `gorm:"column:createdby_name;type:varchar"`
	UpdatedByName string `gorm:"column:updatedby_name;type:varchar"`
	Serialized    []byte `gorm:"column:serialized;type:bytea"`
}

func (oldCollections) TableName() string { return "collections" }

type oldCollectionsEmbeddedCollections struct {
	CollectionsID       string         `gorm:"column:collections_id;type:varchar;primaryKey"`
	Idx                 int            `gorm:"column:idx;type:integer;primaryKey;index:collectionsembeddedcollections_idx,type:btree"`
	ID                  string         `gorm:"column:id;type:varchar"`
	CollectionsRef      oldCollections `gorm:"foreignKey:collections_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	CollectionsCycleRef oldCollections `gorm:"foreignKey:id;references:id;belongsTo;constraint:OnDelete:RESTRICT"`
}

func (oldCollectionsEmbeddedCollections) TableName() string {
	return "collections_embedded_collections"
}

// --- compliance_operator_profile_v2 ---

type oldComplianceOperatorProfileV2 struct {
	ID             string                                           `gorm:"column:id;type:varchar;primaryKey"`
	ProfileID      string                                           `gorm:"column:profileid;type:varchar"`
	Name           string                                           `gorm:"column:name;type:varchar"`
	ProfileVersion string                                           `gorm:"column:profileversion;type:varchar"`
	ProductType    string                                           `gorm:"column:producttype;type:varchar"`
	Standard       string                                           `gorm:"column:standard;type:varchar"`
	ClusterID      string                                           `gorm:"column:clusterid;type:uuid;index:complianceoperatorprofilev2_sac_filter,type:hash"`
	ProfileRefID   string                                           `gorm:"column:profilerefid;type:uuid"`
	OperatorKind   storage.ComplianceOperatorProfileV2_OperatorKind `gorm:"column:operatorkind;type:integer"`
	Serialized     []byte                                           `gorm:"column:serialized;type:bytea"`
}

func (oldComplianceOperatorProfileV2) TableName() string {
	return "compliance_operator_profile_v2"
}

type oldComplianceOperatorProfileV2Rules struct {
	ComplianceOperatorProfileV2ID  string                         `gorm:"column:compliance_operator_profile_v2_id;type:varchar;primaryKey"`
	Idx                            int                            `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorprofilev2rules_idx,type:btree"`
	RuleName                       string                         `gorm:"column:rulename;type:varchar"`
	ComplianceOperatorProfileV2Ref oldComplianceOperatorProfileV2 `gorm:"foreignKey:compliance_operator_profile_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorProfileV2Rules) TableName() string {
	return "compliance_operator_profile_v2_rules"
}

// --- compliance_operator_report_snapshot_v2 ---

type oldComplianceOperatorReportSnapshotV2 struct {
	ReportID                             string                                                    `gorm:"column:reportid;type:uuid;primaryKey"`
	ScanConfigurationID                  string                                                    `gorm:"column:scanconfigurationid;type:varchar"`
	Name                                 string                                                    `gorm:"column:name;type:varchar"`
	ReportStatusRunState                 storage.ComplianceOperatorReportStatus_RunState           `gorm:"column:reportstatus_runstate;type:integer"`
	ReportStatusStartedAt                *time.Time                                                `gorm:"column:reportstatus_startedat;type:timestamp"`
	ReportStatusCompletedAt              *time.Time                                                `gorm:"column:reportstatus_completedat;type:timestamp"`
	ReportStatusReportRequestType        storage.ComplianceOperatorReportStatus_RunMethod          `gorm:"column:reportstatus_reportrequesttype;type:integer"`
	ReportStatusReportNotificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod `gorm:"column:reportstatus_reportnotificationmethod;type:integer"`
	UserID                               string                                                    `gorm:"column:user_id;type:varchar"`
	UserName                             string                                                    `gorm:"column:user_name;type:varchar"`
	Serialized                           []byte                                                    `gorm:"column:serialized;type:bytea"`
}

func (oldComplianceOperatorReportSnapshotV2) TableName() string {
	return "compliance_operator_report_snapshot_v2"
}

type oldComplianceOperatorReportSnapshotV2Scans struct {
	ComplianceOperatorReportSnapshotV2ReportID string                                `gorm:"column:compliance_operator_report_snapshot_v2_reportid;type:uuid;primaryKey"`
	Idx                                        int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorreportsnapshotv2scans_idx,type:btree"`
	ScanRefID                                  string                                `gorm:"column:scanrefid;type:varchar"`
	LastStartedTime                            *time.Time                            `gorm:"column:laststartedtime;type:timestamp"`
	ComplianceOperatorReportSnapshotV2Ref      oldComplianceOperatorReportSnapshotV2 `gorm:"foreignKey:compliance_operator_report_snapshot_v2_reportid;references:reportid;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorReportSnapshotV2Scans) TableName() string {
	return "compliance_operator_report_snapshot_v2_scans"
}

// --- compliance_operator_rule_v2 ---

type oldComplianceOperatorRuleV2 struct {
	ID         string               `gorm:"column:id;type:varchar;primaryKey"`
	Name       string               `gorm:"column:name;type:varchar"`
	RuleType   string               `gorm:"column:ruletype;type:varchar"`
	Severity   storage.RuleSeverity `gorm:"column:severity;type:integer"`
	ClusterID  string               `gorm:"column:clusterid;type:uuid;index:complianceoperatorrulev2_sac_filter,type:hash"`
	RuleRefID  string               `gorm:"column:rulerefid;type:uuid"`
	Serialized []byte               `gorm:"column:serialized;type:bytea"`
}

func (oldComplianceOperatorRuleV2) TableName() string {
	return "compliance_operator_rule_v2"
}

type oldComplianceOperatorRuleV2Controls struct {
	ComplianceOperatorRuleV2ID  string                      `gorm:"column:compliance_operator_rule_v2_id;type:varchar;primaryKey"`
	Idx                         int                         `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorrulev2controls_idx,type:btree"`
	Standard                    string                      `gorm:"column:standard;type:varchar"`
	Control                     string                      `gorm:"column:control;type:varchar"`
	ComplianceOperatorRuleV2Ref oldComplianceOperatorRuleV2 `gorm:"foreignKey:compliance_operator_rule_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorRuleV2Controls) TableName() string {
	return "compliance_operator_rule_v2_controls"
}

// --- compliance_operator_scan_configuration_v2 (3 children) ---

type oldComplianceOperatorScanConfigurationV2 struct {
	ID             string `gorm:"column:id;type:uuid;primaryKey"`
	ScanConfigName string `gorm:"column:scanconfigname;type:varchar;unique"`
	ModifiedByName string `gorm:"column:modifiedby_name;type:varchar"`
	Serialized     []byte `gorm:"column:serialized;type:bytea"`
}

func (oldComplianceOperatorScanConfigurationV2) TableName() string {
	return "compliance_operator_scan_configuration_v2"
}

type oldComplianceOperatorScanConfigurationV2Profiles struct {
	ComplianceOperatorScanConfigurationV2ID  string                                   `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                      `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2profiles_idx,type:btree"`
	ProfileName                              string                                   `gorm:"column:profilename;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref oldComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorScanConfigurationV2Profiles) TableName() string {
	return "compliance_operator_scan_configuration_v2_profiles"
}

type oldComplianceOperatorScanConfigurationV2Clusters struct {
	ComplianceOperatorScanConfigurationV2ID  string                                   `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                      `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2clusters_idx,type:btree"`
	ClusterID                                string                                   `gorm:"column:clusterid;type:uuid;index:complianceoperatorscanconfigurationv2clusters_sac_filter,type:hash"`
	ComplianceOperatorScanConfigurationV2Ref oldComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorScanConfigurationV2Clusters) TableName() string {
	return "compliance_operator_scan_configuration_v2_clusters"
}

type oldComplianceOperatorScanConfigurationV2Notifiers struct {
	ComplianceOperatorScanConfigurationV2ID  string                                   `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                      `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2notifiers_idx,type:btree"`
	ID                                       string                                   `gorm:"column:id;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref oldComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldComplianceOperatorScanConfigurationV2Notifiers) TableName() string {
	return "compliance_operator_scan_configuration_v2_notifiers"
}

// --- deployments (2 children, 4 grandchildren) ---

type oldDeployments struct {
	ID                            string                  `gorm:"column:id;type:uuid;primaryKey"`
	Name                          string                  `gorm:"column:name;type:varchar"`
	Hash                          uint64                  `gorm:"column:hash;type:numeric"`
	Type                          string                  `gorm:"column:type;type:varchar"`
	Namespace                     string                  `gorm:"column:namespace;type:varchar;index:deployments_sac_filter,type:btree"`
	NamespaceID                   string                  `gorm:"column:namespaceid;type:uuid"`
	OrchestratorComponent         bool                    `gorm:"column:orchestratorcomponent;type:bool"`
	Labels                        map[string]string       `gorm:"column:labels;type:jsonb"`
	PodLabels                     map[string]string       `gorm:"column:podlabels;type:jsonb"`
	Created                       *time.Time              `gorm:"column:created;type:timestamp"`
	ClusterID                     string                  `gorm:"column:clusterid;type:uuid;index:deployments_sac_filter,type:btree"`
	ClusterName                   string                  `gorm:"column:clustername;type:varchar"`
	Annotations                   map[string]string       `gorm:"column:annotations;type:jsonb"`
	Priority                      int64                   `gorm:"column:priority;type:bigint"`
	ImagePullSecrets              *pq.StringArray         `gorm:"column:imagepullsecrets;type:text[]"`
	ServiceAccount                string                  `gorm:"column:serviceaccount;type:varchar"`
	ServiceAccountPermissionLevel storage.PermissionLevel `gorm:"column:serviceaccountpermissionlevel;type:integer"`
	RiskScore                     float32                 `gorm:"column:riskscore;type:numeric;index:deployments_riskscore,type:btree"`
	PlatformComponent             bool                    `gorm:"column:platformcomponent;type:bool"`
	Serialized                    []byte                  `gorm:"column:serialized;type:bytea"`
}

func (oldDeployments) TableName() string { return "deployments" }

type oldDeploymentsContainers struct {
	DeploymentsID                         string                `gorm:"column:deployments_id;type:uuid;primaryKey"`
	Idx                                   int                   `gorm:"column:idx;type:integer;primaryKey;index:deploymentscontainers_idx,type:btree"`
	ImageID                               string                `gorm:"column:image_id;type:varchar;index:deploymentscontainers_image_id,type:hash"`
	ImageNameRegistry                     string                `gorm:"column:image_name_registry;type:varchar"`
	ImageNameRemote                       string                `gorm:"column:image_name_remote;type:varchar"`
	ImageNameTag                          string                `gorm:"column:image_name_tag;type:varchar"`
	ImageNameFullName                     string                `gorm:"column:image_name_fullname;type:varchar"`
	ImageIDV2                             string                `gorm:"column:image_idv2;type:varchar;index:deploymentscontainers_image_idv2,type:btree"`
	SecurityContextPrivileged             bool                  `gorm:"column:securitycontext_privileged;type:bool"`
	SecurityContextDropCapabilities       *pq.StringArray       `gorm:"column:securitycontext_dropcapabilities;type:text[]"`
	SecurityContextAddCapabilities        *pq.StringArray       `gorm:"column:securitycontext_addcapabilities;type:text[]"`
	SecurityContextReadOnlyRootFilesystem bool                  `gorm:"column:securitycontext_readonlyrootfilesystem;type:bool"`
	ResourcesCPUCoresRequest              float32               `gorm:"column:resources_cpucoresrequest;type:numeric"`
	ResourcesCPUCoresLimit                float32               `gorm:"column:resources_cpucoreslimit;type:numeric"`
	ResourcesMemoryMbRequest              float32               `gorm:"column:resources_memorymbrequest;type:numeric"`
	ResourcesMemoryMbLimit                float32               `gorm:"column:resources_memorymblimit;type:numeric"`
	Type                                  storage.ContainerType `gorm:"column:type;type:integer"`
	DeploymentsRef                        oldDeployments        `gorm:"foreignKey:deployments_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsContainers) TableName() string { return "deployments_containers" }

type oldDeploymentsContainersEnvs struct {
	DeploymentsID            string                                                 `gorm:"column:deployments_id;type:uuid;primaryKey"`
	DeploymentsContainersIdx int                                                    `gorm:"column:deployments_containers_idx;type:integer;primaryKey"`
	Idx                      int                                                    `gorm:"column:idx;type:integer;primaryKey;index:deploymentscontainersenvs_idx,type:btree"`
	Key                      string                                                 `gorm:"column:key;type:varchar"`
	Value                    string                                                 `gorm:"column:value;type:varchar"`
	EnvVarSource             storage.ContainerConfig_EnvironmentConfig_EnvVarSource `gorm:"column:envvarsource;type:integer"`
	DeploymentsContainersRef oldDeploymentsContainers                               `gorm:"foreignKey:deployments_id,deployments_containers_idx;references:deployments_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsContainersEnvs) TableName() string { return "deployments_containers_envs" }

type oldDeploymentsContainersVolumes struct {
	DeploymentsID            string                   `gorm:"column:deployments_id;type:uuid;primaryKey"`
	DeploymentsContainersIdx int                      `gorm:"column:deployments_containers_idx;type:integer;primaryKey"`
	Idx                      int                      `gorm:"column:idx;type:integer;primaryKey;index:deploymentscontainersvolumes_idx,type:btree"`
	Name                     string                   `gorm:"column:name;type:varchar"`
	Source                   string                   `gorm:"column:source;type:varchar"`
	Destination              string                   `gorm:"column:destination;type:varchar"`
	ReadOnly                 bool                     `gorm:"column:readonly;type:bool"`
	Type                     string                   `gorm:"column:type;type:varchar"`
	DeploymentsContainersRef oldDeploymentsContainers `gorm:"foreignKey:deployments_id,deployments_containers_idx;references:deployments_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsContainersVolumes) TableName() string { return "deployments_containers_volumes" }

type oldDeploymentsContainersSecrets struct {
	DeploymentsID            string                   `gorm:"column:deployments_id;type:uuid;primaryKey"`
	DeploymentsContainersIdx int                      `gorm:"column:deployments_containers_idx;type:integer;primaryKey"`
	Idx                      int                      `gorm:"column:idx;type:integer;primaryKey;index:deploymentscontainerssecrets_idx,type:btree"`
	Name                     string                   `gorm:"column:name;type:varchar"`
	Path                     string                   `gorm:"column:path;type:varchar"`
	DeploymentsContainersRef oldDeploymentsContainers `gorm:"foreignKey:deployments_id,deployments_containers_idx;references:deployments_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsContainersSecrets) TableName() string { return "deployments_containers_secrets" }

type oldDeploymentsPorts struct {
	DeploymentsID  string                           `gorm:"column:deployments_id;type:uuid;primaryKey"`
	Idx            int                              `gorm:"column:idx;type:integer;primaryKey;index:deploymentsports_idx,type:btree"`
	ContainerPort  int32                            `gorm:"column:containerport;type:integer"`
	Protocol       string                           `gorm:"column:protocol;type:varchar"`
	Exposure       storage.PortConfig_ExposureLevel `gorm:"column:exposure;type:integer"`
	DeploymentsRef oldDeployments                   `gorm:"foreignKey:deployments_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsPorts) TableName() string { return "deployments_ports" }

type oldDeploymentsPortsExposureInfos struct {
	DeploymentsID       string                           `gorm:"column:deployments_id;type:uuid;primaryKey"`
	DeploymentsPortsIdx int                              `gorm:"column:deployments_ports_idx;type:integer;primaryKey"`
	Idx                 int                              `gorm:"column:idx;type:integer;primaryKey;index:deploymentsportsexposureinfos_idx,type:btree"`
	Level               storage.PortConfig_ExposureLevel `gorm:"column:level;type:integer"`
	ServiceName         string                           `gorm:"column:servicename;type:varchar"`
	ServicePort         int32                            `gorm:"column:serviceport;type:integer"`
	NodePort            int32                            `gorm:"column:nodeport;type:integer"`
	ExternalIps         *pq.StringArray                  `gorm:"column:externalips;type:text[]"`
	ExternalHostnames   *pq.StringArray                  `gorm:"column:externalhostnames;type:text[]"`
	DeploymentsPortsRef oldDeploymentsPorts              `gorm:"foreignKey:deployments_id,deployments_ports_idx;references:deployments_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldDeploymentsPortsExposureInfos) TableName() string {
	return "deployments_ports_exposure_infos"
}

// --- images ---

type oldImages struct {
	ID                   string            `gorm:"column:id;type:varchar;primaryKey"`
	NameRegistry         string            `gorm:"column:name_registry;type:varchar"`
	NameRemote           string            `gorm:"column:name_remote;type:varchar"`
	NameTag              string            `gorm:"column:name_tag;type:varchar"`
	NameFullName         string            `gorm:"column:name_fullname;type:varchar"`
	MetadataV1Created    *time.Time        `gorm:"column:metadata_v1_created;type:timestamp"`
	MetadataV1User       string            `gorm:"column:metadata_v1_user;type:varchar"`
	MetadataV1Command    *pq.StringArray   `gorm:"column:metadata_v1_command;type:text[]"`
	MetadataV1Entrypoint *pq.StringArray   `gorm:"column:metadata_v1_entrypoint;type:text[]"`
	MetadataV1Volumes    *pq.StringArray   `gorm:"column:metadata_v1_volumes;type:text[]"`
	MetadataV1Labels     map[string]string `gorm:"column:metadata_v1_labels;type:jsonb"`
	ScanScanTime         *time.Time        `gorm:"column:scan_scantime;type:timestamp"`
	ScanOperatingSystem  string            `gorm:"column:scan_operatingsystem;type:varchar"`
	SignatureFetched     *time.Time        `gorm:"column:signature_fetched;type:timestamp"`
	Components           int32             `gorm:"column:components;type:integer"`
	Cves                 int32             `gorm:"column:cves;type:integer"`
	FixableCves          int32             `gorm:"column:fixablecves;type:integer"`
	LastUpdated          *time.Time        `gorm:"column:lastupdated;type:timestamp"`
	Priority             int64             `gorm:"column:priority;type:bigint"`
	RiskScore            float32           `gorm:"column:riskscore;type:numeric"`
	TopCvss              float32           `gorm:"column:topcvss;type:numeric"`
	Serialized           []byte            `gorm:"column:serialized;type:bytea"`
}

func (oldImages) TableName() string { return "images" }

type oldImagesLayers struct {
	ImagesID    string    `gorm:"column:images_id;type:varchar;primaryKey"`
	Idx         int       `gorm:"column:idx;type:integer;primaryKey;index:imageslayers_idx,type:btree"`
	Instruction string    `gorm:"column:instruction;type:varchar"`
	Value       string    `gorm:"column:value;type:varchar"`
	ImagesRef   oldImages `gorm:"foreignKey:images_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldImagesLayers) TableName() string { return "images_layers" }

// --- images_v2 ---

type oldImagesV2 struct {
	ID                                string            `gorm:"column:id;type:varchar;primaryKey"`
	Digest                            string            `gorm:"column:digest;type:varchar"`
	NameRegistry                      string            `gorm:"column:name_registry;type:varchar"`
	NameRemote                        string            `gorm:"column:name_remote;type:varchar"`
	NameTag                           string            `gorm:"column:name_tag;type:varchar"`
	NameFullName                      string            `gorm:"column:name_fullname;type:varchar"`
	MetadataV1Created                 *time.Time        `gorm:"column:metadata_v1_created;type:timestamp"`
	MetadataV1User                    string            `gorm:"column:metadata_v1_user;type:varchar"`
	MetadataV1Command                 *pq.StringArray   `gorm:"column:metadata_v1_command;type:text[]"`
	MetadataV1Entrypoint              *pq.StringArray   `gorm:"column:metadata_v1_entrypoint;type:text[]"`
	MetadataV1Volumes                 *pq.StringArray   `gorm:"column:metadata_v1_volumes;type:text[]"`
	MetadataV1Labels                  map[string]string `gorm:"column:metadata_v1_labels;type:jsonb"`
	ScanScanTime                      *time.Time        `gorm:"column:scan_scantime;type:timestamp"`
	ScanOperatingSystem               string            `gorm:"column:scan_operatingsystem;type:varchar"`
	SignatureFetched                  *time.Time        `gorm:"column:signature_fetched;type:timestamp"`
	ScanStatsComponentCount           int32             `gorm:"column:scanstats_componentcount;type:integer"`
	ScanStatsCveCount                 int32             `gorm:"column:scanstats_cvecount;type:integer"`
	ScanStatsFixableCveCount          int32             `gorm:"column:scanstats_fixablecvecount;type:integer"`
	ScanStatsUnknownCveCount          int32             `gorm:"column:scanstats_unknowncvecount;type:integer"`
	ScanStatsFixableUnknownCveCount   int32             `gorm:"column:scanstats_fixableunknowncvecount;type:integer"`
	ScanStatsCriticalCveCount         int32             `gorm:"column:scanstats_criticalcvecount;type:integer"`
	ScanStatsFixableCriticalCveCount  int32             `gorm:"column:scanstats_fixablecriticalcvecount;type:integer"`
	ScanStatsImportantCveCount        int32             `gorm:"column:scanstats_importantcvecount;type:integer"`
	ScanStatsFixableImportantCveCount int32             `gorm:"column:scanstats_fixableimportantcvecount;type:integer"`
	ScanStatsModerateCveCount         int32             `gorm:"column:scanstats_moderatecvecount;type:integer"`
	ScanStatsFixableModerateCveCount  int32             `gorm:"column:scanstats_fixablemoderatecvecount;type:integer"`
	ScanStatsLowCveCount              int32             `gorm:"column:scanstats_lowcvecount;type:integer"`
	ScanStatsFixableLowCveCount       int32             `gorm:"column:scanstats_fixablelowcvecount;type:integer"`
	LastUpdated                       *time.Time        `gorm:"column:lastupdated;type:timestamp"`
	Priority                          int64             `gorm:"column:priority;type:bigint"`
	RiskScore                         float32           `gorm:"column:riskscore;type:numeric"`
	TopCvss                           float32           `gorm:"column:topcvss;type:numeric"`
	Serialized                        []byte            `gorm:"column:serialized;type:bytea"`
}

func (oldImagesV2) TableName() string { return "images_v2" }

type oldImagesV2Layers struct {
	ImagesV2ID  string      `gorm:"column:images_v2_id;type:varchar;primaryKey"`
	Idx         int         `gorm:"column:idx;type:integer;primaryKey;index:imagesv2layers_idx,type:btree"`
	Instruction string      `gorm:"column:instruction;type:varchar"`
	Value       string      `gorm:"column:value;type:varchar"`
	ImagesV2Ref oldImagesV2 `gorm:"foreignKey:images_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldImagesV2Layers) TableName() string { return "images_v2_layers" }

// --- nodes ---

type oldNodes struct {
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

func (oldNodes) TableName() string { return "nodes" }

type oldNodesTaints struct {
	NodesID     string              `gorm:"column:nodes_id;type:uuid;primaryKey"`
	Idx         int                 `gorm:"column:idx;type:integer;primaryKey;index:nodestaints_idx,type:btree"`
	Key         string              `gorm:"column:key;type:varchar"`
	Value       string              `gorm:"column:value;type:varchar"`
	TaintEffect storage.TaintEffect `gorm:"column:tainteffect;type:integer"`
	NodesRef    oldNodes            `gorm:"foreignKey:nodes_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldNodesTaints) TableName() string { return "nodes_taints" }

// --- pods ---

type oldPods struct {
	ID           string `gorm:"column:id;type:uuid;primaryKey"`
	Name         string `gorm:"column:name;type:varchar"`
	DeploymentID string `gorm:"column:deploymentid;type:uuid"`
	Namespace    string `gorm:"column:namespace;type:varchar;index:pods_sac_filter,type:btree"`
	ClusterID    string `gorm:"column:clusterid;type:uuid;index:pods_sac_filter,type:btree"`
	Serialized   []byte `gorm:"column:serialized;type:bytea"`
}

func (oldPods) TableName() string { return "pods" }

type oldPodsLiveInstances struct {
	PodsID      string  `gorm:"column:pods_id;type:uuid;primaryKey"`
	Idx         int     `gorm:"column:idx;type:integer;primaryKey;index:podsliveinstances_idx,type:btree"`
	ImageDigest string  `gorm:"column:imagedigest;type:varchar"`
	PodsRef     oldPods `gorm:"foreignKey:pods_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldPodsLiveInstances) TableName() string { return "pods_live_instances" }

// --- report_configurations ---

type oldReportConfigurations struct {
	ID                        string                                 `gorm:"column:id;type:varchar;primaryKey"`
	Name                      string                                 `gorm:"column:name;type:varchar"`
	Type                      storage.ReportConfiguration_ReportType `gorm:"column:type;type:integer"`
	ScopeID                   string                                 `gorm:"column:scopeid;type:varchar"`
	ResourceScopeCollectionID string                                 `gorm:"column:resourcescope_collectionid;type:varchar"`
	CreatorName               string                                 `gorm:"column:creator_name;type:varchar"`
	Serialized                []byte                                 `gorm:"column:serialized;type:bytea"`
}

func (oldReportConfigurations) TableName() string { return "report_configurations" }

type oldReportConfigurationsNotifiers struct {
	ReportConfigurationsID  string                  `gorm:"column:report_configurations_id;type:varchar;primaryKey"`
	Idx                     int                     `gorm:"column:idx;type:integer;primaryKey;index:reportconfigurationsnotifiers_idx,type:btree"`
	ID                      string                  `gorm:"column:id;type:varchar"`
	ReportConfigurationsRef oldReportConfigurations `gorm:"foreignKey:report_configurations_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldReportConfigurationsNotifiers) TableName() string {
	return "report_configurations_notifiers"
}

// --- role_bindings ---

type oldRoleBindings struct {
	ID          string            `gorm:"column:id;type:uuid;primaryKey"`
	Name        string            `gorm:"column:name;type:varchar"`
	Namespace   string            `gorm:"column:namespace;type:varchar;index:rolebindings_sac_filter,type:btree"`
	ClusterID   string            `gorm:"column:clusterid;type:uuid;index:rolebindings_sac_filter,type:btree"`
	ClusterName string            `gorm:"column:clustername;type:varchar"`
	ClusterRole bool              `gorm:"column:clusterrole;type:bool"`
	Labels      map[string]string `gorm:"column:labels;type:jsonb"`
	Annotations map[string]string `gorm:"column:annotations;type:jsonb"`
	RoleID      string            `gorm:"column:roleid;type:uuid"`
	Serialized  []byte            `gorm:"column:serialized;type:bytea"`
}

func (oldRoleBindings) TableName() string { return "role_bindings" }

type oldRoleBindingsSubjects struct {
	RoleBindingsID  string              `gorm:"column:role_bindings_id;type:uuid;primaryKey"`
	Idx             int                 `gorm:"column:idx;type:integer;primaryKey;index:rolebindingssubjects_idx,type:btree"`
	Kind            storage.SubjectKind `gorm:"column:kind;type:integer"`
	Name            string              `gorm:"column:name;type:varchar"`
	RoleBindingsRef oldRoleBindings     `gorm:"foreignKey:role_bindings_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldRoleBindingsSubjects) TableName() string { return "role_bindings_subjects" }

// --- secrets (child + grandchild) ---

type oldSecrets struct {
	ID          string     `gorm:"column:id;type:uuid;primaryKey"`
	Name        string     `gorm:"column:name;type:varchar"`
	ClusterID   string     `gorm:"column:clusterid;type:uuid;index:secrets_sac_filter,type:btree"`
	ClusterName string     `gorm:"column:clustername;type:varchar"`
	Namespace   string     `gorm:"column:namespace;type:varchar;index:secrets_sac_filter,type:btree"`
	CreatedAt   *time.Time `gorm:"column:createdat;type:timestamp"`
	Serialized  []byte     `gorm:"column:serialized;type:bytea"`
}

func (oldSecrets) TableName() string { return "secrets" }

type oldSecretsFiles struct {
	SecretsID   string             `gorm:"column:secrets_id;type:uuid;primaryKey"`
	Idx         int                `gorm:"column:idx;type:integer;primaryKey;index:secretsfiles_idx,type:btree"`
	Type        storage.SecretType `gorm:"column:type;type:integer"`
	CertEndDate *time.Time         `gorm:"column:cert_enddate;type:timestamp"`
	SecretsRef  oldSecrets         `gorm:"foreignKey:secrets_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldSecretsFiles) TableName() string { return "secrets_files" }

type oldSecretsFilesRegistries struct {
	SecretsID       string          `gorm:"column:secrets_id;type:uuid;primaryKey"`
	SecretsFilesIdx int             `gorm:"column:secrets_files_idx;type:integer;primaryKey"`
	Idx             int             `gorm:"column:idx;type:integer;primaryKey;index:secretsfilesregistries_idx,type:btree"`
	Name            string          `gorm:"column:name;type:varchar"`
	SecretsFilesRef oldSecretsFiles `gorm:"foreignKey:secrets_id,secrets_files_idx;references:secrets_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldSecretsFilesRegistries) TableName() string { return "secrets_files_registries" }

// --- vulnerability_requests (3 children) ---

type oldVulnerabilityRequests struct {
	ID                                string                           `gorm:"column:id;type:varchar;primaryKey"`
	Name                              string                           `gorm:"column:name;type:varchar;unique"`
	TargetState                       storage.VulnerabilityState       `gorm:"column:targetstate;type:integer"`
	Status                            storage.RequestStatus            `gorm:"column:status;type:integer"`
	Expired                           bool                             `gorm:"column:expired;type:bool"`
	RequestorName                     string                           `gorm:"column:requestor_name;type:varchar"`
	CreatedAt                         *time.Time                       `gorm:"column:createdat;type:timestamp"`
	LastUpdated                       *time.Time                       `gorm:"column:lastupdated;type:timestamp"`
	ScopeImageScopeRegistry           string                           `gorm:"column:scope_imagescope_registry;type:varchar"`
	ScopeImageScopeRemote             string                           `gorm:"column:scope_imagescope_remote;type:varchar"`
	ScopeImageScopeTag                string                           `gorm:"column:scope_imagescope_tag;type:varchar"`
	RequesterV2ID                     string                           `gorm:"column:requesterv2_id;type:varchar"`
	RequesterV2Name                   string                           `gorm:"column:requesterv2_name;type:varchar"`
	DeferralReqExpiryExpiresOn        *time.Time                       `gorm:"column:deferralreq_expiry_expireson;type:timestamp"`
	DeferralReqExpiryExpiresWhenFixed bool                             `gorm:"column:deferralreq_expiry_expireswhenfixed;type:bool"`
	DeferralReqExpiryExpiryType       storage.RequestExpiry_ExpiryType `gorm:"column:deferralreq_expiry_expirytype;type:integer"`
	CvesCves                          *pq.StringArray                  `gorm:"column:cves_cves;type:text[]"`
	DeferralUpdateCVEs                *pq.StringArray                  `gorm:"column:deferralupdate_cves;type:text[]"`
	FalsePositiveUpdateCVEs           *pq.StringArray                  `gorm:"column:falsepositiveupdate_cves;type:text[]"`
	Serialized                        []byte                           `gorm:"column:serialized;type:bytea"`
}

func (oldVulnerabilityRequests) TableName() string { return "vulnerability_requests" }

type oldVulnerabilityRequestsApprovers struct {
	VulnerabilityRequestsID  string                   `gorm:"column:vulnerability_requests_id;type:varchar;primaryKey"`
	Idx                      int                      `gorm:"column:idx;type:integer;primaryKey;index:vulnerabilityrequestsapprovers_idx,type:btree"`
	Name                     string                   `gorm:"column:name;type:varchar"`
	VulnerabilityRequestsRef oldVulnerabilityRequests `gorm:"foreignKey:vulnerability_requests_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldVulnerabilityRequestsApprovers) TableName() string {
	return "vulnerability_requests_approvers"
}

type oldVulnerabilityRequestsComments struct {
	VulnerabilityRequestsID  string                   `gorm:"column:vulnerability_requests_id;type:varchar;primaryKey"`
	Idx                      int                      `gorm:"column:idx;type:integer;primaryKey;index:vulnerabilityrequestscomments_idx,type:btree"`
	UserName                 string                   `gorm:"column:user_name;type:varchar"`
	VulnerabilityRequestsRef oldVulnerabilityRequests `gorm:"foreignKey:vulnerability_requests_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldVulnerabilityRequestsComments) TableName() string {
	return "vulnerability_requests_comments"
}

type oldVulnerabilityRequestsApproversV2 struct {
	VulnerabilityRequestsID  string                   `gorm:"column:vulnerability_requests_id;type:varchar;primaryKey"`
	Idx                      int                      `gorm:"column:idx;type:integer;primaryKey;index:vulnerabilityrequestsapproversv2_idx,type:btree"`
	ID                       string                   `gorm:"column:id;type:varchar"`
	Name                     string                   `gorm:"column:name;type:varchar"`
	VulnerabilityRequestsRef oldVulnerabilityRequests `gorm:"foreignKey:vulnerability_requests_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

func (oldVulnerabilityRequestsApproversV2) TableName() string {
	return "vulnerability_requests_approvers_v2"
}

// --- CreateStmts for all 16 schema groups ---

var oldCreateStmts = []*postgres.CreateStmts{
	// auth_machine_to_machine_configs
	{
		GormModel: (*oldAuthMachineToMachineConfigs)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldAuthMachineToMachineConfigsMappings)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// base_images
	{
		GormModel: (*oldBaseImages)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldBaseImagesLayers)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// collections
	{
		GormModel: (*oldCollections)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldCollectionsEmbeddedCollections)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// compliance_operator_profile_v2
	{
		GormModel: (*oldComplianceOperatorProfileV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldComplianceOperatorProfileV2Rules)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// compliance_operator_report_snapshot_v2
	{
		GormModel: (*oldComplianceOperatorReportSnapshotV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldComplianceOperatorReportSnapshotV2Scans)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// compliance_operator_rule_v2
	{
		GormModel: (*oldComplianceOperatorRuleV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldComplianceOperatorRuleV2Controls)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// compliance_operator_scan_configuration_v2 (3 children)
	{
		GormModel: (*oldComplianceOperatorScanConfigurationV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldComplianceOperatorScanConfigurationV2Profiles)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*oldComplianceOperatorScanConfigurationV2Clusters)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*oldComplianceOperatorScanConfigurationV2Notifiers)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// deployments (2 children, 4 grandchildren)
	{
		GormModel: (*oldDeployments)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*oldDeploymentsContainers)(nil),
				Children: []*postgres.CreateStmts{
					{GormModel: (*oldDeploymentsContainersEnvs)(nil), Children: []*postgres.CreateStmts{}},
					{GormModel: (*oldDeploymentsContainersVolumes)(nil), Children: []*postgres.CreateStmts{}},
					{GormModel: (*oldDeploymentsContainersSecrets)(nil), Children: []*postgres.CreateStmts{}},
				},
			},
			{
				GormModel: (*oldDeploymentsPorts)(nil),
				Children: []*postgres.CreateStmts{
					{GormModel: (*oldDeploymentsPortsExposureInfos)(nil), Children: []*postgres.CreateStmts{}},
				},
			},
		},
	},
	// images
	{
		GormModel: (*oldImages)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldImagesLayers)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// images_v2
	{
		GormModel: (*oldImagesV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldImagesV2Layers)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// nodes
	{
		GormModel: (*oldNodes)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldNodesTaints)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// pods
	{
		GormModel: (*oldPods)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldPodsLiveInstances)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// report_configurations
	{
		GormModel: (*oldReportConfigurations)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldReportConfigurationsNotifiers)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// role_bindings
	{
		GormModel: (*oldRoleBindings)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldRoleBindingsSubjects)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
	// secrets (child + grandchild)
	{
		GormModel: (*oldSecrets)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*oldSecretsFiles)(nil),
				Children: []*postgres.CreateStmts{
					{GormModel: (*oldSecretsFilesRegistries)(nil), Children: []*postgres.CreateStmts{}},
				},
			},
		},
	},
	// vulnerability_requests (3 children)
	{
		GormModel: (*oldVulnerabilityRequests)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*oldVulnerabilityRequestsApprovers)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*oldVulnerabilityRequestsComments)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*oldVulnerabilityRequestsApproversV2)(nil), Children: []*postgres.CreateStmts{}},
		},
	},
}

// --- test suite ---

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	dbs := s.dbs()
	for _, stmt := range oldCreateStmts {
		pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, stmt)
	}
}

func (s *migrationTestSuite) dbs() *types.Databases {
	return &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
}

// indexExists checks whether an index with the given name exists in the public schema.
func (s *migrationTestSuite) indexExists(name string) bool {
	var n int
	err := s.db.DB.QueryRow(s.ctx,
		"SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1", name).Scan(&n)
	if err == pgx.ErrNoRows {
		return false
	}
	s.Require().NoError(err)
	return true
}

func (s *migrationTestSuite) TestMigration() {
	dbs := s.dbs()

	// Verify every index we intend to drop actually exists.
	for _, name := range indexesToDrop {
		s.Require().True(s.indexExists(name), "index %s should exist before migration", name)
	}

	// Run the migration.
	s.Require().NoError(migration.Run(dbs))

	// Verify every index is gone.
	for _, name := range indexesToDrop {
		s.Require().False(s.indexExists(name), "index %s should be dropped after migration", name)
	}

	// Run again to verify idempotency: DROP INDEX IF EXISTS should not error.
	s.Require().NoError(migration.Run(dbs))
}
