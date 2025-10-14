package cvssv2

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCVSSV2(t *testing.T) {
	// Automatic test cases. Look through the values and add them and derive the expected response
	impactSlice := []string{"N", "P", "C"}
	for _, av := range []string{"L", "A", "N"} {
		for _, ac := range []string{"H", "M", "L"} {
			for _, au := range []string{"M", "S", "N"} {
				for _, c := range impactSlice {
					for _, i := range impactSlice {
						for _, a := range impactSlice {
							v := fmt.Sprintf("AV:%s/AC:%s/Au:%s/C:%s/I:%s/A:%s", av, ac, au, c, i, a)
							t.Run(v, func(t *testing.T) {
								wrapper := NewTestCVSSV2Wrapper()
								err := ParseCVSSV2(wrapper, v)
								assert.NoError(t, err)
								assert.Equal(t, v, wrapper.GetVector())
								assert.Equal(t, attackVectorMap[av], wrapper.GetCVSSV2().GetAttackVector())
								assert.Equal(t, accessComplexityMap[ac], wrapper.GetCVSSV2().GetAccessComplexity())
								assert.Equal(t, authenticationMap[au], wrapper.GetCVSSV2().GetAuthentication())
								assert.Equal(t, impactMap[c], wrapper.GetCVSSV2().GetConfidentiality())
								assert.Equal(t, impactMap[i], wrapper.GetCVSSV2().GetIntegrity())
								assert.Equal(t, impactMap[a], wrapper.GetCVSSV2().GetAvailability())
							})
						}
					}
				}
			}
		}
	}

	// Negative cases
	var cases = []string{
		"randomstring",
		"AV:N/AC:M/Au:S/C:N/I:P/A",
		"AV:N/AC:M/Au:S/C:N/I:P/A:Z",
		"AV:N/AC:M/Au:S/C:N/I:P/A:NOPE",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			wrapper := NewTestCVSSV2Wrapper()
			err := ParseCVSSV2(wrapper, c)
			assert.Error(t, err)
		})
	}
}

func Test_CalculateScores(t *testing.T) {
	f, err := os.Open("testdata/cvss.v2.samples")
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
			wrapper := NewTestCVSSV2Wrapper()
			err := ParseWCVSSV2(wrapper, vec)
			assert.NoError(t, err)
			err = CalculateScores(wrapper)
			assert.NoError(t, err)
			assert.InEpsilon(t, bS, wrapper.GetCVSSV2().GetScore(), 0.09)
			assert.InEpsilon(t, eS, wrapper.GetCVSSV2().GetExploitabilityScore(), 0.09)
			assert.InEpsilon(t, iS, wrapper.GetCVSSV2().GetImpactScore(), 0.09)
		})
	}
	require.NoError(t, s.Err())
}

// region helpers

// TestCVSSV2Wrapper is a test data structure that wraps around *storage.CVSSV2
// and implements the Writer interface for testing purposes.
type TestCVSSV2Wrapper struct {
	cvss *storage.CVSSV2
}

// NewTestCVSSV2Wrapper creates a new TestCVSSV2Wrapper with an empty CVSSV2 instance.
func NewTestCVSSV2Wrapper() *TestCVSSV2Wrapper {
	return &TestCVSSV2Wrapper{
		cvss: &storage.CVSSV2{},
	}
}

// GetCVSSV2 returns the underlying CVSSV2 instance.
func (w *TestCVSSV2Wrapper) GetCVSSV2() *storage.CVSSV2 {
	return w.cvss
}

// Writer interface implementation

func (w *TestCVSSV2Wrapper) GetVector() string {
	return w.cvss.GetVector()
}

func (w *TestCVSSV2Wrapper) SetVector(vector string) {
	w.cvss.Vector = vector
}

func (w *TestCVSSV2Wrapper) SetAttackVector(attackVector storage.CVSSV2_AttackVector) {
	w.cvss.AttackVector = attackVector
}

func (w *TestCVSSV2Wrapper) SetAccessComplexity(accessComplexity storage.CVSSV2_AccessComplexity) {
	w.cvss.AccessComplexity = accessComplexity
}

