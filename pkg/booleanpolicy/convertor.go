package booleanpolicy

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

type individualFieldConverter func(fields *storage.PolicyFields) []*storage.PolicyGroup

var andFieldsConverters = []individualFieldConverter{
	convertImageNamePolicy,
	convertImageAgeDays,
	convertDockerFileLineRule,
	convertCve,
	convertComponent,
	convertImageScanAge,
	convertNoScanExists,
	convertEnv,
	convertVolumePolicy,
	convertPortPolicy,
	convertRequiredLabel,
	convertRequiredAnnotation,
	convertDisallowedAnnotation,
	convertRequiredImageLabel,
	convertDisallowedImageLabel,
	convertPrivileged,
	convertProcessPolicy,
	convertHostMountPolicy,
	convertWhitelistEnabled,
	convertFixedBy,
	convertReadOnlyRootFs,
	convertCvss,
	convertDropCapabilities,
	convertAddCapabilities,
	convertPermissionPolicy,
	convertExposureLevelPolicy,
}

// EnsureConverted converts the given policy into a Boolean policy, if it is not one already.
func EnsureConverted(p *storage.Policy) error {
	if p == nil {
		return errors.New("nil policy")
	}
	if p.GetPolicyVersion() != legacyVersion && p.GetPolicyVersion() != Version {
		return errors.New("unknown version")
	}
	if p.GetPolicyVersion() == Version && len(p.GetPolicySections()) == 0 {
		return errors.New("empty sections")
	}
	if p.GetPolicyVersion() == legacyVersion {
		// If a policy is sent with legacyVersion, but contains sections, that's okay -- we will use those sections
		// as-is, and infer that it's of the new Version.
		if p.GetFields() == nil && len(p.GetPolicySections()) == 0 {
			return errors.New("empty policy")
		}
	}
	if p.GetPolicyVersion() == legacyVersion {
		p.PolicyVersion = Version
		p.PolicySections = append(p.PolicySections, ConvertPolicyFieldsToSections(p.GetFields())...)
		p.Fields = nil
	}
	return nil
}

// CloneAndEnsureConverted returns a clone of the input that is upgraded if it is a legacy policy
func CloneAndEnsureConverted(p *storage.Policy) (*storage.Policy, error) {
	cloned := p.Clone()
	if err := EnsureConverted(cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

// ConvertPolicyFieldsToSections converts policy fields (version = "") to policy sections (version = "2.0").
func ConvertPolicyFieldsToSections(fields *storage.PolicyFields) []*storage.PolicySection {
	var andGroups []*storage.PolicyGroup
	for _, fieldConverter := range andFieldsConverters {
		andGroups = append(andGroups, fieldConverter(fields)...)
	}

	orGroups := convertContainerResourcePolicy(fields)

	if len(andGroups) == 0 && len(orGroups) == 0 {
		return nil
	}

	if len(orGroups) == 0 {
		return []*storage.PolicySection{
			{
				PolicyGroups: andGroups,
			},
		}
	}

	// Legacy container resource policies are implicitly ORd together.  For some policy term A and some resource policy
	// terms B and C a legacy policy implements the logic "A AND (B OR C)".  To implement this in boolean policies we
	// have to create multiple policy sections, each containing all of the AND search terms and one of the OR search
	// terms for "(A AND B) OR (A AND C)"
	var sections []*storage.PolicySection
	for _, orGroup := range orGroups {
		section := &storage.PolicySection{
			PolicyGroups: make([]*storage.PolicyGroup, 0, len(andGroups)+1),
		}
		for _, andGroup := range andGroups {
			section.PolicyGroups = append(section.PolicyGroups, andGroup.Clone())
		}
		section.PolicyGroups = append(section.PolicyGroups, orGroup)
		sections = append(sections, section)
	}

	return sections
}

func convertImageScanAge(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetScanAgeDays() == nil {
		return nil
	}

	return []*storage.PolicyGroup{
		{
			FieldName: ImageScanAge,
			Values:    getPolicyValues(fields.GetScanAgeDays()),
		},
	}
}

func convertNoScanExists(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetNoScanExists() == nil {
		return nil
	}

	return []*storage.PolicyGroup{
		{
			FieldName: UnscannedImage,
			Values:    getPolicyValues(fields.GetNoScanExists()),
		},
	}
}

func convertEnv(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetEnv()
	if p == nil {
		return nil
	}

	return []*storage.PolicyGroup{
		{
			FieldName: EnvironmentVariable,
			Values:    getPolicyValues(fmt.Sprintf("%s=%s=%s", p.GetEnvVarSource(), p.GetKey(), p.GetValue())),
		},
	}
}

func convertRequiredLabel(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if p := convertKeyValuePolicy(fields.GetRequiredLabel(), RequiredLabel); p != nil {
		return []*storage.PolicyGroup{p}
	}

	return nil
}

func convertRequiredAnnotation(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if p := convertKeyValuePolicy(fields.GetRequiredAnnotation(), RequiredAnnotation); p != nil {
		return []*storage.PolicyGroup{p}
	}

	return nil
}

func convertDisallowedAnnotation(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if p := convertKeyValuePolicy(fields.GetDisallowedAnnotation(), DisallowedAnnotation); p != nil {
		return []*storage.PolicyGroup{p}
	}

	return nil
}

func convertRequiredImageLabel(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if p := convertKeyValuePolicy(fields.GetRequiredImageLabel(), RequiredImageLabel); p != nil {
		return []*storage.PolicyGroup{p}
	}

	return nil
}

func convertDisallowedImageLabel(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if p := convertKeyValuePolicy(fields.GetDisallowedImageLabel(), DisallowedImageLabel); p != nil {
		return []*storage.PolicyGroup{p}
	}

	return nil
}

func convertPrivileged(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetPrivileged() == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: Privileged,
		Values:    getPolicyValues(fields.GetPrivileged()),
	},
	}
}

