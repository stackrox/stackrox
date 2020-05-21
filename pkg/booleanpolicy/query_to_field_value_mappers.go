package booleanpolicy

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/set"
)

var (
	qToFieldValueMappers = map[search.FieldLabel]*fieldNameAndFVB{
		search.ComponentVersion:              newMapper(ImageComponent, rightCompoundMap),
		search.DockerfileInstructionKeyword:  newMapper(DockerfileLine, leftCompoundMap),
		search.DockerfileInstructionValue:    newMapper(DockerfileLine, rightCompoundMap),
		search.EnvironmentKey:                newMapper(EnvironmentVariable, envKeyCompoundMap),
		search.EnvironmentValue:              newMapper(EnvironmentVariable, envValCompoundMap),
		search.EnvironmentVarSrc:             newMapper(EnvironmentVariable, envSrcCompoundMap),
		search.Annotation:                    newMapper(DisallowedAnnotation, directMap),
		search.ImageLabel:                    newMapper(DisallowedImageLabel, directMap),
		search.VolumeReadonly:                newMapper(WritableVolume, invertBooleanMap),
		search.ImageCreatedTime:              newMapper(ImageAge, numberOfDaysSinceMap),
		search.ImageScanTime:                 newMapper(ImageScanAge, numberOfDaysSinceMap),
		search.ServiceAccountPermissionLevel: newMapper(MinimumRBACPermissions, serviceAccountPermissionLevelMap),
		search.ExposureLevel:                 newMapper(PortExposure, directMap),
		search.AddCapabilities:               newMapper(AddCaps, directMap),
		search.CVE:                           newMapper(CVE, directMap),
		search.CVSS:                          newMapper(CVSS, criteriaMap),
		search.CPUCoresLimit:                 newMapper(ContainerCPULimit, criteriaMap),
		search.CPUCoresRequest:               newMapper(ContainerCPURequest, criteriaMap),
		search.MemoryLimit:                   newMapper(ContainerMemLimit, criteriaMap),
		search.MemoryRequest:                 newMapper(ContainerMemRequest, criteriaMap),
		search.FixedBy:                       newMapper(FixedBy, directMap),
		search.Component:                     newMapper(ImageComponent, leftCompoundMap),
		search.ImageRegistry:                 newMapper(ImageRegistry, directMap),
		search.ImageRemote:                   newMapper(ImageRemote, directMap),
		search.ImageTag:                      newMapper(ImageTag, directMap),
		search.Port:                          newMapper(Port, directMap),
		search.Privileged:                    newMapper(Privileged, directMap),
		search.ProcessAncestor:               newMapper(ProcessAncestor, directMap),
		search.ProcessArguments:              newMapper(ProcessArguments, directMap),
		search.ProcessName:                   newMapper(ProcessName, directMap),
		search.ProcessUID:                    newMapper(ProcessUID, directMap),
		search.PortProtocol:                  newMapper(Protocol, directMap),
		search.ReadOnlyRootFilesystem:        newMapper(ReadOnlyRootFS, directMap),
		search.VolumeDestination:             newMapper(VolumeDestination, directMap),
		search.VolumeName:                    newMapper(VolumeName, directMap),
		search.VolumeSource:                  newMapper(VolumeSource, directMap),
		search.VolumeType:                    newMapper(VolumeType, directMap),
	}
)

type queryToFieldValueMapper func(searchTerms []string) (policyTerms []string, criteriaChanged bool)

type fieldNameAndFVB struct {
	fieldName        string
	fieldValueMapper queryToFieldValueMapper
}

func newMapper(fieldName string, fvb queryToFieldValueMapper) *fieldNameAndFVB {
	return &fieldNameAndFVB{
		fieldName:        fieldName,
		fieldValueMapper: fvb,
	}
}

func directMap(searchTerms []string) ([]string, bool) {
	return searchTerms, false
}

func leftCompoundMap(searchTerms []string) ([]string, bool) {
	// This maps the given search terms to the left side of a two-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = term + "="
	}
	return mapped, false
}

func rightCompoundMap(searchTerms []string) ([]string, bool) {
	// This maps the given search terms to the right side of a two-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = "=" + term
	}
	return mapped, false
}

