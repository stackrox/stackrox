package booleanpolicy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	globstar "github.com/bmatcuk/doublestar/v4"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/querybuilders"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/features"
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

// ResetFieldMetadataSingleton is for testing purposes only, and can be
// used to ensure that criteria are added / removed when feature flags
// are enabled / disabled respectively.
func ResetFieldMetadataSingleton(_ *testing.T) {
	fieldMetadataInstanceInit = sync.Once{}
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
	// FileAccess for a file-based runtime event
	FileAccess = "fileAccess"
)

type valueValidatorFunc func(config *validateConfiguration, value string) (bool, error)
type regexValueValidatorFunc func(*validateConfiguration) *regexp.Regexp

func regexValueValidator(f regexValueValidatorFunc) valueValidatorFunc {
	return func(config *validateConfiguration, value string) (bool, error) {
		if !f(config).MatchString(value) {
			return false, fmt.Errorf("must match %q", f(config).String())
		}
		return true, nil
	}
}

type metadataAndQB struct {
	operatorsForbidden bool
	negationForbidden  bool
	qb                 querybuilders.QueryBuilder
	validator          valueValidatorFunc
	contextFields      violationmessages.ContextQueryFields
	eventSourceContext []storage.EventSource
	fieldTypes         []RuntimeFieldType
}

func (m *metadataAndQB) IsOfType(expectedType RuntimeFieldType) bool {
	return slices.Contains(m.fieldTypes, expectedType)
}

func (m *metadataAndQB) IsDeploymentEventField() bool {
	return m.IsOfType(Process) || m.IsOfType(NetworkFlow) || m.IsOfType(KubeEvent) || m.IsOfType(FileAccess)
}

func (m *metadataAndQB) IsAuditLogEventField() bool {
	return m.IsOfType(AuditLogEvent)
}

func (m *metadataAndQB) IsFileEventField() bool {
	return m.IsOfType(FileAccess)
}

func (m *metadataAndQB) IsFromEventSource(eventSource storage.EventSource) bool {
	return slices.Contains(m.eventSourceContext, eventSource)
}

func (m *metadataAndQB) IsNotApplicableEventSource() bool {
	return m.IsFromEventSource(storage.EventSource_NOT_APPLICABLE)
}

func (f *FieldMetadata) findField(fieldName string) *metadataAndQB {
	field := f.fieldsToQB[fieldName]
	if field == nil {
		log.Warnf("policy field %s not found", fieldName)
	}
	return field
}

// FieldIsOfType returns true if the specified field is of the specified type
func (f *FieldMetadata) FieldIsOfType(fieldName string, expectedType RuntimeFieldType) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsOfType(expectedType)
	}
	return false
}

// IsDeploymentEventField returns true if the field is an deployment event field
func (f *FieldMetadata) IsDeploymentEventField(fieldName string) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsDeploymentEventField()
	}
	return false

}

// IsAuditLogEventField returns true if the field is an audit log field
func (f *FieldMetadata) IsAuditLogEventField(fieldName string) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsAuditLogEventField()
	}
	return false

}

// IsFileEventField returns true if the field is a node event field
func (f *FieldMetadata) IsFileEventField(fieldName string) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsFileEventField()
	}

	return false
}

func (f *FieldMetadata) IsFromEventSource(fieldName string, eventSource storage.EventSource) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsFromEventSource(eventSource)
	}
	return false
}