func convertWhitelistEnabled(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetWhitelist() == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: WhitelistsEnabled,
		Values:    getPolicyValues(fields.GetWhitelistEnabled()),
	}}
}

func convertFixedBy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetFixedBy()
	if p == "" {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: FixedBy,
		Values:    getPolicyValues(p),
	}}
}

func convertReadOnlyRootFs(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetReadOnlyRootFs() == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: ReadOnlyRootFS,
		Values:    getPolicyValues(fields.GetReadOnlyRootFs()),
	}}
}

func getStringListPolicyValues(p []string) []*storage.PolicyValue {
	ifaceSlice := make([]interface{}, len(p))
	for i, pval := range p {
		ifaceSlice[i] = pval
	}
	return getPolicyValues(ifaceSlice...)
}

func getPolicyValues(p ...interface{}) []*storage.PolicyValue {
	vs := make([]*storage.PolicyValue, 0, len(p))
	for _, v := range p {
		switch val := v.(type) {
		case string:
			vs = append(vs, &storage.PolicyValue{Value: val})
		case int64:
			vs = append(vs, &storage.PolicyValue{Value: strconv.FormatInt(val, 10)})
		case bool:
			vs = append(vs, &storage.PolicyValue{Value: strconv.FormatBool(val)})
		default:
			utils.Should(errors.Errorf("invalid policy type: %T", val))
		}
	}

	if len(vs) == 0 {
		return nil
	}

	return vs
}

func convertImageNamePolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetImageName()
	if p == nil {
		return nil
	}

	var res []*storage.PolicyGroup
	if p.GetRegistry() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ImageRegistry,
			Values:    getPolicyValues(p.GetRegistry()),
		})
	}

	if p.GetRemote() != "" {
		actualValue := fmt.Sprintf("%s.*%s.*", search.RegexPrefix, p.GetRemote())
		res = append(res, &storage.PolicyGroup{
			FieldName: ImageRemote,
			Values:    getPolicyValues(actualValue),
		})
	}

	if p.GetTag() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ImageTag,
			Values:    getPolicyValues(p.GetTag()),
		})
	}

	return res
}

func convertImageAgeDays(fields *storage.PolicyFields) []*storage.PolicyGroup {
	if fields.GetSetImageAgeDays() == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: ImageAge,
		Values:    getPolicyValues(fields.GetImageAgeDays()),
	}}
}

