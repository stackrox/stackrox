package violations

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// A ViolationPrinterFunc prints violation messages from a section name and map of required to values
type ViolationPrinterFunc func(string, map[string][]string) ([]string, error)

func stringSetFromMatchFields(fieldMap map[string][]string) set.StringSet {
	keys := make([]string, 0, len(fieldMap))
	for key := range fieldMap {
		keys = append(keys, key)
	}
	return set.NewStringSet(keys...)
}

func stringSetFromPolicySectionFields(section *storage.PolicySection) set.StringSet {
	sectionFields := set.NewStringSet()
	for _, group := range section.GetPolicyGroups() {
		sectionFields.Add(group.GetFieldName())
	}
	return sectionFields
}

type violationPrinter struct {
	required set.StringSet // These fields must all be in the result, and must be valid search tags
	printer  ViolationPrinterFunc
}

var (
	// TODO(rc) use appropriate constants for policy field keys
	policyFieldsToPrinters = map[storage.LifecycleStage]map[string][]violationPrinter{
		storage.LifecycleStage_DEPLOY: {
			"Add Capabilities":            {{required: set.NewStringSet(search.AddCapabilities.String()), printer: addCapabilityPrinter}},
			"CVE":                         {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"CVSS":                        {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"Container CPU Limit":         {{required: set.NewStringSet(search.CPUCoresLimit.String()), printer: resourcePrinter}},
			"Container CPU Request":       {{required: set.NewStringSet(search.CPUCoresRequest.String()), printer: resourcePrinter}},
			"Container Memory Limit":      {{required: set.NewStringSet(search.MemoryLimit.String()), printer: resourcePrinter}},
			"Container Memory Request":    {{required: set.NewStringSet(search.MemoryRequest.String()), printer: resourcePrinter}},
			"Disallowed Annotation":       {{required: set.NewStringSet(search.Annotation.String()), printer: mapPrinter}},
			"Disallowed Image Label":      {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			"Dockerfile Line":             {{required: set.NewStringSet(augmentedobjs.DockerfileLineCustomTag), printer: linePrinter}},
			"Drop Capabilities":           {{required: set.NewStringSet(search.DropCapabilities.String()), printer: dropCapabilityPrinter}},
			"Environment Variable":        {{required: set.NewStringSet(augmentedobjs.EnvironmentVarCustomTag), printer: envPrinter}},
			"Fixed By":                    {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"Image Age":                   {{required: set.NewStringSet(search.ImageCreatedTime.String()), printer: imageAgePrinter}},
			"Image Component":             {{required: set.NewStringSet(augmentedobjs.ComponentAndVersionCustomTag), printer: componentPrinter}},
			"Image Registry":              {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Image Remote":                {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Image Scan Age":              {{required: set.NewStringSet(search.ImageScanTime.String()), printer: imageScanAgePrinter}},
			"Image Tag":                   {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Minimum RBAC Permissions":    {{required: set.NewStringSet(search.ServiceAccountPermissionLevel.String()), printer: rbacPrinter}},
			"Port":                        {{required: set.NewStringSet(search.Port.String()), printer: portPrinter}},
			"Port Exposure Method":        {{required: set.NewStringSet(search.ExposureLevel.String()), printer: portExposurePrinter}},
			"Privileged":                  {{required: set.NewStringSet(search.Privileged.String()), printer: privilegedPrinter}},
			"Protocol":                    {{required: set.NewStringSet(search.Port.String()), printer: portPrinter}},
			"Read-Only Root Filesystem":   {{required: set.NewStringSet(search.ReadOnlyRootFilesystem.String()), printer: readOnlyRootFSPrinter}},
			"Required Annotation":         {{required: set.NewStringSet(search.Annotation.String()), printer: mapPrinter}},
			"Required Image Label":        {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			"Required Label":              {{required: set.NewStringSet(search.Label.String()), printer: mapPrinter}},
			"Unexpected Process Executed": {{required: set.NewStringSet(augmentedobjs.NotWhitelistedCustomTag), printer: processWhitelistPrinter}},
			"Unscanned Image":             {{required: set.NewStringSet(augmentedobjs.ImageScanCustomTag), printer: imageScanPrinter}},
			"Volume Destination":          {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			"Volume Name":                 {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			"Volume Source":               {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			"Volume Type":                 {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			"Writable Host Mount":         {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}},
			"Writable Volume":             {{required: set.NewStringSet(search.VolumeName.String()), printer: volumePrinter}}},
		storage.LifecycleStage_BUILD: {
			"CVE":                    {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"CVSS":                   {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"Disallowed Image Label": {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			"Dockerfile Line":        {{required: set.NewStringSet(augmentedobjs.DockerfileLineCustomTag), printer: linePrinter}},
			"Fixed By":               {{required: set.NewStringSet(search.CVE.String()), printer: cvePrinter}},
			"Image Age":              {{required: set.NewStringSet(search.ImageCreatedTime.String()), printer: imageAgePrinter}},
			"Image Component":        {{required: set.NewStringSet(augmentedobjs.ComponentAndVersionCustomTag), printer: componentPrinter}},
			"Image Registry":         {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Image Remote":           {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Image Scan Age":         {{required: set.NewStringSet(search.ImageScanTime.String()), printer: imageScanAgePrinter}},
			"Image Tag":              {{required: set.StringSet{}, printer: imageDetailsPrinter}},
			"Required Image Label":   {{required: set.NewStringSet(search.ImageLabel.String()), printer: mapPrinter}},
			"Unscanned Image":        {{required: set.NewStringSet(augmentedobjs.ImageScanCustomTag), printer: imageScanPrinter}}}}

	requiredProcessFields = set.NewStringSet(search.ProcessName.String(), search.ProcessAncestor.String(), search.ProcessUID.String(), search.ProcessArguments.String())
)

func lookupViolationPrinters(stage storage.LifecycleStage, sectionFields set.StringSet, fieldMap map[string][]string) []ViolationPrinterFunc {
	matchFields := stringSetFromMatchFields(fieldMap)
	var printers []ViolationPrinterFunc
	if printersAndFields, ok := policyFieldsToPrinters[stage]; ok {
		for field := range sectionFields {
			if printerMD, ok := printersAndFields[field]; ok {
				for _, p := range printerMD {
					if p.required.Cardinality() == 0 || p.required.Intersect(matchFields).Cardinality() > 0 {
						printers = append(printers, p.printer)
					}
				}
			}
		}
	}
	return printers
}

func checkForProcessViolation(section *storage.PolicySection, result *evaluator.Result) bool {
	sectionFields := stringSetFromPolicySectionFields(section)
	if requiredProcessFields.Intersect(sectionFields).Cardinality() == 0 {
		return false
	}
	for _, fieldMap := range result.Matches {
		matchFields := stringSetFromMatchFields(fieldMap)
		if requiredProcessFields.Intersect(matchFields).Cardinality() > 0 {
			return true
		}
	}
	return false
}

// ViolationPrinter creates violation messages based on evaluation results
func ViolationPrinter(stage storage.LifecycleStage, section *storage.PolicySection, result *evaluator.Result, indicator *storage.ProcessIndicator) ([]*storage.Alert_Violation, bool, error) {
	errorList := errorhelpers.NewErrorList("violation printer")
	messages := set.NewStringSet()
	sectionFields := stringSetFromPolicySectionFields(section)
	for _, fieldMap := range result.Matches {
		printers := lookupViolationPrinters(stage, sectionFields, fieldMap)
		if len(printers) == 0 {
			continue
		}
		for _, printer := range printers {
			messagesForResult, err := printer(section.GetSectionName(), fieldMap)
			if err != nil {
				errorList.AddError(err)
				continue
			}
			messages.AddAll(messagesForResult...)
		}
	}
	isProcessViolation := indicator != nil && checkForProcessViolation(section, result)
	if len(messages) == 0 && !isProcessViolation {
		errorList.AddError(errors.New("missing messages"))
	}
	alertViolations := make([]*storage.Alert_Violation, 0)
	for message := range messages {
		alertViolations = append(alertViolations, &storage.Alert_Violation{Message: message})
	}
	return alertViolations, isProcessViolation, errorList.ToError()
}
