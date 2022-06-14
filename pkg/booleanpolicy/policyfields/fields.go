package policyfields

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/set"
)

// EnvKVPair represents an EnvKVPair defined in a policy.
// It exists solely to solve as the return type of GetEnvKeyValues.
type EnvKVPair struct {
	Key   string
	Value string
}

// GetEnvKeyValues gets env key values from a policy.
func GetEnvKeyValues(p *storage.Policy) []EnvKVPair {
	var pairs []EnvKVPair
	booleanpolicy.ForEachValueWithFieldName(p, fieldnames.EnvironmentVariable, func(value string) bool {
		splitValue := strings.Split(value, "=")
		var envKey, envValue string
		if len(splitValue) > 1 {
			envKey = splitValue[1]
		}
		if len(splitValue) > 2 {
			envValue = splitValue[2]
		}
		pairs = append(pairs, EnvKVPair{Key: envKey, Value: envValue})
		return true
	})
	return pairs

}

// GetCVEs returns the CVE fields in the given policy.
func GetCVEs(p *storage.Policy) []string {
	return booleanpolicy.GetValuesWithFieldName(p, fieldnames.CVE)
}

// ContainsCVSSField returns whether the given policy contains a CVSS field.
func ContainsCVSSField(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.CVSS)
}

// ContainsSeverityField returns whether the given policy contains a Severity field.
func ContainsSeverityField(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.Severity)
}

// GetProcessNames gets any ProcessName fields from the policy.
func GetProcessNames(p *storage.Policy) []string {
	return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ProcessName)
}

// GetImageTags gets any ImageTag fields from the policy.
func GetImageTags(p *storage.Policy) []string {
	return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ImageTag)

}

// ContainsImageAgeField returns whether the policy contains an image age field.
func ContainsImageAgeField(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ImageAge)
}

// ContainsVolumeSourceField returns whether the policy contains a volume source field.
func ContainsVolumeSourceField(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.VolumeSource)
}

// GetImageRegistries returns image registry fields from the policy.
func GetImageRegistries(p *storage.Policy) []string {
	return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ImageRegistry)
}

var (
	portOrPortExposureFields = set.NewFrozenStringSet(
		fieldnames.ExposedPort,
		fieldnames.ExposedPortProtocol,
		fieldnames.PortExposure,
	)
)

// ContainsPortOrPortExposureFields returns whether the policy contains any port or port exposure fields.
func ContainsPortOrPortExposureFields(p *storage.Policy) bool {
	for _, section := range p.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if portOrPortExposureFields.Contains(group.GetFieldName()) {
				return true
			}
		}
	}
	return false
}

// ContainsCPUResourceLimit returns whether the policy contains the CPU resource limit field.
func ContainsCPUResourceLimit(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ContainerCPULimit)
}

// ContainsMemResourceLimit returns whether the policy contains the mem resource limit field.
func ContainsMemResourceLimit(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ContainerMemLimit)
}

// ContainsScanRequiredFields returns whether the policy contains fields related to image scanning,
// which require a scan result and may otherwise fail, i.e. fieldnames.UnscannedImage or
// fieldnames.ImageSignatureVerifiedBy.
func ContainsScanRequiredFields(p *storage.Policy) bool {
	return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.UnscannedImage) ||
		booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ImageSignatureVerifiedBy)
}
