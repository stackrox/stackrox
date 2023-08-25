package clairv4

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVulnName(t *testing.T) {
	testcases := []struct {
		original string
		expected string
	}{
		{
			// Alpine
			original: "CVE-2018-16840",
			expected: "CVE-2018-16840",
		},
		{
			// Amazon
			original: "ALAS-2022-1654",
			expected: "ALAS-2022-1654",
		},
		{
			// Debian
			original: "DSA-4591-1 cyrus-sasl2",
			expected: "DSA-4591-1",
		},
		{
			// pyup.io
			original: "pyup.io-38834 (CVE-2020-26137)",
			expected: "CVE-2020-26137",
		},
		{
			// RHEL
			original: "RHSA-2023:0173: libxml2 security update (Moderate)",
			expected: "RHSA-2023:0173",
		},
		{
			// Ubuntu
			original: "CVE-2022-45061 on Ubuntu 22.04 LTS (jammy) - medium.",
			expected: "CVE-2022-45061",
		},
		{
			// Something random
			original: "cool CVE right here",
			expected: "cool CVE right here",
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.original, func(t *testing.T) {
			assert.Equal(t, testcase.expected, vulnName(testcase.original))
		})
	}
}
