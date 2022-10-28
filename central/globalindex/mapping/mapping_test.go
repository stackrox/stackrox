package mapping

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

var (
	postgresImageToVulnFieldMap = map[search.FieldLabel]*search.Field{
		search.AddCapabilities: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.add_capabilities",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.AdmissionControlStatus: {
			FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Cluster: {
			FieldPath: getDeploymentPrefix() + ".cluster_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getDeploymentPrefix() + ".cluster_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ClusterLabel: {
			FieldPath: getClusterPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterStatus: {
			FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CollectorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Component: {
			FieldPath: getImageComponentPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentCount: {
			FieldPath: getImagePrefix() + ".SetComponents.components",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ComponentID: {
			FieldPath: getImageComponentPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentLocation: {
			FieldPath: getImageComponentEdgePrefix() + ".location",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_COMPONENT_EDGE,
			Analyzer:  "",
		},
		search.ComponentPriority: {
			FieldPath: getImageComponentPrefix() + ".priority",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentRiskScore: {
			FieldPath: getImageComponentPrefix() + ".risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentSource: {
			FieldPath: getImageComponentPrefix() + ".source",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentTopCVSS: {
			FieldPath: getImageComponentPrefix() + ".SetTopCvss.top_cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.ComponentVersion: {
			FieldPath: getImageComponentPrefix() + ".version",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_COMPONENTS,
			Analyzer:  "",
		},
		search.CPUCoresLimit: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_limit",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.CPUCoresRequest: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_request",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Created: {
			FieldPath: getDeploymentPrefix() + ".created.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.CVE: {
			FieldPath: getImageCVEPrefix() + ".cve_base_info.cve",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVECount: {
			FieldPath: getImagePrefix() + ".SetCves.cves",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.CVECreatedTime: {
			FieldPath: getImageCVEPrefix() + ".cve_base_info.created_at.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEID: {
			FieldPath: getImageCVEPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEPublishedOn: {
			FieldPath: getImageCVEPrefix() + ".cve_base_info.published_on.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressed: {
			FieldPath: getImageCVEPrefix() + ".snoozed",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressExpiry: {
			FieldPath: getImageCVEPrefix() + ".snooze_expiry.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVSS: {
			FieldPath: getImageCVEPrefix() + ".cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.DeploymentAnnotation: {
			FieldPath: getDeploymentPrefix() + ".annotations",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentID: {
			FieldPath: getDeploymentPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentLabel: {
			FieldPath: getDeploymentPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentName: {
			FieldPath: getDeploymentPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentPriority: {
			FieldPath: getDeploymentPrefix() + ".priority",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentRiskScore: {
			FieldPath: getDeploymentPrefix() + ".risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentType: {
			FieldPath: getDeploymentPrefix() + ".type",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DockerfileInstructionKeyword: {
			FieldPath: getImagePrefix() + ".metadata.v1.layers.instruction",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.DockerfileInstructionValue: {
			FieldPath: getImagePrefix() + ".metadata.v1.layers.value",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.DropCapabilities: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.drop_capabilities",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentKey: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.key",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentValue: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.value",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentVarSrc: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.env_var_source",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposedNodePort: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.node_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposingService: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposingServicePort: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposureLevel: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.level",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExternalHostname: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_hostnames",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExternalIP: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_ips",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.FirstImageOccurrenceTimestamp: {
			FieldPath: getImageCVEEdgePrefix() + ".first_image_occurrence.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGE_VULN_EDGE,
			Analyzer:  "",
		},
		search.Fixable: {
			FieldPath: getComponentVulnEdgePrefix() + ".is_fixable",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
			Analyzer:  "",
		},
		search.FixableCVECount: {
			FieldPath: getImagePrefix() + ".SetFixable.fixable_cves",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.FixedBy: {
			FieldPath: getComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
			Analyzer:  "",
		},
		search.ImageCommand: {
			FieldPath: getImagePrefix() + ".metadata.v1.command",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageCreatedTime: {
			FieldPath: getImagePrefix() + ".metadata.v1.created.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageEntrypoint: {
			FieldPath: getImagePrefix() + ".metadata.v1.entrypoint",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageLabel: {
			FieldPath: getImagePrefix() + ".metadata.v1.labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageName: {
			FieldPath: getImagePrefix() + ".name.full_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "standard",
		},
		search.ImageOS: {
			FieldPath: getImagePrefix() + ".scan.operating_system",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImagePriority: {
			FieldPath: getImagePrefix() + ".priority",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImagePullSecret: {
			FieldPath: getDeploymentPrefix() + ".image_pull_secrets",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageRegistry: {
			FieldPath: getImagePrefix() + ".name.registry",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageRemote: {
			FieldPath: getImagePrefix() + ".name.remote",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageRiskScore: {
			FieldPath: getImagePrefix() + ".risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageScanTime: {
			FieldPath: getImagePrefix() + ".scan.scan_time.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageSHA: {
			FieldPath: getImagePrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageSignatureFetchedTime: {
			FieldPath: getImagePrefix() + ".signature.fetched.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageTag: {
			FieldPath: getImagePrefix() + ".name.tag",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageTopCVSS: {
			FieldPath: getImagePrefix() + ".SetTopCvss.top_cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageUser: {
			FieldPath: getImagePrefix() + ".metadata.v1.user",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageVolumes: {
			FieldPath: getImagePrefix() + ".metadata.v1.volumes",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImpactScore: {
			FieldPath: getImageCVEPrefix() + ".impact_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.LastContactTime: {
			FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.LastUpdatedTime: {
			FieldPath: getImagePrefix() + ".last_updated.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.MaxExposureLevel: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.MemoryLimit: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_limit",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.MemoryRequest: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_request",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Namespace: {
			FieldPath: getDeploymentPrefix() + ".namespace",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.NamespaceAnnotation: {
			FieldPath: getNamespacePrefix() + ".annotations",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_NAMESPACES,
			Analyzer:  "",
		},
		search.NamespaceID: {
			FieldPath: getDeploymentPrefix() + ".namespace_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.NamespaceLabel: {
			FieldPath: getNamespacePrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_NAMESPACES,
			Analyzer:  "",
		},
		search.OperatingSystem: {
			FieldPath: getImageCVEPrefix() + ".operating_system",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.OrchestratorComponent: {
			FieldPath: getDeploymentPrefix() + ".orchestrator_component",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.PodLabel: {
			FieldPath: getDeploymentPrefix() + ".pod_labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Port: {
			FieldPath: getDeploymentPrefix() + ".ports.container_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.PortProtocol: {
			FieldPath: getDeploymentPrefix() + ".ports.protocol",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Privileged: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.privileged",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ReadOnlyRootFilesystem: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.read_only_root_filesystem",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ScannerStatus: {
			FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.SecretName: {
			FieldPath: getDeploymentPrefix() + ".containers.secrets.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.SecretPath: {
			FieldPath: getDeploymentPrefix() + ".containers.secrets.path",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.SensorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ServiceAccountName: {
			FieldPath: getDeploymentPrefix() + ".service_account",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ServiceAccountPermissionLevel: {
			FieldPath: getDeploymentPrefix() + ".service_account_permission_level",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Severity: {
			FieldPath: getImageCVEPrefix() + ".severity",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULNERABILITIES,
			Analyzer:  "",
		},
		search.VolumeDestination: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.destination",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeName: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeReadonly: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.read_only",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeSource: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.source",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeType: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.type",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VulnerabilityState: {
			FieldPath: getImageCVEEdgePrefix() + ".state",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGE_VULN_EDGE,
			Analyzer:  "",
		},
	}
)

func getActiveComponentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "activecomponent"
	}
	return "active_component"
}

func getAlertPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "alert"
	}
	return "list_alert"
}

func getClusterPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "cluster"
	}
	return "cluster"
}

func getClusterCVEPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "clustercve"
	}
	return "c_v_e"
}

func getClusterCVEEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "clustercveedge"
	}
	return "cluster_c_v_e_edge"
}

func getComponentVulnEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "componentcveedge"
	}
	return "component_c_v_e_edge"
}

func getDeploymentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "deployment"
	}
	return "deployment"
}

func getImagePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "image"
	}
	return "image"
}

func getImageComponentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecomponent"
	}
	return "image_component"
}

func getImageComponentEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecomponentedge"
	}
	return "image_component_edge"
}

func getImageCVEPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecve"
	}
	return "c_v_e"
}

func getImageCVEEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecveedge"
	}
	return "image_c_v_e_edge"
}

func getNamespacePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "namespacemetadata"
	}
	return "namespace_metadata"
}

func getProcessIndicatorPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "processindicator"
	}
	return "process_indicator"
}

func TestActiveComponentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentID: {
			FieldPath: getActiveComponentPrefix() + ".component_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ACTIVE_COMPONENT,
			Analyzer:  "",
		},
		search.ContainerName: {
			FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.container_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ACTIVE_COMPONENT,
			Analyzer:  "",
		},
		search.DeploymentID: {
			FieldPath: getActiveComponentPrefix() + ".deployment_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ACTIVE_COMPONENT,
			Analyzer:  "",
		},
		search.ImageSHA: {
			FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.image_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ACTIVE_COMPONENT,
			Analyzer:  "",
		},
	}
	validateOptionsMap(t,
		v1.SearchCategory_ACTIVE_COMPONENT,
		expectedSearchFieldMap,
		nil,
		nil)
}

func TestAlertMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Category: {
			FieldPath: getAlertPrefix() + ".policy.categories",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.DeploymentID: {
			FieldPath: getAlertPrefix() + ".Entity.deployment.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.DeploymentName: {
			FieldPath: getAlertPrefix() + ".Entity.deployment.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Inactive: {
			FieldPath: getAlertPrefix() + ".Entity.deployment.inactive",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.PolicyID: {
			FieldPath: getAlertPrefix() + ".policy.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.PolicyName: {
			FieldPath: getAlertPrefix() + ".policy.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ResourceName: {
			FieldPath: getAlertPrefix() + ".Entity.resource.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Severity: {
			FieldPath: getAlertPrefix() + ".policy.severity",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ViolationState: {
			FieldPath: getAlertPrefix() + ".state",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ViolationTime: {
			FieldPath: getAlertPrefix() + ".time.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster: {
			FieldPath: getAlertPrefix() + ".common_entity_info.cluster_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getAlertPrefix() + ".common_entity_info.cluster_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Enforcement: {
			FieldPath: getAlertPrefix() + ".enforcement_action",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.LifecycleStage: {
			FieldPath: getAlertPrefix() + ".lifecycle_stage",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Namespace: {
			FieldPath: getAlertPrefix() + ".common_entity_info.namespace",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.NamespaceID: {
			FieldPath: getAlertPrefix() + ".common_entity_info.namespace_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ResourceType: {
			FieldPath: getAlertPrefix() + ".common_entity_info.resource_type",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.SORTPolicyName: {
			FieldPath: getAlertPrefix() + ".policy.developer_internal_fields.SORT_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "keyword",
		},
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster: {
			FieldPath: getAlertPrefix() + ".cluster_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getAlertPrefix() + ".cluster_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Enforcement: {
			FieldPath: getAlertPrefix() + ".enforcement.action",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.LifecycleStage: {
			FieldPath: getAlertPrefix() + ".lifecycle_stage",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.Namespace: {
			FieldPath: getAlertPrefix() + ".namespace",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.NamespaceID: {
			FieldPath: getAlertPrefix() + ".namespace_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.ResourceType: {
			FieldPath: getAlertPrefix() + ".Entity.resource.resource_type",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "",
		},
		search.SORTPolicyName: {
			FieldPath: getAlertPrefix() + ".policy.SORT_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "keyword",
		},
	}
	validateOptionsMap(t,
		v1.SearchCategory_ALERTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClusterVulnEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ClusterCVEFixable: {
			FieldPath: getClusterCVEEdgePrefix() + ".is_fixable",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
			Analyzer:  "",
		},
		search.ClusterCVEFixedBy: {
			FieldPath: getClusterCVEEdgePrefix() + ".HasFixedBy.fixed_by",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: {
			FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Cluster: {
			FieldPath: getClusterPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getClusterPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterLabel: {
			FieldPath: getClusterPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterStatus: {
			FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CollectorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CVE: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.cve",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVECreatedTime: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.created_at.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEID: {
			FieldPath: getClusterCVEPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEPublishedOn: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.published_on.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressed: {
			FieldPath: getClusterCVEPrefix() + ".snoozed",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressExpiry: {
			FieldPath: getClusterCVEPrefix() + ".snooze_expiry.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEType: {
			FieldPath: getClusterCVEPrefix() + ".type",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVSS: {
			FieldPath: getClusterCVEPrefix() + ".cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.ImpactScore: {
			FieldPath: getClusterCVEPrefix() + ".impact_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.LastContactTime: {
			FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ScannerStatus: {
			FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.SensorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Severity: {
			FieldPath: getClusterCVEPrefix() + ".severity",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
	}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTER_VULN_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClusterVulnerabilitiesMapping(t *testing.T) {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: {
			FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Cluster: {
			FieldPath: getClusterPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterCVEFixable: {
			FieldPath: getClusterCVEEdgePrefix() + ".is_fixable",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
			Analyzer:  "",
		},
		search.ClusterCVEFixedBy: {
			FieldPath: getClusterCVEEdgePrefix() + ".HasFixedBy.fixed_by",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getClusterPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterLabel: {
			FieldPath: getClusterPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterStatus: {
			FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CollectorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CVE: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.cve",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVECreatedTime: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.created_at.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEID: {
			FieldPath: getClusterCVEPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEPublishedOn: {
			FieldPath: getClusterCVEPrefix() + ".cve_base_info.published_on.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressed: {
			FieldPath: getClusterCVEPrefix() + ".snoozed",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVESuppressExpiry: {
			FieldPath: getClusterCVEPrefix() + ".snooze_expiry.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVEType: {
			FieldPath: getClusterCVEPrefix() + ".type",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.CVSS: {
			FieldPath: getClusterCVEPrefix() + ".cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.ImpactScore: {
			FieldPath: getClusterCVEPrefix() + ".impact_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
		search.LastContactTime: {
			FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ScannerStatus: {
			FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.SensorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Severity: {
			FieldPath: getClusterCVEPrefix() + ".severity",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
			Analyzer:  "",
		},
	}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTER_VULNERABILITIES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClustersMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: {
			FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.Cluster: {
			FieldPath: getClusterPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getClusterPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterLabel: {
			FieldPath: getClusterPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ClusterStatus: {
			FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.CollectorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.LastContactTime: {
			FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.ScannerStatus: {
			FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
		search.SensorStatus: {
			FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_CLUSTERS,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTERS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)

}

func TestComplianceControlMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Control: {
			FieldPath: "control.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
			Analyzer:  "",
		},
		search.ControlID: {
			FieldPath: "control.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
			Analyzer:  "",
		},
		search.ControlGroupID: {
			FieldPath: "control.group_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
			Analyzer:  "",
		},
		search.StandardID: {
			FieldPath: "control.standard_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_COMPLIANCE_CONTROL,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestComplianceStandardMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Standard: {
			FieldPath: "standard.metadata.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPLIANCE_STANDARD,
			Analyzer:  "",
		},
		search.StandardID: {
			FieldPath: "standard.metadata.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPLIANCE_STANDARD,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_COMPLIANCE_STANDARD,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestComponentVulnEdgeMapping(t *testing.T) {
	targetMap := GetEntityOptionsMap()[v1.SearchCategory_COMPONENT_VULN_EDGE]
	for k, v := range targetMap.Original() {
		fmt.Println(k)
		fmt.Println(v)
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Fixable: {
			FieldPath: getComponentVulnEdgePrefix() + ".is_fixable",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
			Analyzer:  "",
		},
		search.FixedBy: {
			FieldPath: getComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresImageToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_COMPONENT_VULN_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestDeploymentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AddCapabilities: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.add_capabilities",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Cluster: {
			FieldPath: getDeploymentPrefix() + ".cluster_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ClusterID: {
			FieldPath: getDeploymentPrefix() + ".cluster_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Component: {
			FieldPath: getImagePrefix() + ".scan.components.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ComponentCount: {
			FieldPath: getImagePrefix() + ".SetComponents.components",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ComponentRiskScore: {
			FieldPath: getImagePrefix() + ".scan.components.risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ComponentVersion: {
			FieldPath: getImagePrefix() + ".scan.components.version",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ContainerID: {
			FieldPath: getProcessIndicatorPrefix() + ".signal.container_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ContainerName: {
			FieldPath: getProcessIndicatorPrefix() + ".container_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.CPUCoresLimit: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_limit",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.CPUCoresRequest: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_request",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Created: {
			FieldPath: getDeploymentPrefix() + ".created.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.CVE: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.cve",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.CVECount: {
			FieldPath: getImagePrefix() + ".SetCves.cves",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.CVEPublishedOn: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.published_on.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.CVESuppressed: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.suppressed",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.CVSS: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.DeploymentAnnotation: {
			FieldPath: getDeploymentPrefix() + ".annotations",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentID: {
			FieldPath: getDeploymentPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentLabel: {
			FieldPath: getDeploymentPrefix() + ".labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentName: {
			FieldPath: getDeploymentPrefix() + ".name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentPriority: {
			FieldPath: getDeploymentPrefix() + ".priority",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentRiskScore: {
			FieldPath: getDeploymentPrefix() + ".risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DeploymentType: {
			FieldPath: getDeploymentPrefix() + ".type",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.DockerfileInstructionKeyword: {
			FieldPath: getImagePrefix() + ".metadata.v1.layers.instruction",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.DockerfileInstructionValue: {
			FieldPath: getImagePrefix() + ".metadata.v1.layers.value",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.DropCapabilities: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.drop_capabilities",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentKey: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.key",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentValue: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.value",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.EnvironmentVarSrc: {
			FieldPath: getDeploymentPrefix() + ".containers.config.env.env_var_source",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposedNodePort: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.node_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposingService: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposingServicePort: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExposureLevel: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.level",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExternalHostname: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_hostnames",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ExternalIP: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_ips",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.FixableCVECount: {
			FieldPath: getImagePrefix() + ".SetFixable.fixable_cves",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.FixedBy: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.SetFixedBy.fixed_by",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageCommand: {
			FieldPath: getImagePrefix() + ".metadata.v1.command",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageCreatedTime: {
			FieldPath: getImagePrefix() + ".metadata.v1.created.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageEntrypoint: {
			FieldPath: getImagePrefix() + ".metadata.v1.entrypoint",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageLabel: {
			FieldPath: getImagePrefix() + ".metadata.v1.labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageName: {
			FieldPath: getDeploymentPrefix() + ".containers.image.name.full_name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "standard",
		},
		search.ImageOS: {
			FieldPath: getImagePrefix() + ".scan.operating_system",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImagePriority: {
			FieldPath: getImagePrefix() + ".priority",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImagePullSecret: {
			FieldPath: getDeploymentPrefix() + ".image_pull_secrets",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageRegistry: {
			FieldPath: getDeploymentPrefix() + ".containers.image.name.registry",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageRemote: {
			FieldPath: getDeploymentPrefix() + ".containers.image.name.remote",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageRiskScore: {
			FieldPath: getImagePrefix() + ".risk_score",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageScanTime: {
			FieldPath: getImagePrefix() + ".scan.scan_time.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageSHA: {
			FieldPath: getDeploymentPrefix() + ".containers.image.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageSignatureFetchedTime: {
			FieldPath: getImagePrefix() + ".signature.fetched.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageTag: {
			FieldPath: getDeploymentPrefix() + ".containers.image.name.tag",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ImageTopCVSS: {
			FieldPath: getImagePrefix() + ".SetTopCvss.top_cvss",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageUser: {
			FieldPath: getImagePrefix() + ".metadata.v1.user",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.ImageVolumes: {
			FieldPath: getImagePrefix() + ".metadata.v1.volumes",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.LastUpdatedTime: {
			FieldPath: getImagePrefix() + ".last_updated.seconds",
			Type:      v1.SearchDataType_SEARCH_DATETIME,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
		search.MaxExposureLevel: {
			FieldPath: getDeploymentPrefix() + ".ports.exposure",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.MemoryLimit: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_limit",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.MemoryRequest: {
			FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_request",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Namespace: {
			FieldPath: getDeploymentPrefix() + ".namespace",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.NamespaceID: {
			FieldPath: getDeploymentPrefix() + ".namespace_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.OrchestratorComponent: {
			FieldPath: getDeploymentPrefix() + ".orchestrator_component",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.PodID: {
			FieldPath: getProcessIndicatorPrefix() + ".pod_id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.PodLabel: {
			FieldPath: getDeploymentPrefix() + ".pod_labels",
			Type:      v1.SearchDataType_SEARCH_MAP,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.PodUID: {
			FieldPath: getProcessIndicatorPrefix() + ".pod_uid",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    true,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.Port: {
			FieldPath: getDeploymentPrefix() + ".ports.container_port",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.PortProtocol: {
			FieldPath: getDeploymentPrefix() + ".ports.protocol",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.Privileged: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.privileged",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ProcessArguments: {
			FieldPath: getProcessIndicatorPrefix() + ".signal.args",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ProcessExecPath: {
			FieldPath: getProcessIndicatorPrefix() + ".signal.exec_file_path",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ProcessID: {
			FieldPath: getProcessIndicatorPrefix() + ".id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    true,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ProcessName: {
			FieldPath: getProcessIndicatorPrefix() + ".signal.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ProcessUID: {
			FieldPath: getProcessIndicatorPrefix() + ".signal.uid",
			Type:      v1.SearchDataType_SEARCH_NUMERIC,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_PROCESS_INDICATORS,
			Analyzer:  "",
		},
		search.ReadOnlyRootFilesystem: {
			FieldPath: getDeploymentPrefix() + ".containers.security_context.read_only_root_filesystem",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.SecretName: {
			FieldPath: getDeploymentPrefix() + ".containers.secrets.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.SecretPath: {
			FieldPath: getDeploymentPrefix() + ".containers.secrets.path",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ServiceAccountName: {
			FieldPath: getDeploymentPrefix() + ".service_account",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.ServiceAccountPermissionLevel: {
			FieldPath: getDeploymentPrefix() + ".service_account_permission_level",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeDestination: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.destination",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeName: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.name",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeReadonly: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.read_only",
			Type:      v1.SearchDataType_SEARCH_BOOL,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeSource: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.source",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VolumeType: {
			FieldPath: getDeploymentPrefix() + ".containers.volumes.type",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Hidden:    false,
			Category:  v1.SearchCategory_DEPLOYMENTS,
			Analyzer:  "",
		},
		search.VulnerabilityState: {
			FieldPath: getImagePrefix() + ".scan.components.vulns.state",
			Type:      v1.SearchDataType_SEARCH_ENUM,
			Store:     false,
			Hidden:    false,
			Category:  v1.SearchCategory_IMAGES,
			Analyzer:  "",
		},
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_DEPLOYMENTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func validateOptionsMap(
	t *testing.T,
	category v1.SearchCategory,
	expectedFieldMap map[search.FieldLabel]*search.Field,
	expectedLegacyAddFieldMap map[search.FieldLabel]*search.Field,
	expectedPostgresAddFieldMap map[search.FieldLabel]*search.Field) {
	// Extract OptionsMap registered in the global index mapping for the search category
	actualMap := GetEntityOptionsMap()[category]
	expectedLen := len(expectedFieldMap) + len(expectedLegacyAddFieldMap) + len(expectedPostgresAddFieldMap)
	expectedSearchFieldLabels := make([]search.FieldLabel, 0, expectedLen)
	for k := range expectedFieldMap {
		expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		for k := range expectedPostgresAddFieldMap {
			expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
		}
	} else {
		for k := range expectedLegacyAddFieldMap {
			expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
		}
	}
	originalMap := actualMap.Original()
	actualSearchFieldLabels := make([]search.FieldLabel, 0, len(originalMap))
	for k := range originalMap {
		actualSearchFieldLabels = append(actualSearchFieldLabels, k)
	}
	assert.ElementsMatch(t, expectedSearchFieldLabels, actualSearchFieldLabels)
	for k := range expectedFieldMap {
		field, found := actualMap.Get(k.String())
		assert.Equal(t, expectedFieldMap[k], field)
		assert.True(t, found)
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		for k := range expectedPostgresAddFieldMap {
			field, found := actualMap.Get(k.String())
			assert.Equal(t, expectedPostgresAddFieldMap[k], field)
			assert.True(t, found)
		}
	} else {
		for k := range expectedLegacyAddFieldMap {
			field, found := actualMap.Get(k.String())
			assert.Equal(t, expectedLegacyAddFieldMap[k], field)
			assert.True(t, found)
		}
	}
}
