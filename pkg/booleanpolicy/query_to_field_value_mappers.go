package booleanpolicy

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/set"
)

var (
	qToFieldValueMappers = map[search.FieldLabel]*fieldNameAndFVB{
		search.ComponentVersion:              newMapper(fieldnames.ImageComponent, rightCompoundMap),
		search.DockerfileInstructionKeyword:  newMapper(fieldnames.DockerfileLine, leftCompoundMap),
		search.DockerfileInstructionValue:    newMapper(fieldnames.DockerfileLine, rightCompoundMap),
		search.EnvironmentKey:                newMapper(fieldnames.EnvironmentVariable, envKeyCompoundMap),
		search.EnvironmentValue:              newMapper(fieldnames.EnvironmentVariable, envValCompoundMap),
		search.EnvironmentVarSrc:             newMapper(fieldnames.EnvironmentVariable, envSrcCompoundMap),
		search.DeploymentAnnotation:          newMapper(fieldnames.DisallowedAnnotation, leftRightCompoundMap),
		search.ImageLabel:                    newMapper(fieldnames.DisallowedImageLabel, leftRightCompoundMap),
		search.VolumeReadonly:                newMapper(fieldnames.WritableMountedVolume, invertBooleanMap),
		search.ImageCreatedTime:              newMapper(fieldnames.ImageAge, numberOfDaysSinceMap),
		search.ImageScanTime:                 newMapper(fieldnames.ImageScanAge, numberOfDaysSinceMap),
		search.ServiceAccountPermissionLevel: newMapper(fieldnames.MinimumRBACPermissions, serviceAccountPermissionLevelMap),
		search.ExposureLevel:                 newMapper(fieldnames.PortExposure, directMap),
		search.AddCapabilities:               newMapper(fieldnames.AddCaps, directMap),
		search.CVE:                           newMapper(fieldnames.CVE, directMap),
		search.CVSS:                          newMapper(fieldnames.CVSS, criteriaMap),
		search.CPUCoresLimit:                 newMapper(fieldnames.ContainerCPULimit, criteriaMap),
		search.CPUCoresRequest:               newMapper(fieldnames.ContainerCPURequest, criteriaMap),
		search.MemoryLimit:                   newMapper(fieldnames.ContainerMemLimit, criteriaMap),
		search.MemoryRequest:                 newMapper(fieldnames.ContainerMemRequest, criteriaMap),
		search.FixedBy:                       newMapper(fieldnames.FixedBy, directMap),
		search.Component:                     newMapper(fieldnames.ImageComponent, leftCompoundMap),
		search.ImageRegistry:                 newMapper(fieldnames.ImageRegistry, directMap),
		search.ImageRemote:                   newMapper(fieldnames.ImageRemote, directMap),
		search.ImageTag:                      newMapper(fieldnames.ImageTag, directMap),
		search.Port:                          newMapper(fieldnames.ExposedPort, directMap),
		search.Privileged:                    newMapper(fieldnames.PrivilegedContainer, directMap),
		search.ProcessAncestor:               newMapper(fieldnames.ProcessAncestor, directMap),
		search.ProcessArguments:              newMapper(fieldnames.ProcessArguments, directMap),
		search.ProcessName:                   newMapper(fieldnames.ProcessName, directMap),
		search.ProcessUID:                    newMapper(fieldnames.ProcessUID, directMap),
		search.PortProtocol:                  newMapper(fieldnames.ExposedPortProtocol, directMap),
		search.ReadOnlyRootFilesystem:        newMapper(fieldnames.ReadOnlyRootFS, directMap),
		search.VolumeDestination:             newMapper(fieldnames.VolumeDestination, directMap),
		search.VolumeName:                    newMapper(fieldnames.VolumeName, directMap),
		search.VolumeSource:                  newMapper(fieldnames.VolumeSource, directMap),
		search.VolumeType:                    newMapper(fieldnames.VolumeType, directMap),
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
		mapped[i] = "=" + term + "="
	}
	return mapped, false
}

func envValCompoundMap(searchTerms []string) ([]string, bool) {
	// Environment Variable Value should be mapped to the middle of a three-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = "==" + term
	}
	return mapped, false
}

func envSrcCompoundMap(searchTerms []string) ([]string, bool) {
	// Environment Variable Source should be mapped to the right side of a three-part compound policy field value
	mapped := make([]string, len(searchTerms))
	for i, term := range searchTerms {
		mapped[i] = term + "=="
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

func leftRightCompoundMap(searchTerms []string) ([]string, bool) {
	var mustBeCompound []string
	for _, term := range searchTerms {
		if !strings.Contains(term, "=") {
			term = term + "="
		}
		mustBeCompound = append(mustBeCompound, term)
	}

	return mustBeCompound, false
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
