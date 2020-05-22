package violationmessages

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// A printerFunc prints violation messages from a map of required fields to values
type printerFunc func(map[string][]string) ([]string, error)

func stringSetFromPolicySectionFields(section *storage.PolicySection) set.StringSet {
	sectionFields := set.NewStringSet()
	for _, group := range section.GetPolicyGroups() {
		sectionFields.Add(group.GetFieldName())
	}
	return sectionFields
}

type violationPrinter struct {
	required set.StringSet // These fields must all be in the result, and must be valid search tags
	printer  printerFunc
}

var (
	policyFieldsToPrinters = map[storage.LifecycleStage]map[string][]violationPrinter{
		storage.LifecycleStage_DEPLOY: {
			fieldnames.AddCaps:                {{required: set.NewStringSet(search.AddCapabilities.String()), printer: addCapabilityPrinter}},
			fieldnames.CVE:                    {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.CVSS:                   {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.ContainerCPULimit:      {{required: set.NewStringSet(search.CPUCoresLimit.String()), printer: resourcePrinter}},
			fieldnames.ContainerCPURequest:    {{required: set.NewStringSet(search.CPUCoresRequest.String()), printer: resourcePrinter}},
			fieldnames.ContainerMemLimit:      {{required: set.NewStringSet(search.MemoryLimit.String()), printer: resourcePrinter}},
			fieldnames.ContainerMemRequest:    {{required: set.NewStringSet(search.MemoryRequest.String()), printer: resourcePrinter}},
			fieldnames.DisallowedAnnotation:   {{required: set.NewStringSet(search.Annotation.String()), printer: mapPrinter}},
			fieldnames.DisallowedImageLabel:   {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			fieldnames.DockerfileLine:         {{required: set.NewStringSet(augmentedobjs.DockerfileLineCustomTag), printer: linePrinter}},
			fieldnames.DropCaps:               {{required: set.NewStringSet(search.DropCapabilities.String()), printer: dropCapabilityPrinter}},
			fieldnames.EnvironmentVariable:    {{required: set.NewStringSet(augmentedobjs.EnvironmentVarCustomTag), printer: envPrinter}},
			fieldnames.FixedBy:                {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.ImageAge:               {{required: set.NewStringSet(search.ImageCreatedTime.String()), printer: imageAgePrinter}},
			fieldnames.ImageComponent:         {{required: set.NewStringSet(augmentedobjs.ComponentAndVersionCustomTag), printer: componentPrinter}},
			fieldnames.ImageRegistry:          {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.ImageRemote:            {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.ImageScanAge:           {{required: set.NewStringSet(search.ImageScanTime.String()), printer: imageScanAgePrinter}},
			fieldnames.ImageTag:               {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.MinimumRBACPermissions: {{required: set.NewStringSet(search.ServiceAccountPermissionLevel.String()), printer: rbacPrinter}},
			fieldnames.Port:                   {{required: set.NewStringSet(search.Port.String()), printer: portPrinter}},
			fieldnames.PortExposure:           {{required: set.NewStringSet(search.ExposureLevel.String()), printer: portExposurePrinter}},
			fieldnames.Privileged:             {{required: set.NewStringSet(search.Privileged.String()), printer: privilegedPrinter}},
			fieldnames.Protocol:               {{required: set.NewStringSet(search.Port.String()), printer: portPrinter}},
			fieldnames.ReadOnlyRootFS:         {{required: set.NewStringSet(search.ReadOnlyRootFilesystem.String()), printer: readOnlyRootFSPrinter}},
			fieldnames.RequiredAnnotation:     {{required: set.NewStringSet(search.Annotation.String()), printer: mapPrinter}},
			fieldnames.RequiredImageLabel:     {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			fieldnames.RequiredLabel:          {{required: set.NewStringSet(search.Label.String()), printer: mapPrinter}},
			fieldnames.WhitelistsEnabled:      {{required: set.NewStringSet(augmentedobjs.NotWhitelistedCustomTag), printer: processWhitelistPrinter}},
			fieldnames.UnscannedImage:         {{required: set.NewStringSet(augmentedobjs.ImageScanCustomTag), printer: imageScanPrinter}},
			fieldnames.VolumeDestination:      {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			fieldnames.VolumeName:             {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			fieldnames.VolumeSource:           {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			fieldnames.VolumeType:             {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			fieldnames.WritableHostMount:      {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			fieldnames.WritableVolume:         {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
		},
		storage.LifecycleStage_BUILD: {
			fieldnames.CVE:                  {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.CVSS:                 {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.DisallowedImageLabel: {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			fieldnames.DockerfileLine:       {{required: set.NewStringSet(augmentedobjs.DockerfileLineCustomTag), printer: linePrinter}},
			fieldnames.FixedBy:              {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			fieldnames.ImageAge:             {{required: set.NewStringSet(search.ImageCreatedTime.String()), printer: imageAgePrinter}},
			fieldnames.ImageComponent:       {{required: set.NewStringSet(augmentedobjs.ComponentAndVersionCustomTag), printer: componentPrinter}},
			fieldnames.ImageRegistry:        {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.ImageRemote:          {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.ImageScanAge:         {{required: set.NewStringSet(search.ImageScanTime.String()), printer: imageScanAgePrinter}},
			fieldnames.ImageTag:             {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			fieldnames.RequiredImageLabel:   {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			fieldnames.UnscannedImage:       {{required: set.NewStringSet(augmentedobjs.ImageScanCustomTag), printer: imageScanPrinter}},
		},
	}

	requiredProcessFields = set.NewStringSet(search.ProcessName.String(), search.ProcessAncestor.String(), search.ProcessUID.String(), search.ProcessArguments.String(), augmentedobjs.NotWhitelistedCustomTag)
)

func containsAllRequiredFields(fieldMap map[string][]string, required set.StringSet) bool {
	for field := range required {
		if _, ok := fieldMap[field]; !ok {
			return false
		}
	}
	return true
}

func lookupViolationPrinters(stage storage.LifecycleStage, sectionFields set.StringSet, fieldMap map[string][]string) []printerFunc {
	var printers []printerFunc
	if printersAndFields, ok := policyFieldsToPrinters[stage]; ok {
		for field := range sectionFields {
			if printerMD, ok := printersAndFields[field]; ok {
				for _, p := range printerMD {
					if containsAllRequiredFields(fieldMap, p.required) {
						printers = append(printers, p.printer)
					}
				}
			}
		}
	}
	return printers
}

func checkForProcessViolation(result *evaluator.Result) bool {
	for _, fieldMap := range result.Matches {
		for k := range fieldMap {
			if requiredProcessFields.Contains(k) {
				return true
			}
		}
	}
	return false
}

// Render creates violation messages based on evaluation results
func Render(stage storage.LifecycleStage, section *storage.PolicySection, result *evaluator.Result, indicator *storage.ProcessIndicator) ([]*storage.Alert_Violation, bool, error) {
	errorList := errorhelpers.NewErrorList("violation printer")
	messages := set.NewStringSet()
	sectionFields := stringSetFromPolicySectionFields(section)
	for _, fieldMap := range result.Matches {
		printers := lookupViolationPrinters(stage, sectionFields, fieldMap)
		if len(printers) == 0 {
			continue
		}
		for _, printer := range printers {
			messagesForResult, err := printer(fieldMap)
			if err != nil {
				errorList.AddError(err)
				continue
			}
			messages.AddAll(messagesForResult...)
		}
	}
	isProcessViolation := indicator != nil && checkForProcessViolation(result)
	if len(messages) == 0 && !isProcessViolation {
		errorList.AddError(errors.New("missing messages"))
	}
	alertViolations := make([]*storage.Alert_Violation, 0, len(messages))
	for message := range messages {
		alertViolations = append(alertViolations, &storage.Alert_Violation{Message: message})
	}
	return alertViolations, isProcessViolation, errorList.ToError()
}
