package booleanpolicy

import (
	"regexp"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/querybuilders"
	"github.com/stackrox/rox/pkg/booleanpolicy/violations"
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
	contextFields      violations.ContextQueryFields
}

// This block enumerates field short names.
var (
	AddCaps                = newField("Add Capabilities", querybuilders.ForFieldLabelExact(search.AddCapabilities), violations.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	CVE                    = newField("CVE", querybuilders.ForCVE(), violations.VulnContextFields, stringValueRegex)
	CVSS                   = newField("CVSS", querybuilders.ForCVSS(), violations.VulnContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	ContainerCPULimit      = newField("Container CPU Limit", querybuilders.ForFieldLabel(search.CPUCoresLimit), violations.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	ContainerCPURequest    = newField("Container CPU Request", querybuilders.ForFieldLabel(search.CPUCoresRequest), violations.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	ContainerMemLimit      = newField("Container Memory Limit", querybuilders.ForFieldLabel(search.MemoryLimit), violations.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	ContainerMemRequest    = newField("Container Memory Request", querybuilders.ForFieldLabel(search.MemoryRequest), violations.ResourceContextFields, comparatorDecimalValueRegex, operatorsForbidden)
	DisallowedAnnotation   = newField("Disallowed Annotation", querybuilders.ForFieldLabelMap(search.Annotation, query.ShouldContain), nil, keyValueValueRegex, negationForbidden)
	DisallowedImageLabel   = newField("Disallowed Image Label", querybuilders.ForFieldLabelMap(search.ImageLabel, query.ShouldContain), violations.ImageContextFields, keyValueValueRegex, negationForbidden)
	DockerfileLine         = newField("Dockerfile Line", querybuilders.ForCompound(augmentedobjs.DockerfileLineCustomTag, 2), violations.ImageContextFields, dockerfileLineValueRegex, negationForbidden)
	DropCaps               = newField("Drop Capabilities", querybuilders.ForDropCaps(), violations.ContainerContextFields, capabilitiesValueRegex, negationForbidden)
	EnvironmentVariable    = newField("Environment Variable", querybuilders.ForCompound(augmentedobjs.EnvironmentVarCustomTag, 3), violations.EnvVarContextFields, environmentVariableWithSourceRegex, negationForbidden)
	FixedBy                = newField("Fixed By", querybuilders.ForFieldLabelRegex(search.FixedBy), violations.VulnContextFields, stringValueRegex)
	ImageAge               = newField("Image Age", querybuilders.ForDays(search.ImageCreatedTime), violations.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	ImageComponent         = newField("Image Component", querybuilders.ForCompound(augmentedobjs.ComponentAndVersionCustomTag, 2), violations.ImageContextFields, keyValueValueRegex, negationForbidden)
	ImageRegistry          = newField("Image Registry", querybuilders.ForFieldLabelRegex(search.ImageRegistry), violations.ImageContextFields, stringValueRegex)
	ImageRemote            = newField("Image Remote", querybuilders.ForFieldLabelRegex(search.ImageRemote), violations.ImageContextFields, stringValueRegex)
	ImageScanAge           = newField("Image Scan Age", querybuilders.ForDays(search.ImageScanTime), violations.ImageContextFields, integerValueRegex, negationForbidden, operatorsForbidden)
	ImageTag               = newField("Image Tag", querybuilders.ForFieldLabelRegex(search.ImageTag), violations.ImageContextFields, stringValueRegex)
	MinimumRBACPermissions = newField("Minimum RBAC Permissions", querybuilders.ForK8sRBAC(), nil, rbacPermissionValueRegex, operatorsForbidden)
	Port                   = newField("Port", querybuilders.ForFieldLabel(search.Port), violations.PortContextFields, integerValueRegex)
	PortExposure           = newField("Port Exposure Method", querybuilders.ForFieldLabel(search.ExposureLevel), violations.PortContextFields, portExposureValueRegex)
	Privileged             = newField("Privileged", querybuilders.ForFieldLabel(search.Privileged), violations.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	ProcessAncestor        = newField("Process Ancestor", querybuilders.ForFieldLabelRegex(search.ProcessAncestor), nil, stringValueRegex)
	ProcessArguments       = newField("Process Arguments", querybuilders.ForFieldLabelRegex(search.ProcessArguments), nil, stringValueRegex)
	ProcessName            = newField("Process Name", querybuilders.ForFieldLabelRegex(search.ProcessName), nil, stringValueRegex)
	ProcessUID             = newField("Process UID", querybuilders.ForFieldLabel(search.ProcessUID), nil, stringValueRegex)
	Protocol               = newField("Protocol", querybuilders.ForFieldLabelUpper(search.PortProtocol), violations.PortContextFields, stringValueRegex)
	ReadOnlyRootFS         = newField("Read-Only Root Filesystem", querybuilders.ForFieldLabel(search.ReadOnlyRootFilesystem), violations.ContainerContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	RequiredAnnotation     = newField("Required Annotation", querybuilders.ForFieldLabelMap(search.Annotation, query.ShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	RequiredImageLabel     = newField("Required Image Label", querybuilders.ForFieldLabelMap(search.ImageLabel, query.ShouldNotContain), violations.ImageContextFields, keyValueValueRegex, negationForbidden)
	RequiredLabel          = newField("Required Label", querybuilders.ForFieldLabelMap(search.Label, query.ShouldNotContain), nil, keyValueValueRegex, negationForbidden)
	UnscannedImage         = newField("Unscanned Image", querybuilders.ForFieldLabelNil(augmentedobjs.ImageScanCustomTag), violations.ImageContextFields, booleanValueRegex)
	VolumeDestination      = newField("Volume Destination", querybuilders.ForFieldLabelRegex(search.VolumeDestination), violations.VolumeContextFields, stringValueRegex)
	VolumeName             = newField("Volume Name", querybuilders.ForFieldLabelRegex(search.VolumeName), violations.VolumeContextFields, stringValueRegex)
	VolumeSource           = newField("Volume Source", querybuilders.ForFieldLabelRegex(search.VolumeSource), violations.VolumeContextFields, stringValueRegex)
	VolumeType             = newField("Volume Type", querybuilders.ForFieldLabelRegex(search.VolumeType), violations.VolumeContextFields, stringValueRegex)
	WhitelistsEnabled      = newField("Unexpected Process Executed", querybuilders.ForFieldLabel(augmentedobjs.NotWhitelistedCustomTag), violations.ProcessWhitelistContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	// TODO(rc) check volume type is hostpath and not read only
	WritableHostMount = newField("Writable Host Mount", nil, violations.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
	WritableVolume    = newField("Writable Volume", querybuilders.ForFieldLabelBoolean(search.VolumeReadonly, true), violations.VolumeContextFields, booleanValueRegex, negationForbidden, operatorsForbidden)
)

func newField(fieldName string, qb querybuilders.QueryBuilder, contextFields violations.ContextQueryFields, valueRegex *regexp.Regexp, options ...option) string {
	m := metadataAndQB{
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
	fieldsToQB[fieldName] = &m
	return fieldName
}
