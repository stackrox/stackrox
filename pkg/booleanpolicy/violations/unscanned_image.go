package violations

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

func imageScanPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
	}
	msgTemplate := "{{if .ContainerName}}Image in container '{{.ContainerName}}'{{else}}Image{{end}} has not been scanned"
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	imageScan, err := getSingleValueFromFieldMap(augmentedobjs.ImageScanCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	if imageScan != "<nil>" {
		return nil, errors.New("image has been scanned")
	}
	return executeTemplate(msgTemplate, r)
}
