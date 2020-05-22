package booleanpolicy

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/querybuilders"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/search"
)

var (
	fieldsToQB = make(map[string]*metadataAndQB)
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

func registerFieldMetadata(fieldName string, qb querybuilders.QueryBuilder, contextFields violationmessages.ContextQueryFields, valueRegex *regexp.Regexp, options ...option) {
	if fieldsToQB[fieldName] != nil {
		panic(fmt.Sprintf("found duplicate metadata for field %s", fieldName))
	}

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

	fieldsToQB[fieldName] = m
}

func init() {
	registerFieldMetadata(fieldnames.AddCaps, querybuilders.ForFieldLabelExact(search.AddCapabilities), violationmessages.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.CVE, querybuilders.ForCVE(), violationmessages.VulnContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.CVSS, querybuilders.ForCVSS(), violationmessages.VulnContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.ContainerCPULimit, querybuilders.ForFieldLabel(search.CPUCoresLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.ContainerCPURequest, querybuilders.ForFieldLabel(search.CPUCoresRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.ContainerMemLimit, querybuilders.ForFieldLabel(search.MemoryLimit), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.ContainerMemRequest, querybuilders.ForFieldLabel(search.MemoryRequest), violationmessages.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.DisallowedAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.ShouldContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DisallowedImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.ShouldContain), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DockerfileLine, querybuilders.ForCompound(augmentedobjs.DockerfileLineCustomTag, 2), violationmessages.ImageContextFields, dockerfileLineValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.DropCaps, querybuilders.ForDropCaps(), violationmessages.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.EnvironmentVariable, querybuilders.ForCompound(augmentedobjs.EnvironmentVarCustomTag, 3), violationmessages.EnvVarContextFields, environmentVariableWithSourceRegex, negationForbidden)
	registerFieldMetadata(fieldnames.FixedBy, querybuilders.ForFieldLabelRegex(search.FixedBy), violationmessages.VulnContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageAge, querybuilders.ForDays(search.ImageCreatedTime), violationmessages.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageComponent, querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.ImageRegistry, querybuilders.ForFieldLabelRegex(search.ImageRegistry), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageRemote, querybuilders.ForFieldLabelRegex(search.ImageRemote), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ImageScanAge, querybuilders.ForDays(search.ImageScanTime), violationmessages.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ImageTag, querybuilders.ForFieldLabelRegex(search.ImageTag), violationmessages.ImageContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.MinimumRBACPermissions, querybuilders.ForK8sRBAC(), nil, rbacPermissionValueRegex, operatorsForbidden)
	registerFieldMetadata(fieldnames.Port, querybuilders.ForFieldLabel(search.Port), violationmessages.PortContextFields, integerValueRegex)
	registerFieldMetadata(fieldnames.PortExposure, querybuilders.ForFieldLabel(search.ExposureLevel), violationmessages.PortContextFields, portExposureValueRegex)
	registerFieldMetadata(fieldnames.Privileged, querybuilders.ForFieldLabel(search.Privileged), violationmessages.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.ProcessAncestor, querybuilders.ForFieldLabelRegex(search.ProcessAncestor), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessArguments, querybuilders.ForFieldLabelRegex(search.ProcessArguments), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessName, querybuilders.ForFieldLabelRegex(search.ProcessName), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.ProcessUID, querybuilders.ForFieldLabel(search.ProcessUID), nil, stringValueRegex)
	registerFieldMetadata(fieldnames.Protocol, querybuilders.ForFieldLabelUpper(search.PortProtocol), violationmessages.PortContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.ReadOnlyRootFS, querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem), violationmessages.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.RequiredAnnotation, querybuilders.ForFieldLabelMap(search.Annotation, query.ShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredImageLabel, querybuilders.ForFieldLabelMap(search.ImageLabel, query.ShouldNotContain), violationmessages.ImageContextFields, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.RequiredLabel, querybuilders.ForFieldLabelMap(search.Label, query.ShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	registerFieldMetadata(fieldnames.UnscannedImage, querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag), violationmessages.ImageContextFields, booleanValueRegex)
	registerFieldMetadata(fieldnames.VolumeDestination, querybuilders.ForFieldLabelRegex(search.VolumeDestination), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeName, querybuilders.ForFieldLabelRegex(search.VolumeName), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeSource, querybuilders.ForFieldLabelRegex(search.VolumeSource), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.VolumeType, querybuilders.ForFieldLabelRegex(search.VolumeType), violationmessages.VolumeContextFields, stringValueRegex)
	registerFieldMetadata(fieldnames.WhitelistsEnabled, querybuilders.ForFieldLabel(augmentedobjs.NotWhitelistedCustomTag), violationmessages.ProcessWhitelistContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableHostMount, querybuilders.ForWriteableHostMount(), violationmessages.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	registerFieldMetadata(fieldnames.WritableVolume, querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true), violationmessages.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
}