func convertDockerFileLineRule(fields *storage.PolicyFields) []*storage.PolicyGroup {
	lineRule := fields.GetLineRule()
	if lineRule == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: DockerfileLine,
		Values:    getPolicyValues(fmt.Sprintf("%s=%s", lineRule.GetInstruction(), lineRule.GetValue())),
	}}
}

func convertCvss(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetCvss()
	if p == nil {
		return nil
	}

	return []*storage.PolicyGroup{convertNumericalPolicy(p, CVSS)}
}

func convertCve(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetCve()
	if p == "" {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: CVE,
		Values:    getPolicyValues(p),
	}}
}

func convertNumericalPolicy(p *storage.NumericalPolicy, fieldName string) *storage.PolicyGroup {
	if p == nil {
		return nil
	}

	op := p.GetOp().String()
	opWhitespace := " "
	switch p.GetOp() {
	case storage.Comparator_EQUALS:
		op = ""
		opWhitespace = ""
	case storage.Comparator_GREATER_THAN:
		op = ">"
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		op = ">="
	case storage.Comparator_LESS_THAN:
		op = "<"
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		op = "<="
	default:
		utils.Should(errors.Errorf("invalid op for numerical policy: %+v", p))
	}

	return &storage.PolicyGroup{
		FieldName: fieldName,
		Values: []*storage.PolicyValue{
			{
				Value: fmt.Sprintf("%s%s%f", op, opWhitespace, p.GetValue()),
			},
		},
	}
}

func convertComponent(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetComponent()
	if p == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: ImageComponent,
		Values: []*storage.PolicyValue{
			{
				Value: fmt.Sprintf("%s=%s", p.GetName(), p.GetVersion()),
			},
		},
	}}
}

func convertKeyValuePolicy(p *storage.KeyValuePolicy, fieldName string) *storage.PolicyGroup {
	if p == nil {
		return nil
	}

	return &storage.PolicyGroup{
		FieldName: fieldName,
		Values: []*storage.PolicyValue{
			{
				Value: fmt.Sprintf("%s=%s", p.GetKey(), p.GetValue()),
			},
		},
	}
}

func convertVolumePolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetVolumePolicy()
	if p == nil {
		return nil
	}

	var res []*storage.PolicyGroup
	if p.GetName() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: VolumeName,
			Values:    getPolicyValues(p.GetName()),
		})
	}

	if p.GetType() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: VolumeType,
			Values:    getPolicyValues(p.GetType()),
		})
	}

	if p.GetDestination() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: VolumeDestination,
			Values:    getPolicyValues(p.GetDestination()),
		})
	}

	if p.GetSource() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: VolumeSource,
			Values:    getPolicyValues(p.GetSource()),
		})
	}

	ro := p.GetSetReadOnly()
	if ro != nil {
		res = append(res, &storage.PolicyGroup{
			FieldName: WritableVolume,
			Values:    getPolicyValues(!p.GetReadOnly()),
		})
	}

	return res
}

func convertPortPolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetPortPolicy()
	if p == nil {
		return nil
	}

	var res []*storage.PolicyGroup
	if p.GetPort() != 0 {
		res = append(res, &storage.PolicyGroup{
			FieldName: Port,
			Values:    getPolicyValues(int64(p.GetPort())),
		})
	}

	if p.GetProtocol() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: Protocol,
			Values:    getPolicyValues(p.GetProtocol()),
		})
	}

	return res
}

func convertProcessPolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetProcessPolicy()
	if p == nil {
		return nil
	}

	var res []*storage.PolicyGroup
	if p.GetName() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ProcessName,
			Values:    getPolicyValues(p.GetName()),
		})
	}

	if p.GetAncestor() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ProcessAncestor,
			Values:    getPolicyValues(p.GetAncestor()),
		})
	}

	if p.GetArgs() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ProcessArguments,
			Values:    getPolicyValues(p.GetArgs()),
		})
	}

	if p.GetUid() != "" {
		res = append(res, &storage.PolicyGroup{
			FieldName: ProcessUID,
			Values:    getPolicyValues(p.GetUid()),
		})
	}

	return res
}

func convertHostMountPolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	p := fields.GetHostMountPolicy()
	if p.GetSetReadOnly() == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName: WritableHostMount,
		Values:    getPolicyValues(!p.GetReadOnly()),
	},
	}
}

