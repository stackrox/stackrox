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
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	fieldMetadataInstance     FieldMetadata
	fieldMetadataInstanceInit sync.Once
	errNoSuchField            = errors.New("no such field")
)

// FieldMetadata contains the policy criteria fields and their metadata
type FieldMetadata struct {
	fieldsToQB map[string]*metadataAndQB
}

// FieldMetadataSingleton is a singleton which contains metadata about each policy criteria field
func FieldMetadataSingleton() *FieldMetadata {
	fieldMetadataInstanceInit.Do(func() {
		fieldMetadataInstance = initializeFieldMetadata()
	})
	return &fieldMetadataInstance
}

type option int

const (
	negationForbidden option = iota
	operatorsForbidden
)

// RuntimeFieldType is the type of a runtime policy criteria field
type RuntimeFieldType string

const (
	// AuditLogEvent for a audit log based runtime event
	AuditLogEvent RuntimeFieldType = "auditLogEvent"
	// Process for a process based runtime event
	Process = "process"
	// NetworkFlow for a network flow based runtime event
	NetworkFlow = "networkFlow"
	// KubeEvent for an admission controller based runtime event
	KubeEvent = "kubeEvent"
)

type metadataAndQB struct {
	operatorsForbidden bool
	negationForbidden  bool
	qb                 querybuilders.QueryBuilder
	valueRegex         func(*validateConfiguration) *regexp.Regexp
	contextFields      violationmessages.ContextQueryFields
	eventSourceContext []storage.EventSource
	fieldTypes         []RuntimeFieldType
}

func (f *FieldMetadata) findField(fieldName string) (*metadataAndQB, error) {
	field := f.fieldsToQB[fieldName]
	if field == nil {
		return nil, errNoSuchField
	}
	return field, nil
}

// FieldIsOfType returns true if the specified field is of the specified type
func (f *FieldMetadata) FieldIsOfType(fieldName string, expectedType RuntimeFieldType) bool {
	field := f.fieldsToQB[fieldName]
	if field == nil {
		log.Warnf("policy field %s not found", fieldName)
		return false
	}
	for _, fieldType := range field.fieldTypes {
		if fieldType == expectedType {
			return true
		}
	}
	return false
}

// IsDeploymentEventField returns true if the field is an deployment event field
func (f *FieldMetadata) IsDeploymentEventField(fieldName string) bool {
	return f.FieldIsOfType(fieldName, Process) || f.FieldIsOfType(fieldName, NetworkFlow) ||
		f.FieldIsOfType(fieldName, KubeEvent)
}

// IsAuditLogEventField returns true if the field is an audit log field
func (f *FieldMetadata) IsAuditLogEventField(fieldName string) bool {
	return f.FieldIsOfType(fieldName, AuditLogEvent)
}

// findFieldMetadata searches for a policy criteria field by name and returns the field metadata
func (f *FieldMetadata) findFieldMetadata(fieldName string, _ *validateConfiguration) (*metadataAndQB, error) {
	field := f.fieldsToQB[fieldName]
	if field == nil {
		return nil, errNoSuchField
	}
	return field, nil
}

