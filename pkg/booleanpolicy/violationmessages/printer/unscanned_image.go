package printer

import (
	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
)

const (
	imageScanTemplate = `{{if .ContainerName}}Image in container '{{.ContainerName}}'{{else}}Image{{end}} has {{if not .Scanned}}not {{end}}been scanned`
)

func imageScanPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Scanned       bool
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	imageScan, err := getSingleValueFromFieldMap(augmentedobjs.ImageScanCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	if imageScan != "<nil>" {
		r.Scanned = true
	}
	return executeTemplate(imageScanTemplate, r)
}
