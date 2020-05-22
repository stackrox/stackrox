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
	if p.GetPolicyVersion() == booleanpolicy.Version {
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
	return []EnvKVPair{{Key: p.GetFields().GetEnv().GetKey(), Value: p.GetFields().GetEnv().GetValue()}}
}

// GetCVEs returns the CVE fields in the given policy.
func GetCVEs(p *storage.Policy) []string {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.GetValuesWithFieldName(p, fieldnames.CVE)
	}
	return []string{p.GetFields().GetCve()}
}

// ContainsCVSSField returns whether the given policy contains a CVSS field.
func ContainsCVSSField(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.CVSS)
	}
	return p.GetFields().GetCvss() != nil
}

// GetProcessNames gets any ProcessName fields from the policy.
func GetProcessNames(p *storage.Policy) []string {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ProcessName)
	}
	return []string{p.GetFields().GetProcessPolicy().GetName()}
}

// GetImageTags gets any ImageTag fields from the policy.
func GetImageTags(p *storage.Policy) []string {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ImageTag)
	}
	return []string{p.GetFields().GetImageName().GetTag()}
}

// ContainsImageAgeField returns whether the policy contains an image age field.
func ContainsImageAgeField(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ImageAge)
	}
	return p.GetFields().GetSetImageAgeDays() != nil
}

// ContainsVolumeSourceField returns whether the policy contains a volume source field.
func ContainsVolumeSourceField(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.VolumeSource)
	}
	return p.GetFields().GetVolumePolicy().GetSource() != ""
}

// GetImageRegistries returns image registry fields from the policy.
func GetImageRegistries(p *storage.Policy) []string {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.GetValuesWithFieldName(p, fieldnames.ImageRegistry)
	}
	return []string{p.GetFields().GetImageName().GetRegistry()}
}

var (
	portOrPortExposureFields = set.NewFrozenStringSet(
		fieldnames.Port,
		fieldnames.Protocol,
		fieldnames.PortExposure,
	)
)

// ContainsPortOrPortExposureFields returns whether the policy contains any port or port exposure fields.
func ContainsPortOrPortExposureFields(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsOneOf(p, portOrPortExposureFields)
	}

	return p.GetFields().GetPortPolicy() != nil || len(p.GetFields().GetPortExposurePolicy().GetExposureLevels()) > 0
}

// ContainsCPUResourceLimit returns whether the policy contains the CPU resource limit field.
func ContainsCPUResourceLimit(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ContainerCPULimit)
	}
	return p.GetFields().GetContainerResourcePolicy().GetCpuResourceLimit() != nil
}

// ContainsMemResourceLimit returns whether the policy contains the mem resource limit field.
func ContainsMemResourceLimit(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.ContainerMemLimit)
	}
	return p.GetFields().GetContainerResourcePolicy().GetMemoryResourceLimit() != nil
}

// ContainsUnscannedImageField returns whether the policy contains the unscanned image field.
func ContainsUnscannedImageField(p *storage.Policy) bool {
	if p.GetPolicyVersion() == booleanpolicy.Version {
		return booleanpolicy.ContainsValueWithFieldName(p, fieldnames.UnscannedImage)
	}
	return p.GetFields().GetNoScanExists()
}
