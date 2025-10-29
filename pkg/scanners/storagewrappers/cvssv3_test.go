package storagewrappers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

const (
	testVector31 = "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	testVector32 = "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N"
	testVector33 = "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:C/C:L/I:L/A:N"
	testVector34 = "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"
	testVector35 = "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:C/C:N/I:L/A:N"
)

var (
	testCVSS31 = &storage.CVSSV3{
		Vector:              testVector31,
		ExploitabilityScore: 3.9,
		ImpactScore:         5.9,
		AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
		UserInteraction:     storage.CVSSV3_UI_NONE,
		Scope:               storage.CVSSV3_UNCHANGED,
		Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
		Integrity:           storage.CVSSV3_IMPACT_HIGH,
		Availability:        storage.CVSSV3_IMPACT_HIGH,
		Score:               9.8,
		Severity:            storage.CVSSV3_CRITICAL,
	}
	testCVSS32 = &storage.CVSSV3{
		Vector:              testVector32,
		ExploitabilityScore: 3.9,
		ImpactScore:         1.4,
		AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
		UserInteraction:     storage.CVSSV3_UI_NONE,
		Scope:               storage.CVSSV3_UNCHANGED,
		Confidentiality:     storage.CVSSV3_IMPACT_LOW,
		Integrity:           storage.CVSSV3_IMPACT_NONE,
		Availability:        storage.CVSSV3_IMPACT_NONE,
		Score:               5.3,
		Severity:            storage.CVSSV3_MEDIUM,
	}
	testCVSS33 = &storage.CVSSV3{
		Vector:             testVector33,
		AttackVector:       storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:   storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired: storage.CVSSV3_PRIVILEGE_HIGH,
		UserInteraction:    storage.CVSSV3_UI_REQUIRED,
		Scope:              storage.CVSSV3_CHANGED,
		Confidentiality:    storage.CVSSV3_IMPACT_LOW,
		Integrity:          storage.CVSSV3_IMPACT_LOW,
		Availability:       storage.CVSSV3_IMPACT_NONE,
	}
	testCVSS34 = &storage.CVSSV3{
		Vector:             testVector34,
		AttackVector:       storage.CVSSV3_ATTACK_LOCAL,
		AttackComplexity:   storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired: storage.CVSSV3_PRIVILEGE_LOW,
		UserInteraction:    storage.CVSSV3_UI_NONE,
		Scope:              storage.CVSSV3_UNCHANGED,
		Confidentiality:    storage.CVSSV3_IMPACT_HIGH,
		Integrity:          storage.CVSSV3_IMPACT_HIGH,
		Availability:       storage.CVSSV3_IMPACT_HIGH,
	}
	testCVSS35 = &storage.CVSSV3{
		Vector:             testVector35,
		AttackVector:       storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:   storage.CVSSV3_COMPLEXITY_HIGH,
		PrivilegesRequired: storage.CVSSV3_PRIVILEGE_NONE,
		UserInteraction:    storage.CVSSV3_UI_NONE,
		Scope:              storage.CVSSV3_CHANGED,
		Confidentiality:    storage.CVSSV3_IMPACT_NONE,
		Integrity:          storage.CVSSV3_IMPACT_LOW,
		Availability:       storage.CVSSV3_IMPACT_NONE,
	}
)

type cvssV3TestCase struct {
	wrapper *CVSSV3Wrapper
	cvss    *storage.CVSSV3
}

func TestAsCVSSV3(t *testing.T) {
	for name, tc := range map[string]cvssV3TestCase{
		"Nil input": {
			wrapper: nil,
			cvss:    nil,
		},
		"Wrapper around nil": {
			wrapper: &CVSSV3Wrapper{},
			cvss:    nil,
		},
		"Wrapper around CVSS 1": {
			wrapper: &CVSSV3Wrapper{CVSSV3: testCVSS31.CloneVT()},
			cvss:    testCVSS31,
		},
		"Wrapper around CVSS 2": {
			wrapper: &CVSSV3Wrapper{CVSSV3: testCVSS32.CloneVT()},
			cvss:    testCVSS32,
		},
		"Wrapper around CVSS 3": {
			wrapper: &CVSSV3Wrapper{CVSSV3: testCVSS33.CloneVT()},
			cvss:    testCVSS33,
		},
		"Wrapper around CVSS 4": {
			wrapper: &CVSSV3Wrapper{CVSSV3: testCVSS34.CloneVT()},
			cvss:    testCVSS34,
		},
		"Wrapper around CVSS 5": {
			wrapper: &CVSSV3Wrapper{CVSSV3: testCVSS35.CloneVT()},
			cvss:    testCVSS35,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.wrapper.AsCVSSV3()
			protoassert.Equal(it, tc.cvss, result)
		})
	}
}

func getCVSSV3SetterTestCases() map[string]cvssV3TestCase {
	return map[string]cvssV3TestCase{
		"Nil wrapper": {
			wrapper: nil,
			cvss:    testCVSS31,
		},
		"Wrapper around nil": {
			wrapper: &CVSSV3Wrapper{},
			cvss:    testCVSS31,
		},
		"Set from CVSS 1": {
			wrapper: &CVSSV3Wrapper{CVSSV3: &storage.CVSSV3{}},
			cvss:    testCVSS31,
		},
		"Set from CVSS 2": {
			wrapper: &CVSSV3Wrapper{CVSSV3: &storage.CVSSV3{}},
			cvss:    testCVSS32,
		},
		"Set from CVSS 3": {
			wrapper: &CVSSV3Wrapper{CVSSV3: &storage.CVSSV3{}},
			cvss:    testCVSS33,
		},
		"Set from CVSS 4": {
			wrapper: &CVSSV3Wrapper{CVSSV3: &storage.CVSSV3{}},
			cvss:    testCVSS34,
		},
		"Set from CVSS 5": {
			wrapper: &CVSSV3Wrapper{CVSSV3: &storage.CVSSV3{}},
			cvss:    testCVSS35,
		},
	}
}

