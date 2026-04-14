package internaltov2storage

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageToCentralProfileKind(t *testing.T) {
	testCases := []struct {
		name     string
		input    storage.ComplianceOperatorProfileV2_OperatorKind
		expected central.ComplianceOperatorProfileV2_OperatorKind
	}{
		{
			name:     "profile",
			input:    storage.ComplianceOperatorProfileV2_PROFILE,
			expected: central.ComplianceOperatorProfileV2_PROFILE,
		},
		{
			name:     "tailored profile",
			input:    storage.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
		},
		{
			name:     "unspecified falls back to profile for backward compatibility",
			input:    storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: central.ComplianceOperatorProfileV2_PROFILE,
		},
		{
			name:     "unknown kind falls back to unspecified",
			input:    storage.ComplianceOperatorProfileV2_OperatorKind(999),
			expected: central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := StorageToCentralProfileKind(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func makeProfileMsg(name, namespace, description, title string, ruleNames []string, setValues []*central.ComplianceOperatorProfileV2_SetValue, kind central.ComplianceOperatorProfileV2_OperatorKind) *central.ComplianceOperatorProfileV2 {
	msg := &central.ComplianceOperatorProfileV2{
		Id:           "test-uid",
		ProfileId:    "xccdf_profile_" + name,
		Name:         name,
		Description:  description,
		Title:        title,
		Namespace:    namespace,
		OperatorKind: kind,
		SetValues:    setValues,
	}
	for _, r := range ruleNames {
		msg.Rules = append(msg.Rules, &central.ComplianceOperatorProfileV2_Rule{RuleName: r})
	}
	return msg
}

func TestEquivalenceHash_TailoredProfileGetsHash(t *testing.T) {
	msg := makeProfileMsg("my-tp", "openshift-compliance", "desc", "title",
		[]string{"rule-a", "rule-b"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE)
	result := ComplianceOperatorProfileV2(msg, "00000000-0000-0000-0000-000000000001")
	assert.Len(t, result.GetEquivalenceHash(), 64, "should be hex-encoded SHA-256")
}

func TestEquivalenceHash_RegularProfileNoHash(t *testing.T) {
	msg := makeProfileMsg("ocp4-cis", "openshift-compliance", "desc", "title",
		[]string{"rule-a"}, nil,
		central.ComplianceOperatorProfileV2_PROFILE)
	result := ComplianceOperatorProfileV2(msg, "00000000-0000-0000-0000-000000000001")
	assert.Empty(t, result.GetEquivalenceHash(), "regular profiles must not carry an equivalence hash")
}

func TestEquivalenceHash_UnspecifiedKindNoHash(t *testing.T) {
	msg := makeProfileMsg("ocp4-cis", "openshift-compliance", "desc", "title",
		[]string{"rule-a"}, nil,
		central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED)
	result := ComplianceOperatorProfileV2(msg, "00000000-0000-0000-0000-000000000001")
	assert.Empty(t, result.GetEquivalenceHash())
}

func TestEquivalenceHash_Stability(t *testing.T) {
	msg1 := makeProfileMsg("my-tp", "openshift-compliance", "desc", "title",
		[]string{"rule-a", "rule-b"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE)
	msg2 := makeProfileMsg("my-tp", "openshift-compliance", "desc", "title",
		[]string{"rule-a", "rule-b"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE)
	h1 := ComplianceOperatorProfileV2(msg1, "00000000-0000-0000-0000-000000000001").GetEquivalenceHash()
	h2 := ComplianceOperatorProfileV2(msg2, "00000000-0000-0000-0000-000000000001").GetEquivalenceHash()
	assert.Equal(t, h1, h2, "same inputs must produce same hash")
}

func TestEquivalenceHash_ClusterIndependent(t *testing.T) {
	msg1 := makeProfileMsg("my-tp", "openshift-compliance", "desc", "title",
		[]string{"rule-a"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE)
	msg2 := makeProfileMsg("my-tp", "openshift-compliance", "desc", "title",
		[]string{"rule-a"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE)
	h1 := ComplianceOperatorProfileV2(msg1, "00000000-0000-0000-0000-000000000001").GetEquivalenceHash()
	h2 := ComplianceOperatorProfileV2(msg2, "00000000-0000-0000-0000-000000000002").GetEquivalenceHash()
	assert.Equal(t, h1, h2, "hash must not depend on cluster ID")
}

func TestEquivalenceHash_FieldSensitivity(t *testing.T) {
	baseRules := []string{"rule-a", "rule-b"}

	baseHash := computeEquivalenceHash(makeProfileMsg("my-tp", "openshift-compliance", "desc", "title", baseRules, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	require.Len(t, baseHash, 64)

	tests := map[string]*central.ComplianceOperatorProfileV2{
		"name change":        makeProfileMsg("other-name", "openshift-compliance", "desc", "title", baseRules, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"namespace change":   makeProfileMsg("my-tp", "other-ns", "desc", "title", baseRules, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"description change": makeProfileMsg("my-tp", "openshift-compliance", "other-desc", "title", baseRules, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"title change":       makeProfileMsg("my-tp", "openshift-compliance", "desc", "other-title", baseRules, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"rule change":        makeProfileMsg("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-c"}, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"rule added":         makeProfileMsg("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-b", "rule-c"}, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"rule removed":       makeProfileMsg("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a"}, nil, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
		"setValue added": makeProfileMsg("my-tp", "openshift-compliance", "desc", "title", baseRules, []*central.ComplianceOperatorProfileV2_SetValue{
			{Name: "var-foo", Value: "bar"},
		}, central.ComplianceOperatorProfileV2_TAILORED_PROFILE),
	}

	for name, msg := range tests {
		t.Run(name, func(t *testing.T) {
			h := computeEquivalenceHash(msg)
			assert.NotEqual(t, baseHash, h, "changing %s must produce a different hash", name)
		})
	}
}

func TestEquivalenceHash_RuleOrderIndependent(t *testing.T) {
	h1 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a", "rule-b", "rule-c"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	h2 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-c", "rule-a", "rule-b"}, nil,
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	assert.Equal(t, h1, h2, "rule order must not affect hash")
}

func TestEquivalenceHash_SetValuesOrderIndependent(t *testing.T) {
	svA := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-a", Value: "va"}
	svB := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-b", Value: "vb"}

	h1 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a"}, []*central.ComplianceOperatorProfileV2_SetValue{svA, svB},
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	h2 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a"}, []*central.ComplianceOperatorProfileV2_SetValue{svB, svA},
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	assert.Equal(t, h1, h2, "set_values order must not affect hash")
}

func TestEquivalenceHash_SetValuesSensitivity(t *testing.T) {
	svA := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-a", Value: "va"}
	svB := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-b", Value: "vb"}
	baseHash := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a"}, []*central.ComplianceOperatorProfileV2_SetValue{svA, svB},
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))

	tests := map[string][]*central.ComplianceOperatorProfileV2_SetValue{
		"different value": {{Name: "var-a", Value: "other"}, svB},
		"different name":  {{Name: "var-x", Value: "va"}, svB},
		"extra entry":     {svA, svB, {Name: "var-c", Value: "vc"}},
		"fewer entries":   {svA},
	}

	for name, svs := range tests {
		t.Run(name, func(t *testing.T) {
			h := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
				[]string{"rule-a"}, svs,
				central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
			assert.NotEqual(t, baseHash, h)
		})
	}
}

func TestEquivalenceHash_RationaleExcluded(t *testing.T) {
	sv1 := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-a", Value: "va", Rationale: "reason-1"}
	sv2 := &central.ComplianceOperatorProfileV2_SetValue{Name: "var-a", Value: "va", Rationale: "reason-2"}

	h1 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a"}, []*central.ComplianceOperatorProfileV2_SetValue{sv1},
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	h2 := computeEquivalenceHash(makeProfileMsg("my-tp", "ns", "d", "t",
		[]string{"rule-a"}, []*central.ComplianceOperatorProfileV2_SetValue{sv2},
		central.ComplianceOperatorProfileV2_TAILORED_PROFILE))
	assert.Equal(t, h1, h2, "rationale must not affect hash — it is documentation, not configuration")
}
