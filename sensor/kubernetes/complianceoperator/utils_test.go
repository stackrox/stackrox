package complianceoperator

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileKindToString(t *testing.T) {
	testCases := map[string]struct {
		kind     central.ComplianceOperatorProfileV2_OperatorKind
		expected string
	}{
		"profile": {
			kind:     central.ComplianceOperatorProfileV2_PROFILE,
			expected: complianceoperator.Profile.Kind,
		},
		"tailored profile": {
			kind:     central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: complianceoperator.TailoredProfile.Kind,
		},
		"unspecified": {
			kind:     central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: "",
		},
		"unknown": {
			kind:     central.ComplianceOperatorProfileV2_OperatorKind(999),
			expected: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, profileKindToString(tc.kind))
		})
	}
}

// TestBuildScanSettingBindingProfileRefsFromProfileRefs checks that when Central sends profile_refs
// (name + kind), we build NamedObjectReferences with the correct CO Kind (Profile vs TailoredProfile).
func TestBuildScanSettingBindingProfileRefsFromProfileRefs(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		Profiles: []string{"legacy"},
		ProfileRefs: []*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference{
			{Name: "ocp4-cis", Kind: central.ComplianceOperatorProfileV2_PROFILE},
			{Name: "ocp4-cis-tailored", Kind: central.ComplianceOperatorProfileV2_TAILORED_PROFILE},
		},
	}

	refs := buildScanSettingBindingProfileRefs("ns", req)
	require.Len(t, refs, 2)
	assert.Equal(t, "ocp4-cis", refs[0].Name)
	assert.Equal(t, complianceoperator.Profile.Kind, refs[0].Kind)
	assert.Equal(t, "ocp4-cis-tailored", refs[1].Name)
	assert.Equal(t, complianceoperator.TailoredProfile.Kind, refs[1].Kind)
}

// TestValidateScanSettingBindingProfileRefsFailsOnUnspecified ensures UNSPECIFIED kind is rejected.
func TestValidateScanSettingBindingProfileRefsFailsOnUnspecified(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		ProfileRefs: []*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference{
			{Name: "good", Kind: central.ComplianceOperatorProfileV2_PROFILE},
			{Name: "bad", Kind: central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED},
		},
	}

	err := validateScanSettingBindingProfiles(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
}

// TestValidateScanSettingBindingProfileRefsFailsOnUnknown ensures a truly unknown kind is rejected.
func TestValidateScanSettingBindingProfileRefsFailsOnUnknown(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		ProfileRefs: []*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference{
			{Name: "good", Kind: central.ComplianceOperatorProfileV2_PROFILE},
			{Name: "unknown", Kind: central.ComplianceOperatorProfileV2_OperatorKind(999)},
		},
	}

	err := validateScanSettingBindingProfiles(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

// TestBuildScanSettingBindingProfileRefsLegacyFallback verifies backwards compatibility: when
// profile_refs is empty (old Central), we use the profiles field and default every
// ref to Profile.Kind so older Sensors still work.
func TestBuildScanSettingBindingProfileRefsLegacyFallback(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		Profiles: []string{"p1", "p2"},
	}

	refs := buildScanSettingBindingProfileRefs("ns", req)
	require.Len(t, refs, 2)
	for i, name := range []string{"p1", "p2"} {
		assert.Equal(t, name, refs[i].Name)
		assert.Equal(t, complianceoperator.Profile.Kind, refs[i].Kind)
	}
}
