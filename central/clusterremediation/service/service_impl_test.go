package service

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cve(id string, sev storage.VulnerabilitySeverity) operatorbundle.CVE {
	return operatorbundle.CVE{ID: id, Severity: sev}
}

func TestToReport(t *testing.T) {
	const repo = "quay.io/openshift-release-dev/ocp-v4.0-art-dev"
	crit := storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	imp := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY

	r := operatorbundle.BundleDiffReport{
		Package:         "openshift",
		InstalledBundle: operatorbundle.Bundle{Version: "4.22.15"},
		UpdateCandidate: &operatorbundle.Bundle{Version: "4.22.16"},
		ImageDiffs: []operatorbundle.ImageDiff{{
			Repository:      repo,
			Name:            "etcd",
			Status:          operatorbundle.ImagePaired,
			InstalledDigest: "sha256:old",
			CandidateDigest: "sha256:new",
			Fixed:           []operatorbundle.CVE{cve("CVE-1", crit)},
			StillActive:     []operatorbundle.CVE{cve("CVE-2", imp)},
			New:             []operatorbundle.CVE{cve("CVE-3", imp)},
		}},
	}

	report := toReport(r)
	assert.Equal(t, "4.22.15", report.GetCurrentVersion())
	assert.Equal(t, "4.22.16", report.GetTargetVersion())
	require.Len(t, report.GetImages(), 1)

	img := report.GetImages()[0]
	assert.Equal(t, "etcd", img.GetName())
	assert.Equal(t, repo+"@sha256:old", img.GetInstalledImage())
	assert.Equal(t, repo+"@sha256:new", img.GetCandidateImage())
	// current = fixed + still-active: 1 critical, 1 important.
	assert.Equal(t, int32(1), img.GetCurrent().GetCritical())
	assert.Equal(t, int32(1), img.GetCurrent().GetImportant())
	assert.Equal(t, int32(1), img.GetFixed().GetCritical())
	assert.Equal(t, int32(1), img.GetNewCves().GetImportant())
}