func envKeyCompoundMap(searchTerms []string) ([]string, bool) {
	// Environment Variable Key should be mapped to the left side of a three-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = term + "=="
	}
	return mapped, false
}

func envValCompoundMap(searchTerms []string) ([]string, bool) {
	// Environment Variable Value should be mapped to the middle of a three-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = "=" + term + "="
	}
	return mapped, false
}

func envSrcCompoundMap(searchTerms []string) ([]string, bool) {
	// Environment Variable Source should be mapped to the right side of a three-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = "==" + term
	}
	return mapped, false
}

func numberOfDaysSinceMap(searchTerms []string) ([]string, bool) {
	var policyCriteria []string
	droppedTerms := false
	for _, timeString := range searchTerms {
		// Only convert time searches of the format ">NUMBERd"
		if !strings.HasPrefix(timeString, basematchers.GreaterThan) {
			droppedTerms = true
			continue
		}
		if !strings.HasSuffix(strings.ToLower(timeString), "d") {
			droppedTerms = true
			continue
		}
		// Trim off the >, the d, and any whitespace
		policyCriteria = append(policyCriteria, strings.TrimSpace(timeString[1:len(timeString)-1]))
	}
	return policyCriteria, droppedTerms
}

func serviceAccountPermissionLevelMap(searchTerms []string) ([]string, bool) {
	// Service Account Permission Level is an exact match in search and a >= match in policies.
	// This should take the minimum specified permission level.
	termSet := set.NewStringSet(searchTerms...)
	for i := 0; i < len(storage.PermissionLevel_name); i++ {
		name := storage.PermissionLevel_name[int32(i)]
		if termSet.Contains(name) {
			return []string{name}, true
		}
	}
	return nil, false
}

func invertBooleanMap(searchTerms []string) ([]string, bool) {
	var policyTerms []string
	droppedTerms := false
	for _, searchTerm := range searchTerms {
		if strings.ToLower(searchTerm) == "true" {
			policyTerms = append(policyTerms, "false")
			continue
		} else if strings.ToLower(searchTerm) == "false" {
			policyTerms = append(policyTerms, "true")
			continue
		}
		droppedTerms = true
	}
	return policyTerms, droppedTerms
}

func criteriaMap(searchTerms []string) ([]string, bool) {
	// criteria search terms have the form '>N' while criteria policy terms have the form '> N'.  Both use an implicit
	// '=' if there is no specified criteria.
	policyTerms := make([]string, 0, len(searchTerms))
	for _, searchTerm := range searchTerms {
		content := searchTerm
		var prefix string
		if strings.HasPrefix(searchTerm, basematchers.GreaterThanOrEqualTo) {
			prefix = basematchers.GreaterThanOrEqualTo + " "
			content = searchTerm[2:]
		} else if strings.HasPrefix(searchTerm, basematchers.LessThanOrEqualTo) {
			prefix = basematchers.LessThanOrEqualTo + " "
			content = searchTerm[2:]
		} else if strings.HasPrefix(searchTerm, basematchers.GreaterThan) {
			prefix = basematchers.GreaterThan + " "
			content = searchTerm[1:]
		} else if strings.HasPrefix(searchTerm, basematchers.LessThan) {
			prefix = basematchers.LessThan + " "
			content = searchTerm[1:]
		}
		content = strings.TrimSpace(content)
		policyTerms = append(policyTerms, prefix+content)
	}
	return policyTerms, false
}

// GetPolicyGroupFromSearchTerms accepts a field label and a list of search terms, and turns them into a policy group
func GetPolicyGroupFromSearchTerms(fieldLabel search.FieldLabel, searchTerms []string) (*storage.PolicyGroup, bool, bool) {
	qvm, ok := qToFieldValueMappers[fieldLabel]
	if !ok {
		return nil, false, false
	}

	group := &storage.PolicyGroup{
		FieldName:       qvm.fieldName,
		BooleanOperator: storage.BooleanOperator_OR,
	}

	fieldValues, fieldsDropped := qvm.fieldValueMapper(searchTerms)
	if len(fieldValues) == 0 {
		return nil, false, true
	}
	for _, value := range fieldValues {
		group.Values = append(group.GetValues(), &storage.PolicyValue{
			Value: value,
		})
	}
	return group, fieldsDropped, true
}
