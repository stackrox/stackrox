package storagewrappers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

const (
	testVector1 = "AV:N/AC:L/Au:N/C:P/I:P/A:P"
	testVector2 = "AV:L/AC:L/Au:N/C:C/I:C/A:C"
	testVector3 = "AV:A/AC:M/Au:N/C:N/I:N/A:P"
	testVector4 = "AV:N/AC:L/Au:S/C:P/I:N/A:N"
)

var (
	testCVSS1 = &storage.CVSSV2{
		Vector:              testVector1,
		AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
		AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
		Authentication:      storage.CVSSV2_AUTH_NONE,
		Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
		Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
		Availability:        storage.CVSSV2_IMPACT_PARTIAL,
		ExploitabilityScore: 10.0,
		ImpactScore:         6.4,
		Score:               7.5,
		Severity:            storage.CVSSV2_HIGH,
	}
	testCVSS2 = &storage.CVSSV2{
		Vector:              testVector2,
		AttackVector:        storage.CVSSV2_ATTACK_LOCAL,
		AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
		Authentication:      storage.CVSSV2_AUTH_NONE,
		Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
		Integrity:           storage.CVSSV2_IMPACT_COMPLETE,
		Availability:        storage.CVSSV2_IMPACT_COMPLETE,
		ExploitabilityScore: 3.9,
		ImpactScore:         10.0,
		Score:               7.2,
		Severity:            storage.CVSSV2_HIGH,
	}
	testCVSS3 = &storage.CVSSV2{
		Vector:              testVector3,
		AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
		AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
		Authentication:      storage.CVSSV2_AUTH_NONE,
		Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
		Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
		Availability:        storage.CVSSV2_IMPACT_PARTIAL,
		ExploitabilityScore: 5.5,
		ImpactScore:         2.9,
		Score:               2.9,
		Severity:            storage.CVSSV2_LOW,
	}
	testCVSS4 = &storage.CVSSV2{
		Vector:              testVector4,
		AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
		AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
		Authentication:      storage.CVSSV2_AUTH_SINGLE,
		Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
		Integrity:           storage.CVSSV2_IMPACT_NONE,
		Availability:        storage.CVSSV2_IMPACT_NONE,
		ExploitabilityScore: 8.0,
		ImpactScore:         2.9,
		Score:               4.0,
		Severity:            storage.CVSSV2_MEDIUM,
	}
)

type cvssV2TestCase struct {
	wrapper *CVSSV2Wrapper
	cvss    *storage.CVSSV2
}

func TestAsCVSSV2(t *testing.T) {
	for name, tc := range map[string]cvssV2TestCase{
		"Nil input": {
			wrapper: nil,
			cvss:    nil,
		},
		"Wrapper around nil": {
			wrapper: &CVSSV2Wrapper{},
			cvss:    nil,
		},
		"Wrapper around CVSS 1": {
			wrapper: &CVSSV2Wrapper{CVSSV2: testCVSS1.CloneVT()},
			cvss:    testCVSS1,
		},
		"Wrapper around CVSS 2": {
			wrapper: &CVSSV2Wrapper{CVSSV2: testCVSS2.CloneVT()},
			cvss:    testCVSS2,
		},
		"Wrapper around CVSS 3": {
			wrapper: &CVSSV2Wrapper{CVSSV2: testCVSS3.CloneVT()},
			cvss:    testCVSS3,
		},
		"Wrapper around CVSS 4": {
			wrapper: &CVSSV2Wrapper{CVSSV2: testCVSS4.CloneVT()},
			cvss:    testCVSS4,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.wrapper.AsCVSSV2()
			protoassert.Equal(it, tc.cvss, result)
		})
	}
}

func getSetterTestCases() map[string]cvssV2TestCase {
	return map[string]cvssV2TestCase{
		"Nil wrapper": {
			wrapper: nil,
			cvss:    testCVSS1,
		},
		"Wrapper around nil": {
			wrapper: &CVSSV2Wrapper{},
			cvss:    testCVSS1,
		},
		"Set from CVSS 1": {
			wrapper: &CVSSV2Wrapper{CVSSV2: &storage.CVSSV2{}},
			cvss:    testCVSS1,
		},
		"Set from CVSS 2": {
			wrapper: &CVSSV2Wrapper{CVSSV2: &storage.CVSSV2{}},
			cvss:    testCVSS2,
		},
		"Set from CVSS 3": {
			wrapper: &CVSSV2Wrapper{CVSSV2: &storage.CVSSV2{}},
			cvss:    testCVSS3,
		},
		"Set from CVSS 4": {
			wrapper: &CVSSV2Wrapper{CVSSV2: &storage.CVSSV2{}},
			cvss:    testCVSS4,
		},
	}
}