func (f *FieldMetadata) IsNotApplicableEventSource(fieldName string) bool {
	if field := f.findField(fieldName); field != nil {
		return field.IsNotApplicableEventSource()
	}
	return false
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
	validator valueValidatorFunc, source []storage.EventSource,
	fieldTypes []RuntimeFieldType, options ...option) *metadataAndQB {
	m := &metadataAndQB{
		qb:                 qb,
		contextFields:      contextFields,
		validator:          validator,
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

func (f *FieldMetadata) registerFieldMetadataRegex(fieldName string, qb querybuilders.QueryBuilder,
	contextFields violationmessages.ContextQueryFields,
	regex func(configuration *validateConfiguration) *regexp.Regexp,
	source []storage.EventSource, fieldTypes []RuntimeFieldType, options ...option) {

	f.registerFieldMetadata(fieldName, qb, contextFields, regexValueValidator(regex), source, fieldTypes, options...)
}

func (f *FieldMetadata) registerFieldMetadata(fieldName string, qb querybuilders.QueryBuilder,
	contextFields violationmessages.ContextQueryFields,
	validator valueValidatorFunc,
	source []storage.EventSource, fieldTypes []RuntimeFieldType, options ...option) {

	f.ensureFieldIsUnique(fieldName)
	f.fieldsToQB[fieldName] = newFieldMetadata(qb, contextFields, validator, source, fieldTypes, options...)
}

func initializeFieldMetadata() FieldMetadata {
	f := FieldMetadata{
		fieldsToQB: make(map[string]*metadataAndQB),
	}

	f.registerFieldMetadataRegex(fieldnames.AddCaps,
		querybuilders.ForAddCaps(),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return addCapabilitiesValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.AllowPrivilegeEscalation,
		querybuilders.ForFieldLabel(search.AllowPrivilegeEscalation),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.AppArmorProfile,
		querybuilders.ForFieldLabelRegex(search.AppArmorProfile),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.AutomountServiceAccountToken,
		querybuilders.ForFieldLabel(search.AutomountServiceAccountToken),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden,
		operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.CVE,
		querybuilders.ForCVE(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.CVSS, querybuilders.ForCVSS(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ContainerCPULimit,
		querybuilders.ForFieldLabel(search.CPUCoresLimit),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ContainerCPURequest,
		querybuilders.ForFieldLabel(search.CPUCoresRequest),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ContainerMemLimit,
		querybuilders.ForFieldLabel(search.MemoryLimit),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ContainerMemRequest,
		querybuilders.ForFieldLabel(search.MemoryRequest),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ContainerName,
		querybuilders.ForFieldLabelRegex(search.ContainerName),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.DaysSinceImageFirstDiscovered,
		querybuilders.ForDays(search.FirstImageOccurrenceTimestamp),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.DaysSinceSystemFirstDiscovered,
		querybuilders.ForDays(search.FirstSystemOccurrenceTimestamp),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.DaysSincePublished,
		querybuilders.ForDays(search.CVEPublishedOn),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	if features.CVEFixTimestampCriteria.Enabled() {
		f.registerFieldMetadataRegex(fieldnames.DaysSinceFixAvailable,
			querybuilders.ForDays(search.CVEFixAvailable),
			violationmessages.VulnContextFields,
			func(*validateConfiguration) *regexp.Regexp {
				return integerValueRegex
			},
			[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
			[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)
	}

	f.registerFieldMetadataRegex(fieldnames.DisallowedAnnotation,
		querybuilders.ForFieldLabelMap(search.DeploymentAnnotation, query.MapShouldContain),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.DisallowedImageLabel,
		querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldContain),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.DockerfileLine,
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

	f.registerFieldMetadataRegex(fieldnames.DropCaps,
		querybuilders.ForDropCaps(),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return dropCapabilitiesValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.EnvironmentVariable,
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

	f.registerFieldMetadataRegex(fieldnames.Fixable,
		querybuilders.ForFixable(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.FixedBy,
		querybuilders.ForFixedBy(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.HostIPC,
		querybuilders.ForFieldLabel(search.HostIPC),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.HostNetwork,
		querybuilders.ForFieldLabel(search.HostNetwork),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.HostPID,
		querybuilders.ForFieldLabel(search.HostPID), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.ImageAge,
		querybuilders.ForDays(search.ImageCreatedTime),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.ImageComponent,
		querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ImageOS,
		querybuilders.ForFieldLabelRegex(search.ImageOS),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ImageRegistry,
		querybuilders.ForFieldLabelRegex(search.ImageRegistry),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ImageRemote,
		querybuilders.ForFieldLabelRegex(search.ImageRemote),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ImageScanAge,
		querybuilders.ForDays(search.ImageScanTime),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return integerValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.ImageSignatureVerifiedBy,
		querybuilders.ForImageSignatureVerificationStatus(),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return signatureIntegrationIDValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ImageTag,
		querybuilders.ForFieldLabelRegex(search.ImageTag),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ImageUser,
		querybuilders.ForFieldLabelRegex(search.ImageUser),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.MinimumRBACPermissions,
		querybuilders.ForK8sRBAC(), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return rbacPermissionValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.MountPropagation,
		querybuilders.ForFieldLabel(search.MountPropagation),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return mountPropagationValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.Namespace,
		querybuilders.ForFieldLabelRegex(search.Namespace),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.NvdCvss, querybuilders.ForNvdCVSS(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.ExposedNodePort,
		querybuilders.ForFieldLabel(search.ExposedNodePort),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ExposedPort,
		querybuilders.ForFieldLabel(search.Port),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.PortExposure,
		querybuilders.ForFieldLabel(search.ExposureLevel),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return portExposureValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.PrivilegedContainer,
		querybuilders.ForFieldLabel(search.Privileged),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.ProcessAncestor,
		querybuilders.ForFieldLabelRegex(search.ProcessAncestor),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process, FileAccess})

	f.registerFieldMetadataRegex(fieldnames.ProcessArguments,
		querybuilders.ForFieldLabelContainsRegex(search.ProcessArguments),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process, FileAccess})

	f.registerFieldMetadataRegex(fieldnames.ProcessName,
		querybuilders.ForFieldLabelRegex(search.ProcessName),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process, FileAccess})

	f.registerFieldMetadataRegex(fieldnames.ProcessUID,
		querybuilders.ForFieldLabel(search.ProcessUID),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process, FileAccess})

	f.registerFieldMetadataRegex(fieldnames.ExposedPortProtocol,
		querybuilders.ForFieldLabelUpper(search.PortProtocol),
		violationmessages.PortContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.ReadOnlyRootFS,
		querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.RuntimeClass,
		querybuilders.ForFieldLabelRegex(augmentedobjs.RuntimeClassCustomTag),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
	)

	f.registerFieldMetadataRegex(fieldnames.RequiredAnnotation,
		querybuilders.ForFieldLabelMapRequired(search.DeploymentAnnotation),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.RequiredImageLabel,
		querybuilders.ForFieldLabelMapRequired(search.ImageLabel),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.RequiredLabel,
		querybuilders.ForFieldLabelMapRequired(search.DeploymentLabel),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return keyValueValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.SeccompProfileType,
		querybuilders.ForFieldLabel(search.SeccompProfileType),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return seccompProfileTypeValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.ServiceAccount,
		querybuilders.ForFieldLabelRegex(search.ServiceAccountName),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.Severity,
		querybuilders.ForSeverity(),
		violationmessages.VulnContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return severityValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden)

	f.registerFieldMetadataRegex(fieldnames.UnscannedImage,
		querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag),
		violationmessages.ImageContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.VolumeDestination,
		querybuilders.ForFieldLabelRegex(search.VolumeDestination),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.VolumeName,
		querybuilders.ForFieldLabelRegex(search.VolumeName),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.VolumeSource,
		querybuilders.ForFieldLabelRegex(search.VolumeSource),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.VolumeType,
		querybuilders.ForFieldLabelRegex(search.VolumeType),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{})

	f.registerFieldMetadataRegex(fieldnames.UnexpectedNetworkFlowDetected,
		querybuilders.ForFieldLabel(augmentedobjs.NotInNetworkBaselineCustomTag),
		nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{NetworkFlow}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.UnexpectedProcessExecuted,
		querybuilders.ForFieldLabel(augmentedobjs.NotInProcessBaselineCustomTag),
		violationmessages.ProcessBaselineContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT},
		[]RuntimeFieldType{Process}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.WritableHostMount,
		querybuilders.ForWriteableHostMount(),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.WritableMountedVolume,
		querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true),
		violationmessages.VolumeContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, negationForbidden, operatorsForbidden)

	f.registerFieldMetadataRegex(fieldnames.KubeAPIVerb,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesAPIVerbCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			return auditEventAPIVerbValueRegex
		}, []storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
		negationForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.KubeResource,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceCustomTag),
		nil,
		func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.sourceIsAuditLogEvents {
				return auditEventResourceValueRegex
			}
			return kubernetesResourceValueRegex
		}, []storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT, storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{KubeEvent, AuditLogEvent},
		negationForbidden,
	)

	f.registerFieldMetadataRegex(
		fieldnames.KubeUserName,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserNameCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return kubernetesNameRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT, storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{KubeEvent, AuditLogEvent},
	)

	f.registerFieldMetadataRegex(
		fieldnames.KubeUserGroups,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserGroupsCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return kubernetesNameRegex
		},
		[]storage.EventSource{storage.EventSource_DEPLOYMENT_EVENT, storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{KubeEvent, AuditLogEvent},
	)

	f.registerFieldMetadataRegex(
		fieldnames.KubeResourceName,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceNameCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadataRegex(
		fieldnames.SourceIPAddress,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesSourceIPAddressCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return ipAddressValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)
	f.registerFieldMetadataRegex(
		fieldnames.UserAgent,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesUserAgentCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return stringValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
	)

	f.registerFieldMetadataRegex(
		fieldnames.IsImpersonatedUser,
		querybuilders.ForFieldLabel(augmentedobjs.KubernetesIsImpersonatedCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_AUDIT_LOG_EVENT},
		[]RuntimeFieldType{AuditLogEvent},
		negationForbidden, operatorsForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.Replicas,
		querybuilders.ForFieldLabel(search.Replicas),
		violationmessages.ResourceContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return comparatorDecimalValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
	)

	f.registerFieldMetadataRegex(fieldnames.LivenessProbeDefined,
		querybuilders.ForFieldLabel(search.LivenessProbeDefined),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{},
		negationForbidden, operatorsForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.ReadinessProbeDefined,
		querybuilders.ForFieldLabel(search.ReadinessProbeDefined),
		violationmessages.ContainerContextFields,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.HasIngressNetworkPolicy,
		querybuilders.ForFieldLabel(augmentedobjs.HasIngressPolicyCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	f.registerFieldMetadataRegex(fieldnames.HasEgressNetworkPolicy,
		querybuilders.ForFieldLabel(augmentedobjs.HasEgressPolicyCustomTag), nil,
		func(*validateConfiguration) *regexp.Regexp {
			return booleanValueRegex
		},
		[]storage.EventSource{storage.EventSource_NOT_APPLICABLE},
		[]RuntimeFieldType{}, operatorsForbidden,
	)

	if features.SensitiveFileActivity.Enabled() {
		f.registerFieldMetadata(fieldnames.FilePath,
			querybuilders.ForFieldLabelFilePath(augmentedobjs.FileAccessPathCustomTag), nil,
			func(config *validateConfiguration, value string) (bool, error) {
				if !filepath.IsAbs(value) {
					return false, errors.New("path must be absolute")
				}

				if slices.Contains(strings.Split(value, string(filepath.Separator)), "..") {
					return false, errors.New("path must not contain traversal '..'")
				}

				if !globstar.ValidatePattern(value) {
					return false, errors.New("path contains invalid wildcard pattern")
				}

				return true, nil
			},
			[]storage.EventSource{storage.EventSource_NODE_EVENT, storage.EventSource_DEPLOYMENT_EVENT},
			[]RuntimeFieldType{FileAccess}, negationForbidden,
		)

		f.registerFieldMetadataRegex(fieldnames.FileOperation,
			querybuilders.ForFieldLabel(search.FileOperation), nil,
			func(*validateConfiguration) *regexp.Regexp {
				return fileOperationRegex
			},
			[]storage.EventSource{storage.EventSource_NODE_EVENT, storage.EventSource_DEPLOYMENT_EVENT},
			[]RuntimeFieldType{FileAccess},
		)
	}

	return f
}
