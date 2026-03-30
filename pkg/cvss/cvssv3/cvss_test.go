package cvssv3

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCVSSV3(t *testing.T) {
	cases := []struct {
		input  string
		cvssV3 *storage.CVSSV3
	}{
		{
			input: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N",
			cvssV3: &storage.CVSSV3{
				Vector:             "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N",
				AttackVector:       storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:   storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired: storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:    storage.CVSSV3_UI_NONE,
				Scope:              storage.CVSSV3_UNCHANGED,
				Confidentiality:    storage.CVSSV3_IMPACT_NONE,
				Integrity:          storage.CVSSV3_IMPACT_NONE,
				Availability:       storage.CVSSV3_IMPACT_NONE,
			},
		},
		{
			input: "CVSS:3.0/AV:N/AC:H/PR:H/UI:N/S:U/C:L/I:H/A:L",
			cvssV3: &storage.CVSSV3{
				Vector:             "CVSS:3.0/AV:N/AC:H/PR:H/UI:N/S:U/C:L/I:H/A:L",
				AttackVector:       storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:   storage.CVSSV3_COMPLEXITY_HIGH,
				PrivilegesRequired: storage.CVSSV3_PRIVILEGE_HIGH,
				UserInteraction:    storage.CVSSV3_UI_NONE,
				Scope:              storage.CVSSV3_UNCHANGED,
				Confidentiality:    storage.CVSSV3_IMPACT_LOW,
				Integrity:          storage.CVSSV3_IMPACT_HIGH,
				Availability:       storage.CVSSV3_IMPACT_LOW,
			},
		},
		{
			input: "CVSS:3.1/AV:P/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
			cvssV3: &storage.CVSSV3{
				Vector:             "CVSS:3.1/AV:P/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
				AttackVector:       storage.CVSSV3_ATTACK_PHYSICAL,
				AttackComplexity:   storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired: storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:    storage.CVSSV3_UI_NONE,
				Scope:              storage.CVSSV3_UNCHANGED,
				Confidentiality:    storage.CVSSV3_IMPACT_NONE,
				Integrity:          storage.CVSSV3_IMPACT_HIGH,
				Availability:       storage.CVSSV3_IMPACT_NONE,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			cvss, err := ParseCVSSV3(c.input)
			assert.NoError(t, err)
			protoassert.Equal(t, c.cvssV3, cvss)
		})
	}

	// General valid test cases
	validCases := []string{
		"CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:L/AC:H/PR:H/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:C/C:H/I:N/A:N",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:H",
		"CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:C/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:C/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:U/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:N/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:H/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:H/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H",
		"CVSS:3.1/AV:P/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:P/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:P/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:P/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:P/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
	}

	for _, c := range validCases {
		t.Run(c, func(t *testing.T) {
			_, err := ParseCVSSV3(c)
			assert.NoError(t, err)
		})
	}

	// Negative cases
	errorCases := []string{
		"randomstring",
		"AV:N/AC:M/Au:S/C:N/I:P/A",
		"AV:N/AC:M/Au:S/C:N/I:P/A:Z",
		"AV:N/AC:M/Au:S/C:N/I:P/A:NOPE",
	}
	for _, c := range errorCases {
		t.Run(c, func(t *testing.T) {
			_, err := ParseCVSSV3(c)
			assert.Error(t, err)
		})
	}
}

func Test_CalculateScores(t *testing.T) {
	f, err := os.Open("testdata/cvss.v3.1.samples")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	s := bufio.NewScanner(f)
	var bS, eS, iS float32
	var vec string
	for n := 1; s.Scan(); n++ {
		l := s.Text()
		_, err = fmt.Sscanf(l, "%f %f %f %s\n", &bS, &eS, &iS, &vec)
		require.NoError(t, err)
		t.Run(fmt.Sprintf("#%d/%s", n, l), func(t *testing.T) {
			cvssV3, err := ParseCVSSV3(vec)
			assert.NoError(t, err)
			err = CalculateScores(cvssV3)
			assert.NoError(t, err)
			assert.InEpsilon(t, bS, cvssV3.GetScore(), 0.09)
			assert.InEpsilon(t, eS, cvssV3.GetExploitabilityScore(), 0.09)
			assert.InEpsilon(t, iS, cvssV3.GetImpactScore(), 0.09)
		})
	}
	require.NoError(t, s.Err())
}