func TestCVSSV3WrapperSetVector(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetVector(tc.cvss.GetVector())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, "", tc.wrapper.AsCVSSV3().GetVector())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, "", tc.wrapper.AsCVSSV3().GetVector())
			} else {
				assert.Equal(it, tc.cvss.GetVector(), tc.wrapper.AsCVSSV3().GetVector())
			}
		})
	}
}

func TestCVSSV3WrapperSetExploitabilityScore(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetExploitabilityScore(tc.cvss.GetExploitabilityScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetExploitabilityScore())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetExploitabilityScore())
			} else {
				assert.Equal(it, tc.cvss.GetExploitabilityScore(), tc.wrapper.AsCVSSV3().GetExploitabilityScore())
			}
		})
	}
}

func TestCVSSV3WrapperSetImpactScore(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetImpactScore(tc.cvss.GetImpactScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetImpactScore())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetImpactScore())
			} else {
				assert.Equal(it, tc.cvss.GetImpactScore(), tc.wrapper.AsCVSSV3().GetImpactScore())
			}
		})
	}
}

func TestCVSSV3WrapperSetAttackVector(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAttackVector(tc.cvss.GetAttackVector())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_ATTACK_LOCAL, tc.wrapper.AsCVSSV3().GetAttackVector())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_ATTACK_LOCAL, tc.wrapper.AsCVSSV3().GetAttackVector())
			} else {
				assert.Equal(it, tc.cvss.GetAttackVector(), tc.wrapper.AsCVSSV3().GetAttackVector())
			}
		})
	}
}

func TestCVSSV3WrapperSetAttackComplexity(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAttackComplexity(tc.cvss.GetAttackComplexity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_COMPLEXITY_LOW, tc.wrapper.AsCVSSV3().GetAttackComplexity())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_COMPLEXITY_LOW, tc.wrapper.AsCVSSV3().GetAttackComplexity())
			} else {
				assert.Equal(it, tc.cvss.GetAttackComplexity(), tc.wrapper.AsCVSSV3().GetAttackComplexity())
			}
		})
	}
}

func TestCVSSV3WrapperSetPrivilegesRequired(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetPrivilegesRequired(tc.cvss.GetPrivilegesRequired())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_PRIVILEGE_NONE, tc.wrapper.AsCVSSV3().GetPrivilegesRequired())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_PRIVILEGE_NONE, tc.wrapper.AsCVSSV3().GetPrivilegesRequired())
			} else {
				assert.Equal(it, tc.cvss.GetPrivilegesRequired(), tc.wrapper.AsCVSSV3().GetPrivilegesRequired())
			}
		})
	}
}

func TestCVSSV3WrapperSetUserInteraction(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetUserInteraction(tc.cvss.GetUserInteraction())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_UI_NONE, tc.wrapper.AsCVSSV3().GetUserInteraction())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_UI_NONE, tc.wrapper.AsCVSSV3().GetUserInteraction())
			} else {
				assert.Equal(it, tc.cvss.GetUserInteraction(), tc.wrapper.AsCVSSV3().GetUserInteraction())
			}
		})
	}
}

func TestCVSSV3WrapperSetScope(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetScope(tc.cvss.GetScope())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_UNCHANGED, tc.wrapper.AsCVSSV3().GetScope())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_UNCHANGED, tc.wrapper.AsCVSSV3().GetScope())
			} else {
				assert.Equal(it, tc.cvss.GetScope(), tc.wrapper.AsCVSSV3().GetScope())
			}
		})
	}
}

func TestCVSSV3WrapperSetConfidentiality(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetConfidentiality(tc.cvss.GetConfidentiality())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetConfidentiality())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetConfidentiality())
			} else {
				assert.Equal(it, tc.cvss.GetConfidentiality(), tc.wrapper.AsCVSSV3().GetConfidentiality())
			}
		})
	}
}

func TestCVSSV3WrapperSetIntegrity(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetIntegrity(tc.cvss.GetIntegrity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetIntegrity())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetIntegrity())
			} else {
				assert.Equal(it, tc.cvss.GetIntegrity(), tc.wrapper.AsCVSSV3().GetIntegrity())
			}
		})
	}
}

func TestCVSSV3WrapperSetAvailability(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAvailability(tc.cvss.GetAvailability())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetAvailability())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_IMPACT_NONE, tc.wrapper.AsCVSSV3().GetAvailability())
			} else {
				assert.Equal(it, tc.cvss.GetAvailability(), tc.wrapper.AsCVSSV3().GetAvailability())
			}
		})
	}
}

func TestCVSSV3WrapperSetScore(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetScore(tc.cvss.GetScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetScore())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, float32(0), tc.wrapper.AsCVSSV3().GetScore())
			} else {
				assert.Equal(it, tc.cvss.GetScore(), tc.wrapper.AsCVSSV3().GetScore())
			}
		})
	}
}

func TestCVSSV3WrapperSetSeverity(t *testing.T) {
	for name, tc := range getCVSSV3SetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetSeverity(tc.cvss.GetSeverity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV3_UNKNOWN, tc.wrapper.AsCVSSV3().GetSeverity())
			} else if tc.wrapper.CVSSV3 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV3())
				assert.Equal(it, storage.CVSSV3_UNKNOWN, tc.wrapper.AsCVSSV3().GetSeverity())
			} else {
				assert.Equal(it, tc.cvss.GetSeverity(), tc.wrapper.AsCVSSV3().GetSeverity())
			}
		})
	}
}
