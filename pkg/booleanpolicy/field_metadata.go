package booleanpolicy

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/querybuilders"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
)

var (
	fieldsToQB     = make(map[string]func(*validateConfiguration) *metadataAndQB)
	errNoSuchField = errors.New("no such field")
)

type option int

const (
	negationForbidden option = iota
	operatorsForbidden
)

type metadataAndQB struct {
	operatorsForbidden bool
	negationForbidden  bool
	qb                 querybuilders.QueryBuilder
	valueRegex         *regexp.Regexp
	contextFields      violationmessages.ContextQueryFields
	eventSourceContext []storage.EventSource
}

func isApplicableToEventSource(m *metadataAndQB, source storage.EventSource) bool {
	for _, ec := range m.eventSourceContext {
		if ec == source {
			return true
		}
	}
	return false
}

func findFieldMetadata(fieldName string, config *validateConfiguration) (*metadataAndQB, error) {
	f := fieldsToQB[fieldName]
	if f == nil {
		return nil, errNoSuchField
	}
	return f(config), nil
}

func newFieldMetadata(qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, valueRegex *regexp.Regexp, source []storage.EventSource, options ...option) *metadataAndQB {
	m := &metadataAndQB{
		qb:                 qb,
		contextFields:      contextFields,
		valueRegex:         valueRegex,
		eventSourceContext: source,
	}
	for _, o := range options {
		switch o {
		case negationForbidden:
			m.negationForbidden = true
		case operatorsForbidden:
			m.operatorsForbidden = true
		}
	}

	return m
}

func ensureFieldIsUnique(fieldName string) {
	if fieldsToQB[fieldName] != nil {
		panic(fmt.Sprintf("found duplicate metadata for field %s", fieldName))
	}
}

func registerFieldMetadata(fieldName string, qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, valueRegex *regexp.Regexp, source []storage.EventSource, options ...option) {
	ensureFieldIsUnique(fieldName)

	m := newFieldMetadata(qb, contextFields, valueRegex, source, options...)
	fieldsToQB[fieldName] = func(*validateConfiguration) *metadataAndQB {
		return m
	}
}

func registerFieldMetadataConditionally(
	fieldName string,
	qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields,
	conditionalRegexp func(*validateConfiguration) *regexp.Regexp,
	source []storage.EventSource, options ...option) {
	ensureFieldIsUnique(fieldName)
	fieldsToQB[fieldName] = func(configuration *validateConfiguration) *metadataAndQB {
		return newFieldMetadata(qb, contextFields, conditionalRegexp(configuration), source, options...)
	}
}

