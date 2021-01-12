package booleanpolicy

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/querybuilders"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
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
}

func findFieldMetadata(fieldName string, config *validateConfiguration) (*metadataAndQB, error) {
	f := fieldsToQB[fieldName]
	if f == nil {
		return nil, errNoSuchField
	}
	return f(config), nil
}

func newFieldMetadata(qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, valueRegex *regexp.Regexp, options ...option) *metadataAndQB {
	m := &metadataAndQB{
		qb:            qb,
		contextFields: contextFields,
		valueRegex:    valueRegex,
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

func registerFieldMetadata(fieldName string, qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, valueRegex *regexp.Regexp, options ...option) {
	ensureFieldIsUnique(fieldName)

	m := newFieldMetadata(qb, contextFields, valueRegex, options...)
	fieldsToQB[fieldName] = func(*validateConfiguration) *metadataAndQB {
		return m
	}
}

func registerFieldMetadataConditionally(
	fieldName string,
	qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, conditionalRegexp func(*validateConfiguration) *regexp.Regexp, options ...option,
) {
	ensureFieldIsUnique(fieldName)
	fieldsToQB[fieldName] = func(configuration *validateConfiguration) *metadataAndQB {
		return newFieldMetadata(qb, contextFields, conditionalRegexp(configuration), options...)
	}
}

func init() {
	registerFieldMetadata(fieldnames.AddCaps, querybuilders.ForFieldLabelExact(search.AddCapabilities), violationmessages.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.CVE, querybuilders.ForCVE(), violationmessages.VulnContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.CVSS, querybuilders.ForCVSS(), violationmessages.VulnContextFields, comparatorDecimalValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerCPULimit, querybuilders.ForFieldLabel(search.CPUCoresLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerCPURequest, querybuilders.ForFieldLabel(search.CPUCoresRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerMemLimit, querybuilders.ForFieldLabel(search.MemoryLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerMemRequest, querybuilders.ForFieldLabel(search.MemoryRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ContainerName, querybuilders.ForFieldLabelRegex(search.ContainerName), violationmessages.ContainerContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.DisallowedAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.MapShouldContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DisallowedImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldContain), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DockerfileLine, querybuilders.ForCompound(augmentedobjs.DockerfileLineCustomTag, 2), violationmessages.ImageContextFields, dockerfileLineValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DropCaps, querybuilders.ForDropCaps(), violationmessages.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	registerFieldMetadataConditionally(
		fieldnames.EnvironmentVariable,
		querybuilders.ForCompound(augmentedobjs.EnvironmentVarCustomTag, 3), violationmessages.EnvVarContextFields, func(c *validateConfiguration) *regexp.Regexp {
			if c != nil && c.validateEnvVarSourceRestrictions {
				return environmentVariableWithSourceStrictRegex
			}

			return environmentVariableWithSourceRegex
		}, negationForbidden,
	)
	registerFieldMetadata(fieldnames.FixedBy, querybuilders.ForFixedBy(), violationmessages.VulnContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageAge, querybuilders.ForDays(search.ImageCreatedTime), violationmessages.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageComponent, querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ImageOS, querybuilders.ForFieldLabel(search.ImageOS), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageRegistry, querybuilders.ForFieldLabelRegex(search.ImageRegistry), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageRemote, querybuilders.ForFieldLabelRegex(search.ImageRemote), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageScanAge, querybuilders.ForDays(search.ImageScanTime), violationmessages.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageTag, querybuilders.ForFieldLabelRegex(search.ImageTag), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.MinimumRBACPermissions, querybuilders.ForK8sRBAC(), nil, rbacPermissionValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.Namespace, querybuilders.ForFieldLabelRegex(search.Namespace), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ExposedPort, querybuilders.ForFieldLabel(search.Port), violationmessages.PortContextFields, integerValueRegex)
	registerFieldMetadata(fieldnames.PortExposure, querybuilders.ForFieldLabel(search.ExposureLevel), violationmessages.PortContextFields, portExposureValueRegex)
	registerFieldMetadata(fieldnames.PrivilegedContainer, querybuilders.ForFieldLabel(search.Privileged), violationmessages.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ProcessAncestor, querybuilders.ForFieldLabelRegex(search.ProcessAncestor), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessArguments, querybuilders.ForFieldLabelRegex(search.ProcessArguments), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessName, querybuilders.ForFieldLabelRegex(search.ProcessName), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessUID, querybuilders.ForFieldLabel(search.ProcessUID), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ExposedPortProtocol, querybuilders.ForFieldLabelUpper(search.PortProtocol), violationmessages.PortContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ReadOnlyRootFS, querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem), violationmessages.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.RequiredAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.MapShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.MapShouldNotContain), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredLabel, querybuilders.ForFieldLabelMap(search.Label, query.MapShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ServiceAccount, querybuilders.ForFieldLabelRegex(search.ServiceAccountName), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.UnscannedImage, querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag), violationmessages.ImageContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.VolumeDestination, querybuilders.ForFieldLabelRegex(search.VolumeDestination), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeName, querybuilders.ForFieldLabelRegex(search.VolumeName), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeSource, querybuilders.ForFieldLabelRegex(search.VolumeSource), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeType, querybuilders.ForFieldLabelRegex(search.VolumeType), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.UnexpectedProcessExecuted, querybuilders.ForFieldLabel(augmentedobjs.NotInBaselineCustomTag), violationmessages.ProcessBaselineContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableHostMount, querybuilders.ForWriteableHostMount(), violationmessages.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableMountedVolume, querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true), violationmessages.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.KubeAPIVerb, querybuilders.ForFieldLabel(augmentedobjs.KubernetesAPIVerbCustomTag), nil, kubernetesAPIVerbValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.KubeResource, querybuilders.ForFieldLabel(augmentedobjs.KubernetesResourceCustomTag), nil, kubernetesResourceValueRegex, negationForbidden, operatorsForbidden)
}
