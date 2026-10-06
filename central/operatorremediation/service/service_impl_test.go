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

func TestToResponse(t *testing.T) {
	const repo = "registry.redhat.io/openshift-service-mesh/istio-proxyv2-rhel9"
	crit := storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	imp := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	low := storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY

	report := operatorbundle.BundleDiffReport{
		Package:         "servicemeshoperator3",
		InstalledBundle: operatorbundle.Bundle{Version: "3.2.0"},
		UpdateCandidate: &operatorbundle.Bundle{Version: "3.2.9"},
		ImageDiffs: []operatorbundle.ImageDiff{{
			Repository:      repo,
			Name:            "images_v1_27_3_proxy",
			Status:          operatorbundle.ImagePaired,
			InstalledDigest: "sha256:old",
			CandidateDigest: "sha256:new",
			Fixed:           []operatorbundle.CVE{cve("CVE-1", crit), cve("CVE-2", low)},
			StillActive:     []operatorbundle.CVE{cve("CVE-3", imp)},
			New:             []operatorbundle.CVE{cve("CVE-4", imp)},
		}},
	}

	resp := toResponse(report)
	assert.Equal(t, "servicemeshoperator3", resp.GetOperatorPackage())
	assert.Equal(t, "3.2.0", resp.GetInstalledVersion())
	assert.Equal(t, "3.2.9", resp.GetUpdateVersion())
	require.Len(t, resp.GetImages(), 1)

	img := resp.GetImages()[0]
	assert.Equal(t, repo+"@sha256:old", img.GetInstalledImage())
	assert.Equal(t, repo+"@sha256:new", img.GetCandidateImage())
	assert.Equal(t, "PAIRED", img.GetStatus())

	// current = fixed + still-active: 1 critical, 1 important, 1 low.
	assert.Equal(t, int32(1), img.GetCurrent().GetCritical())
	assert.Equal(t, int32(1), img.GetCurrent().GetImportant())
	assert.Equal(t, int32(1), img.GetCurrent().GetLow())
	// fixed: 1 critical, 1 low.
	assert.Equal(t, int32(1), img.GetFixed().GetCritical())
	assert.Equal(t, int32(1), img.GetFixed().GetLow())
	// new: 1 important.
	assert.Equal(t, int32(1), img.GetNewCves().GetImportant())
}

func TestToResponseNoUpdate(t *testing.T) {
	resp := toResponse(operatorbundle.BundleDiffReport{
		Package:         "p",
		InstalledBundle: operatorbundle.Bundle{Version: "1.0.0"},
		NoUpdateReason:  "no newer patch release available within the installed major.minor",
	})
	assert.Empty(t, resp.GetUpdateVersion())
	assert.Equal(t, "no newer patch release available within the installed major.minor", resp.GetNoUpdateReason())
	assert.Empty(t, resp.GetImages())
}

func TestImageRefEmptyDigest(t *testing.T) {
	assert.Equal(t, "", imageRef("repo", ""))
	assert.Equal(t, "repo@sha256:x", imageRef("repo", "sha256:x"))
}
