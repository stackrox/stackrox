package cvssv2

import (
	"bufio"
	"fmt"
	"os"
	"testing"

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
								v2, err := ParseCVSSV2(v)
								assert.NoError(t, err)
								assert.Equal(t, attackVectorMap[av], v2.GetAttackVector())
								assert.Equal(t, accessComplexityMap[ac], v2.GetAccessComplexity())
								assert.Equal(t, authenticationMap[au], v2.GetAuthentication())
								assert.Equal(t, impactMap[c], v2.GetConfidentiality())
								assert.Equal(t, impactMap[i], v2.GetIntegrity())
								assert.Equal(t, impactMap[a], v2.GetAvailability())
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
			_, err := ParseCVSSV2(c)
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
			cvssV2, err := ParseCVSSV2(vec)
			assert.NoError(t, err)
			err = CalculateScores(cvssV2)
			assert.NoError(t, err)
			assert.InEpsilon(t, bS, cvssV2.GetScore(), 0.09)
			assert.InEpsilon(t, eS, cvssV2.GetExploitabilityScore(), 0.09)
			assert.InEpsilon(t, iS, cvssV2.GetImpactScore(), 0.09)
		})
	}
	require.NoError(t, s.Err())
}