func newFieldMetadata(qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields,
	valueRegex func(configuration *validateConfiguration) *regexp.Regexp, source []storage.EventSource,
	fieldTypes []RuntimeFieldType, options ...option) *metadataAndQB {
	m := &metadataAndQB{
		qb:                 qb,
		contextFields:      contextFields,
		valueRegex:         valueRegex,
		eventSourceContext: source,
		fieldTypes:         fieldTypes,
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

func (f *FieldMetadata) ensureFieldIsUnique(fieldName string) {
	if f.fieldsToQB[fieldName] != nil {
		panic(fmt.Sprintf("found duplicate metadata for field %s", fieldName))
	}
}

func (f *FieldMetadata) registerFieldMetadata(fieldName string, qb querybuilders.QueryBuilder,
	contextFields violationmessages.ContextQueryFields,
	valueRegex func(configuration *validateConfiguration) *regexp.Regexp,
	source []storage.EventSource, fieldTypes []RuntimeFieldType, options ...option) {
	f.ensureFieldIsUnique(fieldName)

	m := newFieldMetadata(qb, contextFields, valueRegex, source, fieldTypes, options...)
	f.fieldsToQB[fieldName] = m
}

func (f *FieldMetadata) registerFieldMetadataConditionally(
	fieldName string,
	qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields,
	conditionalRegexp func(*validateConfiguration) *regexp.Regexp,
	source []storage.EventSource, fieldTypes []RuntimeFieldType, options ...option) {
	f.ensureFieldIsUnique(fieldName)
	f.fieldsToQB[fieldName] = newFieldMetadata(qb, contextFields, conditionalRegexp, source, fieldTypes, options...)

}

func initializeFieldMetadata() FieldMetadata {
	f := FieldMetadata{
		fieldsToQB: make(map[string]*metadataAndQB),
	}

	f.registerFieldMetadata(fieldnames.AddCaps,
		querybuilders.ForAddCaps(),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return addCapabilitiesValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden,
	)

	f.registerFieldMetadata(fieldnames.AllowPrivilegeEscalation,
		querybuilders.ForFieldLabel(search.AllowPrivilegeEscalation),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.AppArmorProfile,
		querybuilders.ForFieldLabelRegex(search.AppArmorProfile),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.AutomountServiceAccountToken,
		querybuilders.ForFieldLabel(search.AutomountServiceAccountToken),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden,
		operatorsForbidden)

	f.registerFieldMetadata(fieldnames.CVE,
		querybuilders.ForCVE(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.CVSS, querybuilders.ForCVSS(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden)

	f.registerFieldMetadata(fieldnames.ContainerCPULimit,
		querybuilders.ForFieldLabel(search.CPUCoresLimit),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ContainerCPURequest,
		querybuilders.ForFieldLabel(search.CPUCoresRequest),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ContainerMemLimit,
		querybuilders.ForFieldLabel(search.MemoryLimit),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ContainerMemRequest,
		querybuilders.ForFieldLabel(search.MemoryRequest),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ContainerName,
		querybuilders.ForFieldLabelRegex(search.ContainerName),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.DisallowedAnnotation,
		querybuilders.ForFieldLabelMap(search.DeploymentAnnotation, query.MapShouldContain),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.DisallowedImageLabel,
		querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldContain),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.DockerfileLine,
		querybuilders.ForCompound(augmentedobjs.DockerfileLineCustomTag, 2),
		violationmessages.ImageContextFields,
		func(c *validateConfiguration) *regexp.Regexp {
			if c.disallowFromInDockerfileLine {
				return dockerfileLineValueRegexNoFrom
			}
			return dockerfileLineValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.DropCaps,
		querybuilders.ForDropCaps(),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return dropCapabilitiesValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataConditionally(
		fieldnames.EnvironmentVariable,
		querybuilders.ForCompound(augmentedobjs.EnvironmentVarCustomTag, 3),
		violationmessages.EnvVarContextFields,
		func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.validateEnvVarSourceRestrictions {
				return environmentVariableWithSourceStrictRegex
			}

			return environmentVariableWithSourceRegex
		}, []storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden,
	)

	f.registerFieldMetadata(fieldnames.FixedBy,
		querybuilders.ForFixedBy(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.HostIPC,
		querybuilders.ForFieldLabel(search.HostIPC),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.HostNetwork,
		querybuilders.ForFieldLabel(search.HostNetwork),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.HostPID,
		querybuilders.ForFieldLabel(search.HostPID), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.ImageAge,
		querybuilders.ForDays(search.ImageCreatedTime),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.ImageComponent,
		querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ImageOS,
		querybuilders.ForFieldLabelRegex(search.ImageOS),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ImageRegistry,
		querybuilders.ForFieldLabelRegex(search.ImageRegistry),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ImageRemote,
		querybuilders.ForFieldLabelRegex(search.ImageRemote),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ImageScanAge,
		querybuilders.ForDays(search.ImageScanTime),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.FirstImageOccuranceAge,
		querybuilders.ForDays(search.FirstImageOccurrenceTimestamp),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.ImageSignatureVerifiedBy,
		querybuilders.ForImageSignatureVerificationStatus(),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return signatureIntegrationIDValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.ImageTag,
		querybuilders.ForFieldLabelRegex(search.ImageTag),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ImageUser,
		querybuilders.ForFieldLabelRegex(search.ImageUser),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.MinimumRBACPermissions,
		querybuilders.ForK8sRBAC(), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return rbacPermissionValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.MountPropagation,
		querybuilders.ForFieldLabel(search.MountPropagation),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return mountPropagationValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.Namespace,
		querybuilders.ForFieldLabelRegex(search.Namespace),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ExposedNodePort,
		querybuilders.ForFieldLabel(search.ExposedNodePort),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ExposedPort,
		querybuilders.ForFieldLabel(search.Port),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.PortExposure,
		querybuilders.ForFieldLabel(search.ExposureLevel),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return portExposureValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.PrivilegedContainer,
		querybuilders.ForFieldLabel(search.Privileged),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.ProcessAncestor,
		querybuilders.ForFieldLabelRegex(search.ProcessAncestor),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process})

	f.registerFieldMetadata(fieldnames.ProcessArguments,
		querybuilders.ForFieldLabelRegex(search.ProcessArguments),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process})

	f.registerFieldMetadata(fieldnames.ProcessName,
		querybuilders.ForFieldLabelRegex(search.ProcessName),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process})

	f.registerFieldMetadata(fieldnames.ProcessUID,
		querybuilders.ForFieldLabel(search.ProcessUID),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process})

	f.registerFieldMetadata(fieldnames.ExposedPortProtocol,
		querybuilders.ForFieldLabelUpper(search.PortProtocol),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.ReadOnlyRootFS,
		querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.RuntimeClass,
		querybuilders.ForFieldLabelRegex(augmentedobjs.RuntimeClassCustomTag),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
	)

	f.registerFieldMetadata(fieldnames.RequiredAnnotation,
		querybuilders.ForFieldLabelMap(search.DeploymentAnnotation, query.MapShouldNotContain),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.RequiredImageLabel,
		querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldNotContain),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.RequiredLabel,
		querybuilders.ForFieldLabelMap(search.DeploymentLabel, query.MapShouldNotContain),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.SeccompProfileType,
		querybuilders.ForFieldLabel(search.SeccompProfileType),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return seccompProfileTypeValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.ServiceAccount,
		querybuilders.ForFieldLabelRegex(search.ServiceAccountName),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.Severity,
		querybuilders.ForSeverity(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return severityValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadata(fieldnames.UnscannedImage,
		querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.VolumeDestination,
		querybuilders.ForFieldLabelRegex(search.VolumeDestination),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.VolumeName,
		querybuilders.ForFieldLabelRegex(search.VolumeName),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.VolumeSource,
		querybuilders.ForFieldLabelRegex(search.VolumeSource),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.VolumeType,
		querybuilders.ForFieldLabelRegex(search.VolumeType),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadata(fieldnames.UnexpectedNetworkFlowDetected,
		querybuilders.ForFieldLabel(augmentedobjs.NotInNetworkBaselineCustomTag),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{NetworkFlow}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.UnexpectedProcessExecuted,
		querybuilders.ForFieldLabel(augmentedobjs.NotInProcessBaselineCustomTag),
		violationmessages.ProcessBaselineContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.WritableHostMount,
		querybuilders.ForWriteableHostMount(),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadata(fieldnames.WritableMountedVolume,
		querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataConditionally(fieldnames.KubeAPIVerb,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesAPIVerbCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.sourceIsAuditLogEvents {
				return auditEventAPIVerbValueRegex
			}
			return kubernetesAPIVerbValueRegex
		}, []storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT, storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent, KubeEvent},
		negationForbidden,
	)

	f.registerFieldMetadataConditionally(fieldnames.KubeResource,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.sourceIsAuditLogEvents {
				return auditEventResourceValueRegex
			}
			return kubernetesResourceValueRegex
		}, []storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT, storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent, KubeEvent},
		negationForbidden,
	)

	f.registerFieldMetadata(
		fieldnames.KubeUserName,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserNameCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return kubernetesNameRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadata(
		fieldnames.KubeUserGroups,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserGroupsCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return kubernetesNameRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadata(
		fieldnames.KubeResourceName,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceNameCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return kubernetesNameRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadata(
		fieldnames.SourceIPAddress,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesSourceIPAddressCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return ipAddressValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)
	f.registerFieldMetadata(
		fieldnames.UserAgent,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserAgentCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadata(
		fieldnames.IsImpersonatedUser,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesIsImpersonatedCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
		negationForbidden, operatorsForbidden,
	)

	f.registerFieldMetadata(fieldnames.Replicas,
		querybuilders.ForFieldLabel(search.Replicas),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
	)

	f.registerFieldMetadata(fieldnames.LivenessProbeDefined,
		querybuilders.ForFieldLabel(search.LivenessProbeDefined),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden, operatorsForbidden,
	)

	f.registerFieldMetadata(fieldnames.ReadinessProbeDefined,
		querybuilders.ForFieldLabel(search.ReadinessProbeDefined),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	f.registerFieldMetadata(fieldnames.HasIngressNetworkPolicy,
		querybuilders.ForFieldLabel(augmentedobjs.HasIngressPolicyCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	f.registerFieldMetadata(fieldnames.HasEgressNetworkPolicy,
		querybuilders.ForFieldLabel(augmentedobjs.HasEgressPolicyCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	return f
}
