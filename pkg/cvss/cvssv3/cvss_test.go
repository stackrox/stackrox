package cvssv3

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, c.cvssV3, cvss)
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
