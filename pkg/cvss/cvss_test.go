package cvss

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
								assert.Equal(t, attackVectorMap[av], v2.AttackVector)
								assert.Equal(t, accessComplexityMap[ac], v2.AccessComplexity)
								assert.Equal(t, authenticationMap[au], v2.Authentication)
								assert.Equal(t, impactMap[c], v2.Confidentiality)
								assert.Equal(t, impactMap[i], v2.Integrity)
								assert.Equal(t, impactMap[a], v2.Availability)
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
