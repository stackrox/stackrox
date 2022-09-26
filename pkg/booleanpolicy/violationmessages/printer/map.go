package printer

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	mapRequiredTemplate   = `Required {{.ResourceName}} not found (found {{.ResourceName}}s: {{.Value}})`
	mapDisallowedTemplate = `Disallowed {{.ResourceName}}s found: {{.Value}}`
)

func requiredLabelPrinter(fieldMap map[string][]string) ([]string, error) {
	return getMapPrinterFor(search.DeploymentLabel, false)(fieldMap)
}

func requiredAnnotationPrinter(fieldMap map[string][]string) ([]string, error) {
	return getMapPrinterFor(search.DeploymentAnnotation, false)(fieldMap)
}

func requiredImageLabelPrinter(fieldMap map[string][]string) ([]string, error) {
	return getMapPrinterFor(search.ImageLabel, false)(fieldMap)
}

func disallowedImageLabelPrinter(fieldMap map[string][]string) ([]string, error) {
	return getMapPrinterFor(search.ImageLabel, true)(fieldMap)
}

func disallowedAnnotationPrinter(fieldMap map[string][]string) ([]string, error) {
	return getMapPrinterFor(search.DeploymentAnnotation, true)(fieldMap)
}

func getMapPrinterFor(fieldLabel search.FieldLabel, disallowed bool) func(map[string][]string) ([]string, error) {
	var baseResourceName string
	switch fieldLabel {
	case search.DeploymentAnnotation:
		baseResourceName = "annotation"
	case search.DeploymentLabel:
		baseResourceName = "label"
	case search.ImageLabel:
		baseResourceName = "label"
	default:
		// Panic here is okay, since this function is called at program-init time.
		utils.CrashOnError(errors.Errorf("unknown field label: %v", fieldLabel))
	}
	return func(fieldMap map[string][]string) ([]string, error) {
		type resultFields struct {
			ResourceName string
			Value        string
		}
		var r *resultFields
		if values, ok := fieldMap[fieldLabel.String()]; ok {
			r = &resultFields{ResourceName: baseResourceName, Value: strings.Join(values, "; ")}
		}
		if r == nil {
			return nil, nil
		}
		if disallowed {
			return executeTemplate(mapDisallowedTemplate, r)
		}
		return executeTemplate(mapRequiredTemplate, r)
	}
}