//TODO: ROX-7357: Fix event source context for existing runtime fields (except audit log fields) to deployment event source
func init() {
	registerFieldMetadata(fieldnames.AddCaps, querybuilders.ForFieldLabelExact(search.AddCapabilities), violationmessages.ContainerContextFields, capabilitiesValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.AppArmorProfile, querybuilders.ForFieldLabelRegex(search.AppArmorProfile), violationmessages.ContainerContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.CVE, querybuilders.ForCVE(), violationmessages.VulnContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.CVSS, querybuilders.ForCVSS(), violationmessages.VulnContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerCPULimit, querybuilders.ForFieldLabel(search.CPUCoresLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerCPURequest, querybuilders.ForFieldLabel(search.CPUCoresRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerMemLimit, querybuilders.ForFieldLabel(search.MemoryLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerMemRequest, querybuilders.ForFieldLabel(search.MemoryRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerName, querybuilders.ForFieldLabelRegex(search.ContainerName), violationmessages.ContainerContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.DisallowedAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.MapShouldContain), nil, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.DisallowedImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldContain), violationmessages.ImageContextFields, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.DockerfileLine, querybuilders.ForCompound(augmentedobjs.DockerfileLineCustomTag, 2), violationmessages.ImageContextFields, dockerfileLineValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.DropCaps, querybuilders.ForDropCaps(), violationmessages.ContainerContextFields, capabilitiesValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadataConditionally(
		fieldnames.EnvironmentVariable,
		querybuilders.ForCompound(augmentedobjs.EnvironmentVarCustomTag, 3), violationmessages.EnvVarContextFields, func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.validateEnvVarSourceRestrictions {
				return environmentVariableWithSourceStrictRegex
			}

			return environmentVariableWithSourceRegex
		}, []storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		negationForbidden,
	)

	registerFieldMetadata(fieldnames.FixedBy, querybuilders.ForFixedBy(), violationmessages.VulnContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.HostIPC, querybuilders.ForFieldLabel(search.HostIPC), nil, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.HostNetwork, querybuilders.ForFieldLabel(search.HostNetwork), nil, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.HostPID, querybuilders.ForFieldLabel(search.HostPID), nil, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageAge, querybuilders.ForDays(search.ImageCreatedTime), violationmessages.ImageContextFields, integerValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageComponent, querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2), violationmessages.ImageContextFields, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.ImageOS, querybuilders.ForFieldLabel(search.ImageOS), violationmessages.ImageContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ImageRegistry, querybuilders.ForFieldLabelRegex(search.ImageRegistry), violationmessages.ImageContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ImageRemote, querybuilders.ForFieldLabelRegex(search.ImageRemote), violationmessages.ImageContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ImageScanAge, querybuilders.ForDays(search.ImageScanTime), violationmessages.ImageContextFields, integerValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageTag, querybuilders.ForFieldLabelRegex(search.ImageTag), violationmessages.ImageContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ImageUser, querybuilders.ForFieldLabelRegex(search.ImageUser), violationmessages.ImageContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.MinimumRBACPermissions, querybuilders.ForK8sRBAC(), nil, rbacPermissionValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, operatorsForbidden)
	registerFieldMetadata(fieldnames.MountPropagation, querybuilders.ForFieldLabel(search.MountPropagation), violationmessages.VolumeContextFields, mountPropagationValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.Namespace, querybuilders.ForFieldLabelRegex(search.Namespace), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ExposedNodePort, querybuilders.ForFieldLabel(search.ExposedNodePort), nil, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ExposedPort, querybuilders.ForFieldLabel(search.Port), violationmessages.PortContextFields, comparatorDecimalValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.PortExposure, querybuilders.ForFieldLabel(search.ExposureLevel), violationmessages.PortContextFields, portExposureValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.PrivilegedContainer, querybuilders.ForFieldLabel(search.Privileged), violationmessages.ContainerContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ProcessAncestor, querybuilders.ForFieldLabelRegex(search.ProcessAncestor), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ProcessArguments, querybuilders.ForFieldLabelRegex(search.ProcessArguments), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ProcessName, querybuilders.ForFieldLabelRegex(search.ProcessName), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ProcessUID, querybuilders.ForFieldLabel(search.ProcessUID), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ExposedPortProtocol, querybuilders.ForFieldLabelUpper(search.PortProtocol), violationmessages.PortContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.ReadOnlyRootFS, querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem), violationmessages.ContainerContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.RequiredAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.MapShouldNotContain), nil, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldNotContain), violationmessages.ImageContextFields, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredLabel, querybuilders.ForFieldLabelMap(search.Label, query.MapShouldNotContain), nil, keyValueValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.SeccompProfileType, querybuilders.ForFieldLabel(search.SeccompProfileType), violationmessages.ContainerContextFields, seccompProfileTypeValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, operatorsForbidden)
	registerFieldMetadata(fieldnames.ServiceAccount, querybuilders.ForFieldLabelRegex(search.ServiceAccountName), nil, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.Severity, querybuilders.ForSeverity(), violationmessages.VulnContextFields, severityValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden)
	registerFieldMetadata(fieldnames.UnscannedImage, querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag), violationmessages.ImageContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.VolumeDestination, querybuilders.ForFieldLabelRegex(search.VolumeDestination), violationmessages.VolumeContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.VolumeName, querybuilders.ForFieldLabelRegex(search.VolumeName), violationmessages.VolumeContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.VolumeSource, querybuilders.ForFieldLabelRegex(search.VolumeSource), violationmessages.VolumeContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.VolumeType, querybuilders.ForFieldLabelRegex(search.VolumeType), violationmessages.VolumeContextFields, stringValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE})
	registerFieldMetadata(fieldnames.UnexpectedNetworkFlowDetected, querybuilders.ForFieldLabel(augmentedobjs.NotInNetworkBaselineCustomTag), nil, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.UnexpectedProcessExecuted, querybuilders.ForFieldLabel(augmentedobjs.NotInProcessBaselineCustomTag), violationmessages.ProcessBaselineContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableHostMount, querybuilders.ForWriteableHostMount(), violationmessages.VolumeContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableMountedVolume, querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true), violationmessages.VolumeContextFields, booleanValueRegex, []storage.EventSource{storage.EventSource_NOT_APPLICABLE}, negationForbidden, operatorsForbidden)

	registerFieldMetadataConditionally(fieldnames.KubeAPIVerb,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesAPIVerbCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			if features.K8sAuditLogDetection.Enabled() && c != nil && c.sourceIsAuditLogEvents {
				return auditEventAPIVerbValueRegex
			}
			return kubernetesAPIVerbValueRegex
		}, []storage.EventSource{storage.EventSource_NOT_APPLICABLE, storage.EventSource_AUDIT_LOG_EVENT}, negationForbidden,
	) //removed operatorForbidden even from adm controller policies

	registerFieldMetadataConditionally(fieldnames.KubeResource,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			if features.K8sAuditLogDetection.Enabled() && c != nil && c.sourceIsAuditLogEvents {
				return auditEventResourceValueRegex
			}
			return kubernetesResourceValueRegex
		}, []storage.EventSource{storage.EventSource_NOT_APPLICABLE, storage.EventSource_AUDIT_LOG_EVENT}, negationForbidden,
	)

	if features.K8sAuditLogDetection.Enabled() {
		registerFieldMetadata(
			fieldnames.KubeResourceName,
			querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceNameCustomTag), nil, kubernetesNameRegex,
			[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT}, negationForbidden,
		)

		registerFieldMetadata(
			fieldnames.KubeUserName,
			querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserNameCustomTag), nil,
			kubernetesNameRegex, []storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT}, negationForbidden,
		)
		registerFieldMetadata(
			fieldnames.KubeUserGroups,
			querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserGroupsCustomTag), nil,
			kubernetesNameRegex, []storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		)
		registerFieldMetadata(
			fieldnames.SourceIPAddress,
			querybuilders.ForFieldLabel(augmentedobjs.KubernetesSourceIPAddressCustomTag), nil,
			ipAddressValueRegex, []storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		)
		registerFieldMetadata(
			fieldnames.UserAgent,
			querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserAgentCustomTag), nil,
			stringValueRegex, []storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		)
	}
}
