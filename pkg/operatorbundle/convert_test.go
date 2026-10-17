package operatorbundle

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageCVEsFromStorage(t *testing.T) {
	t.Run("nil image returns nil", func(t *testing.T) {
		assert.Nil(t, ImageCVEsFromStorage(nil))
	})

	t.Run("extracts repository, digest and de-duplicated CVEs", func(t *testing.T) {
		img := &storage.Image{
			Id: "sha256:abc",
			Name: &storage.ImageName{
				Registry: "registry.redhat.io",
				Remote:   "albo/controller-rhel9",
				FullName: "registry.redhat.io/albo/controller-rhel9@sha256:abc",
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{Vulns: []*storage.EmbeddedVulnerability{
						{Cve: "CVE-1", Cvss: 7.5, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY},
						{Cve: "CVE-2"},
					}},
					{Vulns: []*storage.EmbeddedVulnerability{
						// CVE-1 again, this time with fix info; the fixable variant must win.
						{Cve: "CVE-1", SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"}},
						{Cve: ""}, // empty id skipped
					}},
				},
			},
		}

		result := ImageCVEsFromStorage(img)
		require.NotNil(t, result)
		assert.Equal(t, "registry.redhat.io/albo/controller-rhel9", result.Repository)
		assert.Equal(t, "sha256:abc", result.Digest)
		require.Len(t, result.CVEs, 2)

		byID := indexCVEs(result.CVEs)
		require.Contains(t, byID, "CVE-1")
		assert.Equal(t, "1.1", byID["CVE-1"].FixedBy, "fixable variant should win de-dup")
		assert.True(t, byID["CVE-1"].IsFixable())
		assert.False(t, byID["CVE-2"].IsFixable())
	})

	t.Run("no scan yields empty CVE list", func(t *testing.T) {
		img := &storage.Image{Id: "sha256:abc", Name: &storage.ImageName{FullName: "repo/x:1"}}
		result := ImageCVEsFromStorage(img)
		require.NotNil(t, result)
		assert.Empty(t, result.CVEs)
		assert.Equal(t, "repo/x", result.Repository)
	})
}
