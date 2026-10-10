package operatorbundle

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cve(id string, sev storage.VulnerabilitySeverity, fixedBy string) CVE {
	return CVE{ID: id, Severity: sev, FixedBy: fixedBy}
}

func cveIDs(cves []CVE) []string {
	ids := make([]string, 0, len(cves))
	for _, c := range cves {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestComputeDiff(t *testing.T) {
	const repo = "registry.redhat.io/albo/controller-rhel9"
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY

	t.Run("paired image classifies fixed, active and new", func(t *testing.T) {
		installed := []ImageCVEs{{Repository: repo, Digest: "sha256:old", CVEs: []CVE{
			cve("CVE-1", sev, "1.1"), // fixed
			cve("CVE-2", sev, ""),    // still active
		}}}
		candidate := []ImageCVEs{{Repository: repo, Digest: "sha256:new", CVEs: []CVE{
			cve("CVE-2", sev, ""),    // still active
			cve("CVE-3", sev, "2.0"), // new
		}}}

		diffs := ComputeDiff(installed, candidate)
		require.Len(t, diffs, 1)
		d := diffs[0]
		assert.Equal(t, ImagePaired, d.Status)
		assert.Equal(t, "sha256:old", d.InstalledDigest)
		assert.Equal(t, "sha256:new", d.CandidateDigest)
		assert.Equal(t, []string{"CVE-1"}, cveIDs(d.Fixed))
		assert.Equal(t, []string{"CVE-2"}, cveIDs(d.StillActive))
		assert.Equal(t, []string{"CVE-3"}, cveIDs(d.New))
	})

	t.Run("same repository, different names pair by name not repository", func(t *testing.T) {
		// A multi-version bundle: one repository holds several images distinguished only by
		// name (e.g. istio-cni-rhel9 for different Istio versions). Pairing must key on
		// (repository, name); keying on repository alone would collapse these into one.
		const cniRepo = "registry.redhat.io/openshift-service-mesh/istio-cni-rhel9"
		installed := []ImageCVEs{
			{Repository: cniRepo, Name: "images_v1_27_9_cni", Digest: "sha256:old-279", CVEs: []CVE{cve("CVE-A", sev, "")}},
			{Repository: cniRepo, Name: "images_v1_28_4_cni", Digest: "sha256:old-284", CVEs: []CVE{cve("CVE-B", sev, "")}},
		}
		candidate := []ImageCVEs{
			{Repository: cniRepo, Name: "images_v1_27_9_cni", Digest: "sha256:new-279", CVEs: []CVE{cve("CVE-A", sev, "")}},
			{Repository: cniRepo, Name: "images_v1_28_4_cni", Digest: "sha256:new-284", CVEs: []CVE{}},
		}

		diffs := ComputeDiff(installed, candidate)
		require.Len(t, diffs, 2, "each (repository, name) variant must yield its own diff")

		byName := make(map[string]ImageDiff, len(diffs))
		for _, d := range diffs {
			byName[d.Name] = d
		}

		v279 := byName["images_v1_27_9_cni"]
		assert.Equal(t, ImagePaired, v279.Status)
		assert.Equal(t, "sha256:old-279", v279.InstalledDigest)
		assert.Equal(t, "sha256:new-279", v279.CandidateDigest)
		assert.Equal(t, []string{"CVE-A"}, cveIDs(v279.StillActive))

		v284 := byName["images_v1_28_4_cni"]
		assert.Equal(t, ImagePaired, v284.Status)
		assert.Equal(t, "sha256:new-284", v284.CandidateDigest)
		assert.Equal(t, []string{"CVE-B"}, cveIDs(v284.Fixed), "CVE-B is fixed only in the 1.28.4 variant")
	})

	t.Run("empty names on same repository fall back to repository pairing", func(t *testing.T) {
		// Some operators (e.g. mariadb) leave name empty on all images; repository must still
		// distinguish them.
		installed := []ImageCVEs{
			{Repository: "reg/a", Name: "", Digest: "sha256:a-old", CVEs: []CVE{cve("CVE-1", sev, "")}},
			{Repository: "reg/b", Name: "", Digest: "sha256:b-old", CVEs: []CVE{cve("CVE-2", sev, "")}},
		}
		candidate := []ImageCVEs{
			{Repository: "reg/a", Name: "", Digest: "sha256:a-new", CVEs: []CVE{cve("CVE-1", sev, "")}},
			{Repository: "reg/b", Name: "", Digest: "sha256:b-new", CVEs: []CVE{}},
		}
		diffs := ComputeDiff(installed, candidate)
		require.Len(t, diffs, 2)
		assert.Equal(t, "reg/a", diffs[0].Repository)
		assert.Equal(t, ImagePaired, diffs[0].Status)
		assert.Equal(t, "reg/b", diffs[1].Repository)
		assert.Equal(t, []string{"CVE-2"}, cveIDs(diffs[1].Fixed))
	})

	t.Run("repository only in candidate is ADDED with all CVEs new", func(t *testing.T) {
		candidate := []ImageCVEs{{Repository: repo, Digest: "sha256:new", CVEs: []CVE{
			cve("CVE-9", sev, ""),
		}}}
		diffs := ComputeDiff(nil, candidate)
		require.Len(t, diffs, 1)
		assert.Equal(t, ImageAdded, diffs[0].Status)
		assert.Equal(t, []string{"CVE-9"}, cveIDs(diffs[0].New))
		assert.Empty(t, diffs[0].Fixed)
	})

	t.Run("repository only in installed is REMOVED with all CVEs fixed", func(t *testing.T) {
		installed := []ImageCVEs{{Repository: repo, Digest: "sha256:old", CVEs: []CVE{
			cve("CVE-9", sev, ""),
		}}}
		diffs := ComputeDiff(installed, nil)
		require.Len(t, diffs, 1)
		assert.Equal(t, ImageRemoved, diffs[0].Status)
		assert.Equal(t, []string{"CVE-9"}, cveIDs(diffs[0].Fixed))
		assert.Empty(t, diffs[0].New)
	})

	t.Run("diffs are sorted by repository", func(t *testing.T) {
		installed := []ImageCVEs{
			{Repository: "b/repo"},
			{Repository: "a/repo"},
		}
		diffs := ComputeDiff(installed, nil)
		require.Len(t, diffs, 2)
		assert.Equal(t, "a/repo", diffs[0].Repository)
		assert.Equal(t, "b/repo", diffs[1].Repository)
	})

	t.Run("CVEs sorted by descending severity then id", func(t *testing.T) {
		installed := []ImageCVEs{{Repository: repo, CVEs: []CVE{
			cve("CVE-2", storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, ""),
			cve("CVE-1", storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, ""),
			cve("CVE-3", storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, ""),
		}}}
		diffs := ComputeDiff(installed, nil)
		require.Len(t, diffs, 1)
		assert.Equal(t, []string{"CVE-1", "CVE-3", "CVE-2"}, cveIDs(diffs[0].Fixed))
	})
}