func convertDropCapabilities(fields *storage.PolicyFields) []*storage.PolicyGroup {
	dropCaps := fields.GetDropCapabilities()
	if dropCaps == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName:       DropCaps,
		BooleanOperator: storage.BooleanOperator_OR,
		Values:          getStringListPolicyValues(dropCaps),
	}}
}

func convertAddCapabilities(fields *storage.PolicyFields) []*storage.PolicyGroup {
	addCaps := fields.GetAddCapabilities()
	if addCaps == nil {
		return nil
	}

	return []*storage.PolicyGroup{{
		FieldName:       AddCaps,
		BooleanOperator: storage.BooleanOperator_OR,
		Values:          getStringListPolicyValues(addCaps),
	}}
}

func convertContainerResourcePolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	resPolicy := fields.GetContainerResourcePolicy()
	if resPolicy == nil {
		return nil
	}

	var res []*storage.PolicyGroup
	if resPolicy.GetCpuResourceLimit() != nil {
		res = append(res, convertNumericalPolicy(resPolicy.GetCpuResourceLimit(), ContainerCPULimit))
	}
	if resPolicy.GetCpuResourceRequest() != nil {
		res = append(res, convertNumericalPolicy(resPolicy.GetCpuResourceRequest(), ContainerCPURequest))
	}
	if resPolicy.GetMemoryResourceLimit() != nil {
		res = append(res, convertNumericalPolicy(resPolicy.GetMemoryResourceLimit(), ContainerMemLimit))
	}
	if resPolicy.GetMemoryResourceRequest() != nil {
		res = append(res, convertNumericalPolicy(resPolicy.GetMemoryResourceRequest(), ContainerMemRequest))
	}
	return res
}

func convertPermissionPolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	perPolicy := fields.GetPermissionPolicy()
	if perPolicy == nil {
		return nil
	}

	permissionLevel, err := getRBACPermissionName(perPolicy.GetPermissionLevel())
	if err != nil {
		return nil
	}

	return []*storage.PolicyGroup{
		{
			FieldName: MinimumRBACPermissions,
			Values:    getPolicyValues(permissionLevel),
		},
	}
}

func getRBACPermissionName(level storage.PermissionLevel) (string, error) {
	switch level {
	case storage.PermissionLevel_UNSET:
		return "UNSET", nil
	case storage.PermissionLevel_NONE:
		return "NONE", nil
	case storage.PermissionLevel_DEFAULT:
		return "DEFAULT", nil
	case storage.PermissionLevel_ELEVATED_IN_NAMESPACE:
		return "ELEVATED_IN_NAMESPACE", nil
	case storage.PermissionLevel_ELEVATED_CLUSTER_WIDE:
		return "ELEVATED_CLUSTER_WIDE", nil
	case storage.PermissionLevel_CLUSTER_ADMIN:
		return "CLUSTER_ADMIN", nil
	default:
		return "", errors.New("Invalid RBAC permission level")
	}
}

func convertExposureLevelPolicy(fields *storage.PolicyFields) []*storage.PolicyGroup {
	exposurePolicy := fields.GetPortExposurePolicy()
	if exposurePolicy == nil {
		return nil
	}

	levels := exposurePolicy.GetExposureLevels()
	var levelStrings []string
	for _, levelInt := range levels {
		levelString, err := getExposureLevelName(levelInt)
		if err != nil {
			return nil
		}
		levelStrings = append(levelStrings, levelString)
	}

	return []*storage.PolicyGroup{
		{
			FieldName: PortExposure,
			Values:    getStringListPolicyValues(levelStrings),
		},
	}
}

func getExposureLevelName(level storage.PortConfig_ExposureLevel) (string, error) {
	switch level {
	case storage.PortConfig_UNSET:
		return "UNSET", nil
	case storage.PortConfig_EXTERNAL:
		return "EXTERNAL", nil
	case storage.PortConfig_NODE:
		return "NODE", nil
	case storage.PortConfig_INTERNAL:
		return "INTERNAL", nil
	case storage.PortConfig_HOST:
		return "HOST", nil
	default:
		return "", errors.New("Invalid port exposure level")
	}
}
