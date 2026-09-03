package postgres

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateImageScanHash_DuplicateComponentsChangeHash(t *testing.T) {
	component := &storage.EmbeddedImageScanComponent{
		Name:    "ubi10/ubi-micro",
		Version: "1784668691",
	}

	scanWith1 := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{component},
	}
	scanWith3 := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{component, component, component},
	}

	require.NoError(t, populateImageScanHash(scanWith1))
	require.NoError(t, populateImageScanHash(scanWith3))

	assert.NotEqual(t, scanWith1.GetHash(), scanWith3.GetHash(),
		"removing duplicate components must produce a different hash")
}

func TestPopulateImageScanHash_DuplicateVulnsChangeHash(t *testing.T) {
	vuln := &storage.EmbeddedVulnerability{
		Cve:      "CVE-2025-1234",
		Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
	}

	scanWith1Vuln := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "pkg", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{vuln}},
		},
	}
	scanWith3Vulns := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "pkg", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{vuln, vuln, vuln}},
		},
	}

	require.NoError(t, populateImageScanHash(scanWith1Vuln))
	require.NoError(t, populateImageScanHash(scanWith3Vulns))

	assert.NotEqual(t, scanWith1Vuln.GetHash(), scanWith3Vulns.GetHash(),
		"removing duplicate vulns must produce a different hash")
}

func TestPopulateImageScanHash_DistinctChangesDetected(t *testing.T) {
	scanA := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "pkg", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{
				{Cve: "CVE-2025-1111"},
			}},
		},
	}
	scanB := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "pkg", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{
				{Cve: "CVE-2025-2222"},
			}},
		},
	}

	require.NoError(t, populateImageScanHash(scanA))
	require.NoError(t, populateImageScanHash(scanB))

	assert.NotEqual(t, scanA.GetHash(), scanB.GetHash(),
		"different CVEs must produce a different hash")
}

func TestPopulateImageScanHash_IdenticalScansMatch(t *testing.T) {
	scan1 := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "pkg-a", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{
				{Cve: "CVE-2025-1111"},
			}},
			{Name: "pkg-b", Version: "2.0"},
		},
	}
	scan2 := scan1.CloneVT()

	require.NoError(t, populateImageScanHash(scan1))
	require.NoError(t, populateImageScanHash(scan2))

	assert.Equal(t, scan1.GetHash(), scan2.GetHash(),
		"identical scans must produce the same hash")
}

func TestPopulateImageScanHash_OrderIndependent(t *testing.T) {
	compA := &storage.EmbeddedImageScanComponent{Name: "aaa", Version: "1.0"}
	compB := &storage.EmbeddedImageScanComponent{Name: "bbb", Version: "2.0"}

	scanAB := &storage.ImageScan{Components: []*storage.EmbeddedImageScanComponent{compA, compB}}
	scanBA := &storage.ImageScan{Components: []*storage.EmbeddedImageScanComponent{compB, compA}}

	require.NoError(t, populateImageScanHash(scanAB))
	require.NoError(t, populateImageScanHash(scanBA))

	assert.Equal(t, scanAB.GetHash(), scanBA.GetHash(),
		"component order must not affect the hash")
}
