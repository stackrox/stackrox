package clairclient

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifest_JSONRoundTrip(t *testing.T) {
	manifest := Manifest{
		Hash: "sha256:abc123",
		Layers: []Layer{
			{
				Hash: "sha256:layer1",
				URI:  "https://registry.io/v2/repo/blobs/sha256:layer1",
			},
			{
				Hash: "sha256:layer2",
				URI:  "https://registry.io/v2/repo/blobs/sha256:layer2",
				Headers: map[string][]string{
					"Authorization": {"Bearer token123"},
					"Accept":        {"application/vnd.docker.distribution.manifest.v2+json"},
				},
			},
		},
	}

	data, err := json.Marshal(manifest)
	require.NoError(t, err)

	var decoded Manifest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, manifest.Hash, decoded.Hash)
	assert.Len(t, decoded.Layers, 2)
	assert.Equal(t, manifest.Layers[0].Hash, decoded.Layers[0].Hash)
	assert.Equal(t, manifest.Layers[0].URI, decoded.Layers[0].URI)
	assert.Nil(t, decoded.Layers[0].Headers)
	assert.Equal(t, manifest.Layers[1].Headers, decoded.Layers[1].Headers)
}

func TestIndexReport_Unmarshal(t *testing.T) {
	jsonData := `{
		"manifest_hash": "sha256:abc123",
		"state": "IndexFinished",
		"success": true,
		"err": "",
		"packages": {
			"pkg1": {
				"id": "pkg1",
				"name": "openssl",
				"version": "1.1.1k",
				"kind": "binary",
				"arch": "amd64",
				"module": "openssl-libs",
				"cpe": "cpe:/a:openssl:openssl:1.1.1k",
				"package_db": "var/lib/dpkg/status",
				"repository_hint": "debian",
				"normalized_version": {
					"kind": "dpkg",
					"V": [0, 1, 1, 1, 107]
				}
			}
		},
		"distributions": {
			"dist1": {
				"id": "dist1",
				"did": "debian",
				"name": "Debian GNU/Linux",
				"version": "11",
				"version_code_name": "bullseye",
				"version_id": "11",
				"arch": "amd64",
				"cpe": "cpe:/o:debian:debian_linux:11",
				"pretty_name": "Debian GNU/Linux 11 (bullseye)"
			}
		},
		"repository": {
			"repo1": {
				"id": "repo1",
				"name": "debian-main",
				"key": "debian",
				"uri": "http://deb.debian.org/debian",
				"cpe": "cpe:/o:debian:debian_linux:11"
			}
		},
		"environments": {
			"env1": [
				{
					"package_db": "var/lib/dpkg/status",
					"introduced_in": "sha256:layer1",
					"distribution_id": "dist1",
					"repository_ids": ["repo1"]
				}
			]
		}
	}`

	var report IndexReport
	err := json.Unmarshal([]byte(jsonData), &report)
	require.NoError(t, err)

	assert.Equal(t, "sha256:abc123", report.ManifestHash)
	assert.Equal(t, "IndexFinished", report.State)
	assert.True(t, report.Success)
	assert.Empty(t, report.Err)

	require.Contains(t, report.Packages, "pkg1")
	pkg := report.Packages["pkg1"]
	assert.Equal(t, "openssl", pkg.Name)
	assert.Equal(t, "1.1.1k", pkg.Version)
	assert.Equal(t, "binary", pkg.Kind)
	assert.Equal(t, "amd64", pkg.Arch)
	assert.Equal(t, "dpkg", pkg.NormalizedVersion.Kind)
	assert.Equal(t, []int{0, 1, 1, 1, 107}, pkg.NormalizedVersion.V)

	require.Contains(t, report.Distributions, "dist1")
	dist := report.Distributions["dist1"]
	assert.Equal(t, "Debian GNU/Linux", dist.Name)
	assert.Equal(t, "bullseye", dist.VersionCodeName)

	require.Contains(t, report.Repositories, "repo1")
	repo := report.Repositories["repo1"]
	assert.Equal(t, "debian-main", repo.Name)
	assert.Equal(t, "http://deb.debian.org/debian", repo.URI)

	require.Contains(t, report.Environments, "env1")
	require.Len(t, report.Environments["env1"], 1)
	env := report.Environments["env1"][0]
	assert.Equal(t, "sha256:layer1", env.IntroducedIn)
	assert.Equal(t, "dist1", env.DistributionID)
	assert.Equal(t, []string{"repo1"}, env.RepositoryIDs)
}

func TestVulnerabilityReport_Unmarshal(t *testing.T) {
	issued := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	jsonData := `{
		"manifest_hash": "sha256:abc123",
		"state": "IndexFinished",
		"success": true,
		"err": "",
		"packages": {
			"pkg1": {
				"id": "pkg1",
				"name": "openssl",
				"version": "1.1.1k"
			}
		},
		"distributions": {
			"dist1": {
				"id": "dist1",
				"name": "Debian"
			}
		},
		"repository": {
			"repo1": {
				"id": "repo1",
				"name": "debian-main"
			}
		},
		"environments": {
			"env1": [
				{
					"package_db": "var/lib/dpkg/status",
					"introduced_in": "sha256:layer1"
				}
			]
		},
		"vulnerabilities": {
			"vuln1": {
				"id": "CVE-2021-3450",
				"name": "CVE-2021-3450",
				"description": "OpenSSL TLS vulnerability",
				"issued": "2021-05-01T00:00:00Z",
				"links": "https://nvd.nist.gov/vuln/detail/CVE-2021-3450",
				"severity": "High",
				"normalized_severity": "High",
				"fixed_in_version": "1.1.1l",
				"updater": "debian-updater"
			}
		},
		"package_vulnerabilities": {
			"pkg1": ["vuln1"]
		},
		"enrichments": {
			"vuln1": [
				{"source": "nvd", "score": 7.4}
			]
		}
	}`

	var report VulnerabilityReport
	err := json.Unmarshal([]byte(jsonData), &report)
	require.NoError(t, err)

	assert.Equal(t, "sha256:abc123", report.ManifestHash)
	assert.True(t, report.Success)

	require.Contains(t, report.Packages, "pkg1")
	assert.Equal(t, "openssl", report.Packages["pkg1"].Name)

	require.Contains(t, report.Distributions, "dist1")
	require.Contains(t, report.Repositories, "repo1")
	require.Contains(t, report.Environments, "env1")

	require.Contains(t, report.Vulnerabilities, "vuln1")
	vuln := report.Vulnerabilities["vuln1"]
	assert.Equal(t, "CVE-2021-3450", vuln.ID)
	assert.Equal(t, "OpenSSL TLS vulnerability", vuln.Description)
	assert.Equal(t, issued, vuln.Issued)
	assert.Equal(t, "High", vuln.Severity)
	assert.Equal(t, "1.1.1l", vuln.FixedInVersion)

	require.Contains(t, report.PackageVulnerabilities, "pkg1")
	assert.Equal(t, []string{"vuln1"}, report.PackageVulnerabilities["pkg1"])

	require.Contains(t, report.Enrichments, "vuln1")
	require.Len(t, report.Enrichments["vuln1"], 1)
}