func FuzzParseCVSSV3(f *testing.F) {
	// Seed with valid CVSS v3.0 and v3.1 vectors covering various combinations
	validSeeds := []string{
		// CVSS 3.0 vectors
		"CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N",
		"CVSS:3.0/AV:N/AC:H/PR:H/UI:N/S:U/C:L/I:H/A:L",
		// CVSS 3.1 vectors with all attack vectors
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N",
		"CVSS:3.1/AV:L/AC:H/PR:H/UI:N/S:U/C:H/I:H/A:H",
		"CVSS:3.1/AV:P/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
		// Scope changed
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:H",
		// Different complexity and privilege levels
		"CVSS:3.1/AV:N/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:L",
		"CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:U/C:H/I:H/A:H",
		// Common real-world patterns
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N",
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N",
		// Invalid vectors that should return errors gracefully
		"",
		"randomstring",
		"CVSS:2.0/AV:N/AC:L/Au:N/C:P/I:P/A:P",
		"AV:N/AC:M/Au:S/C:N/I:P/A:Z",
	}

	for _, seed := range validSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, vectorStr string) {
		// The fuzzer should never panic, regardless of input
		// ParseCVSSV3 should either return a valid result or a non-nil error
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseCVSSV3 panicked with input %q: %v", vectorStr, r)
			}
		}()

		result, err := ParseCVSSV3(vectorStr)

		// If parsing succeeds, verify the result is valid
		if err == nil {
			require.NotNil(t, result, "ParseCVSSV3 returned nil result with nil error for input %q", vectorStr)

			// Verify the vector string is preserved
			assert.Equal(t, vectorStr, result.GetVector(), "Vector string should be preserved in result")

			// Verify all enum fields are set to valid values (non-negative)
			assert.GreaterOrEqual(t, int32(result.GetAttackVector()), int32(0), "AttackVector should be valid")
			assert.GreaterOrEqual(t, int32(result.GetAttackComplexity()), int32(0), "AttackComplexity should be valid")
			assert.GreaterOrEqual(t, int32(result.GetPrivilegesRequired()), int32(0), "PrivilegesRequired should be valid")
			assert.GreaterOrEqual(t, int32(result.GetUserInteraction()), int32(0), "UserInteraction should be valid")
			assert.GreaterOrEqual(t, int32(result.GetScope()), int32(0), "Scope should be valid")
			assert.GreaterOrEqual(t, int32(result.GetConfidentiality()), int32(0), "Confidentiality should be valid")
			assert.GreaterOrEqual(t, int32(result.GetIntegrity()), int32(0), "Integrity should be valid")
			assert.GreaterOrEqual(t, int32(result.GetAvailability()), int32(0), "Availability should be valid")

			// If parsing succeeded, CalculateScores should also succeed
			err = CalculateScores(result)
			assert.NoError(t, err, "CalculateScores should succeed for valid parsed CVSS vector %q", vectorStr)

			// Verify scores are in valid ranges
			if err == nil {
				assert.GreaterOrEqual(t, result.GetScore(), float32(0.0), "Score should be >= 0")
				assert.LessOrEqual(t, result.GetScore(), float32(10.0), "Score should be <= 10")
				assert.GreaterOrEqual(t, result.GetExploitabilityScore(), float32(0.0), "ExploitabilityScore should be >= 0")
				assert.GreaterOrEqual(t, result.GetImpactScore(), float32(0.0), "ImpactScore should be >= 0")
			}
		} else {
			// If parsing fails, ensure we got a proper error message
			assert.NotEmpty(t, err.Error(), "Error should have a non-empty message")
			assert.Nil(t, result, "ParseCVSSV3 should return nil result on error")
		}
	})
}
