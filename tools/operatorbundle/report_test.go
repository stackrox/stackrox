package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCSV(t *testing.T) {
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	const repo = "registry.redhat.io/openshift-service-mesh/istio-proxyv2-rhel9"

	reports := []operatorbundle.BundleDiffReport{{
		Package: "servicemeshoperator3",
		ImageDiffs: []operatorbundle.ImageDiff{
			{
				Repository:      repo,
				Name:            "images_v1_27_3_proxy",
				Status:          operatorbundle.ImagePaired,
				InstalledDigest: "sha256:old",
				CandidateDigest: "sha256:new",
				Fixed:           []operatorbundle.CVE{{ID: "CVE-FIX", Severity: sev}},
				StillActive:     []operatorbundle.CVE{{ID: "CVE-ACT", Severity: sev}},
				New:             []operatorbundle.CVE{{ID: "CVE-NEW", Severity: sev}},
			},
			{
				Repository:      repo,
				Name:            "images_v1_26_2_proxy",
				Status:          operatorbundle.ImageRemoved,
				InstalledDigest: "sha256:gone",
				Fixed:           []operatorbundle.CVE{{ID: "CVE-RM", Severity: sev}},
			},
		},
	}}

	var buf bytes.Buffer
	require.NoError(t, renderCSV(&buf, reports))

	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)

	// header + 3 rows (paired) + 1 row (removed)
	require.Len(t, records, 5)
	assert.Equal(t, []string{"installed_image", "candidate_image", "cve", "severity", "status"}, records[0])
	for _, row := range records {
		assert.Len(t, row, 5)
	}

	// Collect data rows keyed by CVE for order-independent assertions.
	byCVE := make(map[string][]string)
	for _, row := range records[1:] {
		byCVE[row[2]] = row
	}

	assert.Equal(t, []string{repo + "@sha256:old", repo + "@sha256:new", "CVE-FIX", "Important", "fixed"}, byCVE["CVE-FIX"])
	assert.Equal(t, []string{repo + "@sha256:old", repo + "@sha256:new", "CVE-ACT", "Important", "not-fixed"}, byCVE["CVE-ACT"])
	assert.Equal(t, []string{repo + "@sha256:old", repo + "@sha256:new", "CVE-NEW", "Important", "new"}, byCVE["CVE-NEW"])

	// REMOVED image: candidate_image column is empty.
	assert.Equal(t, []string{repo + "@sha256:gone", "", "CVE-RM", "Important", "fixed"}, byCVE["CVE-RM"])
}

func TestRenderCSVEmpty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderCSV(&buf, nil))
	assert.Equal(t, "installed_image,candidate_image,cve,severity,status\n", buf.String())
}