func (w *TestCVSSV2Wrapper) SetAuthentication(authentication storage.CVSSV2_Authentication) {
	w.cvss.Authentication = authentication
}

func (w *TestCVSSV2Wrapper) SetConfidentiality(impact storage.CVSSV2_Impact) {
	w.cvss.Confidentiality = impact
}

func (w *TestCVSSV2Wrapper) SetIntegrity(impact storage.CVSSV2_Impact) {
	w.cvss.Integrity = impact
}

func (w *TestCVSSV2Wrapper) SetAvailability(impact storage.CVSSV2_Impact) {
	w.cvss.Availability = impact
}

func (w *TestCVSSV2Wrapper) SetExploitabilityScore(score float32) {
	w.cvss.ExploitabilityScore = score
}

func (w *TestCVSSV2Wrapper) SetImpactScore(score float32) {
	w.cvss.ImpactScore = score
}

func (w *TestCVSSV2Wrapper) SetScore(score float32) {
	w.cvss.Score = score
}

func (w *TestCVSSV2Wrapper) SetSeverity(severity storage.CVSSV2_Severity) {
	w.cvss.Severity = severity
}

func TestTestCVSSV2Wrapper(t *testing.T) {
	// Test that the wrapper correctly implements the Writer interface
	wrapper := NewTestCVSSV2Wrapper()

	// Test setting values using the Writer interface methods
	wrapper.SetVector("AV:N/AC:L/Au:N/C:P/I:P/A:P")
	wrapper.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	wrapper.SetAccessComplexity(storage.CVSSV2_ACCESS_LOW)
	wrapper.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	wrapper.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	wrapper.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	wrapper.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	wrapper.SetExploitabilityScore(10.0)
	wrapper.SetImpactScore(6.4)
	wrapper.SetScore(5.0)
	wrapper.SetSeverity(storage.CVSSV2_MEDIUM)

	// Verify that the underlying CVSSV2 instance has the correct values
	cvss := wrapper.GetCVSSV2()
	assert.Equal(t, "AV:N/AC:L/Au:N/C:P/I:P/A:P", cvss.GetVector())
	assert.Equal(t, storage.CVSSV2_ATTACK_NETWORK, cvss.GetAttackVector())
	assert.Equal(t, storage.CVSSV2_ACCESS_LOW, cvss.GetAccessComplexity())
	assert.Equal(t, storage.CVSSV2_AUTH_NONE, cvss.GetAuthentication())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, cvss.GetConfidentiality())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, cvss.GetIntegrity())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, cvss.GetAvailability())
	assert.Equal(t, float32(10.0), cvss.GetExploitabilityScore())
	assert.Equal(t, float32(6.4), cvss.GetImpactScore())
	assert.Equal(t, float32(5.0), cvss.GetScore())
	assert.Equal(t, storage.CVSSV2_MEDIUM, cvss.GetSeverity())

	// Test that the wrapper can be used with functions expecting a Writer interface
	err := ParseWCVSSV2(wrapper, "AV:N/AC:L/Au:N/C:P/I:P/A:P")
	assert.NoError(t, err)

	// Verify the values were set correctly by the ParseWCVSSV2 function
	assert.Equal(t, "AV:N/AC:L/Au:N/C:P/I:P/A:P", wrapper.GetVector())
	assert.Equal(t, storage.CVSSV2_ATTACK_NETWORK, wrapper.GetCVSSV2().GetAttackVector())
	assert.Equal(t, storage.CVSSV2_ACCESS_LOW, wrapper.GetCVSSV2().GetAccessComplexity())
	assert.Equal(t, storage.CVSSV2_AUTH_NONE, wrapper.GetCVSSV2().GetAuthentication())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, wrapper.GetCVSSV2().GetConfidentiality())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, wrapper.GetCVSSV2().GetIntegrity())
	assert.Equal(t, storage.CVSSV2_IMPACT_PARTIAL, wrapper.GetCVSSV2().GetAvailability())
}

// endregion helpers
