package violations

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

func readOnlyRootFSPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName  string
		ReadOnlyRootFS bool
	}
	msgTemplate := `Container {{if .ContainerName}}'{{.ContainerName}}'{{end}} 
	{{- if .ReadOnlyRootFS }} using read-only root filesystem{{else}} using read-write root filesystem{{end}}`
	r := resultFields{}
	var err error
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	readOnlyRootFS, err := getSingleValueFromFieldMap(search.ReadOnlyRootFilesystem.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.ReadOnlyRootFS, err = strconv.ParseBool(readOnlyRootFS); err != nil {
		return nil, err
	}
	return executeTemplate(msgTemplate, r)
}

func imageAgePrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName     string
		ImageCreationTime string
	}
	msgTemplate := "{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} time of creation {{.ImageCreationTime}}"
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageCreationTime, err = getSingleValueFromFieldMap(search.ImageCreatedTime.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(msgTemplate, r)
}

func imageScanAgePrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageScanTime string
	}
	msgTemplate := `{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} time of last scan {{.ImageScanTime}}`
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageScanTime, err = getSingleValueFromFieldMap(search.ImageScanTime.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(msgTemplate, r)
}

func imageDetailsPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageDetails  string
	}
	msgTemplate := "{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} {{.ImageDetails}}"
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	imageDetails := make([]string, 0)
	if imageTag, err := getSingleValueFromFieldMap(search.ImageTag.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("tag '%s'", imageTag))
	}
	if imageRemote, err := getSingleValueFromFieldMap(search.ImageRemote.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("remote '%s'", imageRemote))
	}
	if imageRegistry, err := getSingleValueFromFieldMap(search.ImageRegistry.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("registry '%s'", imageRegistry))
	}
	if len(imageDetails) == 0 {
		return nil, errors.New("missing image details")
	}
	r.ImageDetails = stringSliceToSortedSentence(imageDetails)
	return executeTemplate(msgTemplate, r)
}

// Print violation message for match against policyFieldsToPrinters privileged container
func privilegedPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Privileged    bool
	}
	msgTemplate := "Container{{if .ContainerName}} '{{.ContainerName}}'{{end}} is{{if not .Privileged}} not{{end}} privileged"
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	privileged, err := getSingleValueFromFieldMap(search.Privileged.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.Privileged, err = strconv.ParseBool(privileged); err != nil {
		return nil, err
	}
	return executeTemplate(msgTemplate, r)
}
