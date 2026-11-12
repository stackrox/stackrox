package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCVEScores(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		cvss, nvd, epss string
		expected        string
	}{
		"none":          {},
		"cvss":          {cvss: "9", expected: "CVSS 9"},
		"nvd":           {nvd: "8.1", expected: "NVD CVSS 8.1"},
		"epss":          {epss: "75%", expected: "EPSS 75%"},
		"cvss and nvd":  {cvss: "6", nvd: "8", expected: "CVSS 6, NVD CVSS 8"},
		"cvss and epss": {cvss: "9", epss: "75%", expected: "CVSS 9, EPSS 75%"},
		"nvd and epss":  {nvd: "8.1", epss: "75%", expected: "NVD CVSS 8.1, EPSS 75%"},
		"all":           {cvss: "9", nvd: "8.1", epss: "75%", expected: "CVSS 9, NVD CVSS 8.1, EPSS 75%"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, cveScores(tc.cvss, tc.nvd, tc.epss))
		})
	}
}

func TestFormatEPSSPercent(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "75%", formatEPSSPercent("0.75"))
	assert.Equal(t, "80%", formatEPSSPercent("0.8"))
	assert.Equal(t, "0.042%", formatEPSSPercent("0.00042"))
	assert.Equal(t, "91.963%", formatEPSSPercent("0.91963"))
	assert.Equal(t, "not-a-number", formatEPSSPercent("not-a-number"))
}

func TestCvePrinterIncludesEPSS(t *testing.T) {
	t.Parallel()
	messages, err := cvePrinter(map[string][]string{
		search.CVE.String():            {"CVE-2024-0001"},
		search.CVSS.String():           {"9"},
		search.NVDCVSS.String():        {"8.1"},
		search.EPSSProbablity.String(): {"0.75"},
		search.Severity.String():       {"CRITICAL_VULNERABILITY_SEVERITY"},
	})
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "CVE-2024-0001 (CVSS 9, NVD CVSS 8.1, EPSS 75%) (severity Critical)")
}

func TestCvePrinterOmitsMissingEPSS(t *testing.T) {
	t.Parallel()
	messages, err := cvePrinter(map[string][]string{
		search.CVE.String():  {"CVE-2024-0002"},
		search.CVSS.String(): {"6"},
	})
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "CVE-2024-0002 (CVSS 6)")
	assert.NotContains(t, messages[0], "EPSS")
}
