package complianceoperator

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
)

func TestProfileKindToString(t *testing.T) {
	testCases := []struct {
		name     string
		kind     central.ComplianceOperatorProfileV2_OperatorKind
		expected string
	}{
		{
			name:     "profile",
			kind:     central.ComplianceOperatorProfileV2_PROFILE,
			expected: complianceoperator.Profile.Kind,
		},
		{
			name:     "tailored profile",
			kind:     central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: complianceoperator.TailoredProfile.Kind,
		},
		{
			name:     "unspecified",
			kind:     central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: "",
		},
		{
			name:     "unknown",
			kind:     central.ComplianceOperatorProfileV2_OperatorKind(999),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kind := profileKindToString(tc.kind)
			if kind != tc.expected {
				t.Fatalf("unexpected kind: got %q want %q", kind, tc.expected)
			}
		})
	}
}

// TestBuildScanSettingBindingProfileRefsFromProfileRefs checks that when Central sends profile_refs
// (name + kind), we build NamedObjectReferences with the correct CO Kind (Profile vs TailoredProfile).
func TestBuildScanSettingBindingProfileRefsFromProfileRefs(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		Profiles: []string{"legacy"},
		ProfileRefs: []*central.ApplyComplianceScanConfigRequest_ProfileReference{
			{Name: "ocp4-cis", Kind: central.ComplianceOperatorProfileV2_PROFILE},
			{Name: "ocp4-cis-tailored", Kind: central.ComplianceOperatorProfileV2_TAILORED_PROFILE},
		},
	}

	refs := buildScanSettingBindingProfileRefs("ns", req)
	if len(refs) != 2 {
		t.Fatalf("unexpected number of refs: got %d want 2", len(refs))
	}
	if refs[0].Name != "ocp4-cis" || refs[0].Kind != complianceoperator.Profile.Kind {
		t.Fatalf("unexpected first ref: got %+v", refs[0])
	}
	if refs[1].Name != "ocp4-cis-tailored" || refs[1].Kind != complianceoperator.TailoredProfile.Kind {
		t.Fatalf("unexpected second ref: got %+v", refs[1])
	}
}

// TestValidateScanSettingBindingProfileRefsFailsOnInvalid ensures we fail on the first invalid
// profile ref.
func TestValidateScanSettingBindingProfileRefsFailsOnInvalid(t *testing.T) {
	req := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName: "scan",
		Profiles: []string{"legacy"},
		ProfileRefs: []*central.ApplyComplianceScanConfigRequest_ProfileReference{
			{Name: "good", Kind: central.ComplianceOperatorProfileV2_PROFILE},
			{Name: "bad", Kind: central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED},
			{Name: "unknown", Kind: central.ComplianceOperatorProfileV2_OperatorKind(999)},
		},
	}

	err := validateScanSettingBindingProfileRefs(req)
	if err == nil {
		t.Fatalf("expected error for invalid profile refs")
	}
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
	if len(refs) != 2 {
		t.Fatalf("unexpected number of refs: got %d want 2", len(refs))
	}
	for i, name := range []string{"p1", "p2"} {
		if refs[i].Name != name {
			t.Fatalf("unexpected ref name at %d: got %q want %q", i, refs[i].Name, name)
		}
		if refs[i].Kind != complianceoperator.Profile.Kind {
			t.Fatalf("unexpected ref kind at %d: got %q want %q", i, refs[i].Kind, complianceoperator.Profile.Kind)
		}
	}
}
