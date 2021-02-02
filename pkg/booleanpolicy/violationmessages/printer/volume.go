package printer

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	volumeTemplate = `{{- if .ReadOnly }}Read-only{{else}}Writable{{end}} volume '{{- .VolumeName}}' has {{ .VolumeDetails }}`
)

func volumePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ReadOnly      bool
		VolumeName    string
		VolumeDetails string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.VolumeName = maybeGetSingleValueFromFieldMap(search.VolumeName.String(), fieldMap)
	if readOnly, err := getSingleValueFromFieldMap(search.VolumeReadonly.String(), fieldMap); err == nil {
		r.ReadOnly, _ = strconv.ParseBool(readOnly)
	}
	volumeDetails := make([]string, 0)
	if source, err := getSingleValueFromFieldMap(search.VolumeSource.String(), fieldMap); err == nil && source != "" {
		volumeDetails = append(volumeDetails, fmt.Sprintf("source '%s'", source))
	}
	if dest, err := getSingleValueFromFieldMap(search.VolumeDestination.String(), fieldMap); err == nil && dest != "" {
		volumeDetails = append(volumeDetails, fmt.Sprintf("destination '%s'", dest))
	}
	if volumeType, err := getSingleValueFromFieldMap(search.VolumeType.String(), fieldMap); err == nil && volumeType != "" {
		volumeDetails = append(volumeDetails, fmt.Sprintf("type '%s'", volumeType))
	}
	if mountPropagation, err := getSingleValueFromFieldMap(search.MountPropagation.String(), fieldMap); err == nil && mountPropagation != "" {
		volumeDetails = append(volumeDetails, fmt.Sprintf("mount propagation '%s'", mountPropagation))
	}
	if len(volumeDetails) == 0 {
		return nil, errors.New("missing volume details")
	}
	r.VolumeDetails = stringSliceToSentence(volumeDetails)

	return executeTemplate(volumeTemplate, r)
}
