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
			wrapper := NewTestCVSSV3Wrapper()
			err := ParseCVSSV3(wrapper, c.input)
			assert.NoError(t, err)
			protoassert.Equal(t, c.cvssV3, wrapper.GetCVSSV3())
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
			wrapper := NewTestCVSSV3Wrapper()
			err := ParseCVSSV3(wrapper, c)
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
			wrapper := NewTestCVSSV3Wrapper()
			err := ParseCVSSV3(wrapper, c)
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
			wrapper := NewTestCVSSV3Wrapper()
			err := ParseCVSSV3(wrapper, vec)
			assert.NoError(t, err)
			err = CalculateScores(wrapper)
			assert.NoError(t, err)
			assert.InEpsilon(t, bS, wrapper.GetCVSSV3().GetScore(), 0.09)
			assert.InEpsilon(t, eS, wrapper.GetCVSSV3().GetExploitabilityScore(), 0.09)
			assert.InEpsilon(t, iS, wrapper.GetCVSSV3().GetImpactScore(), 0.09)
		})
	}
	require.NoError(t, s.Err())
}

// region helpers

// TestCVSSV3Wrapper is a test implementation of the Writer interface
type TestCVSSV3Wrapper struct {
	*storage.CVSSV3
}

func NewTestCVSSV3Wrapper() *TestCVSSV3Wrapper {
	return &TestCVSSV3Wrapper{
		CVSSV3: &storage.CVSSV3{},
	}
}

func (w *TestCVSSV3Wrapper) GetCVSSV3() *storage.CVSSV3 {
	if w == nil {
		return nil
	}
	return w.CVSSV3
}

func (w *TestCVSSV3Wrapper) GetVector() string {
	if w == nil || w.CVSSV3 == nil {
		return ""
	}
	return w.CVSSV3.GetVector()
}

func (w *TestCVSSV3Wrapper) SetVector(vector string) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Vector = vector
}

func (w *TestCVSSV3Wrapper) SetExploitabilityScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.ExploitabilityScore = score
}

func (w *TestCVSSV3Wrapper) SetImpactScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.ImpactScore = score
}

func (w *TestCVSSV3Wrapper) SetAttackVector(attackVector storage.CVSSV3_AttackVector) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.AttackVector = attackVector
}

func (w *TestCVSSV3Wrapper) SetAttackComplexity(attackComplexity storage.CVSSV3_Complexity) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.AttackComplexity = attackComplexity
}

func (w *TestCVSSV3Wrapper) SetPrivilegesRequired(privilegesRequired storage.CVSSV3_Privileges) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.PrivilegesRequired = privilegesRequired
}

func (w *TestCVSSV3Wrapper) SetUserInteraction(userInteraction storage.CVSSV3_UserInteraction) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.UserInteraction = userInteraction
}

func (w *TestCVSSV3Wrapper) SetScope(scope storage.CVSSV3_Scope) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Scope = scope
}

func (w *TestCVSSV3Wrapper) SetConfidentiality(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Confidentiality = impact
}

func (w *TestCVSSV3Wrapper) SetIntegrity(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Integrity = impact
}

func (w *TestCVSSV3Wrapper) SetAvailability(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Availability = impact
}

func (w *TestCVSSV3Wrapper) SetScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Score = score
}

func (w *TestCVSSV3Wrapper) SetSeverity(severity storage.CVSSV3_Severity) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.CVSSV3.Severity = severity
}

// endregion helpers