func TestSetVector(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetVector(tc.cvss.GetVector())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, "", tc.wrapper.AsCVSSV2().GetVector())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, "", tc.wrapper.AsCVSSV2().GetVector())
			} else {
				assert.Equal(it, tc.cvss.GetVector(), tc.wrapper.AsCVSSV2().GetVector())
			}
		})
	}
}

func TestSetAttackVector(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAttackVector(tc.cvss.GetAttackVector())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_ATTACK_LOCAL, tc.wrapper.AsCVSSV2().GetAttackVector())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_ATTACK_LOCAL, tc.wrapper.AsCVSSV2().GetAttackVector())
			} else {
				assert.Equal(it, tc.cvss.GetAttackVector(), tc.wrapper.AsCVSSV2().GetAttackVector())
			}
		})
	}
}

func TestSetAccessComplexity(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAccessComplexity(tc.cvss.GetAccessComplexity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_ACCESS_HIGH, tc.wrapper.AsCVSSV2().GetAccessComplexity())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_ACCESS_HIGH, tc.wrapper.AsCVSSV2().GetAccessComplexity())
			} else {
				assert.Equal(it, tc.cvss.GetAccessComplexity(), tc.wrapper.AsCVSSV2().GetAccessComplexity())
			}
		})
	}
}

func TestSetAuthentication(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAuthentication(tc.cvss.GetAuthentication())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_AUTH_MULTIPLE, tc.wrapper.AsCVSSV2().GetAuthentication())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_AUTH_MULTIPLE, tc.wrapper.AsCVSSV2().GetAuthentication())
			} else {
				assert.Equal(it, tc.cvss.GetAuthentication(), tc.wrapper.AsCVSSV2().GetAuthentication())
			}
		})
	}
}

func TestSetConfidentiality(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetConfidentiality(tc.cvss.GetConfidentiality())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetConfidentiality())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetConfidentiality())
			} else {
				assert.Equal(it, tc.cvss.GetConfidentiality(), tc.wrapper.AsCVSSV2().GetConfidentiality())
			}
		})
	}
}

func TestSetIntegrity(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetIntegrity(tc.cvss.GetIntegrity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetIntegrity())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetIntegrity())
			} else {
				assert.Equal(it, tc.cvss.GetIntegrity(), tc.wrapper.AsCVSSV2().GetIntegrity())
			}
		})
	}
}

func TestSetAvailability(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetAvailability(tc.cvss.GetAvailability())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetAvailability())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_IMPACT_NONE, tc.wrapper.AsCVSSV2().GetAvailability())
			} else {
				assert.Equal(it, tc.cvss.GetAvailability(), tc.wrapper.AsCVSSV2().GetAvailability())
			}
		})
	}
}

func TestSetExploitabilityScore(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetExploitabilityScore(tc.cvss.GetExploitabilityScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetExploitabilityScore())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetExploitabilityScore())
			} else {
				assert.Equal(it, tc.cvss.GetExploitabilityScore(), tc.wrapper.AsCVSSV2().GetExploitabilityScore())
			}
		})
	}
}

func TestSetImpactScore(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetImpactScore(tc.cvss.GetImpactScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetImpactScore())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetImpactScore())
			} else {
				assert.Equal(it, tc.cvss.GetImpactScore(), tc.wrapper.AsCVSSV2().GetImpactScore())
			}
		})
	}
}

func TestSetScore(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetScore(tc.cvss.GetScore())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetScore())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, float32(0.0), tc.wrapper.AsCVSSV2().GetScore())
			} else {
				assert.Equal(it, tc.cvss.GetScore(), tc.wrapper.AsCVSSV2().GetScore())
			}
		})
	}
}

func TestSetSeverity(t *testing.T) {
	for name, tc := range getSetterTestCases() {
		t.Run(name, func(it *testing.T) {
			tc.wrapper.SetSeverity(tc.cvss.GetSeverity())
			if tc.wrapper == nil {
				assert.Nil(it, tc.wrapper)
				assert.Equal(it, storage.CVSSV2_UNKNOWN, tc.wrapper.AsCVSSV2().GetSeverity())
			} else if tc.wrapper.CVSSV2 == nil {
				assert.Nil(it, tc.wrapper.AsCVSSV2())
				assert.Equal(it, storage.CVSSV2_UNKNOWN, tc.wrapper.AsCVSSV2().GetSeverity())
			} else {
				assert.Equal(it, tc.cvss.GetSeverity(), tc.wrapper.AsCVSSV2().GetSeverity())
			}
		})
	}
}
