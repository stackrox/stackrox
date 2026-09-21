package updater

import (
	"archive/zip"
	"encoding/json"
	"math"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
	"github.com/stretchr/testify/require"
)

// TestCIMinimalBundle validates the checked-in bundle through the same
// jsonblob reader used by the updater importer. Alpine vulnerability records
// intentionally carry Unknown severity; Scanner V4 applies the authoritative
// NVD enrichment record when it materializes the effective severity.
func TestCIMinimalBundle(t *testing.T) {
	bundlePath := filepath.Join("ci", "bundles", "ci-minimal", "vulnerabilities.zip")
	r, err := zip.OpenReader(bundlePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Close())
	})

	seenSources := map[string]bool{}
	vulns := make(map[string][]vulnRecord)
	enrichments := make(map[string]enrichmentRecord)

	for _, f := range r.File {
		require.Contains(t, map[string]bool{
			"alpine.json.zst": true, "debian.json.zst": true, "nvd.json.zst": true,
		}, f.Name)
		require.NotContains(t, f.Name, "synthetic")
		seenSources[f.Name] = true

		rc, err := f.Open()
		require.NoError(t, err)
		zr, err := zstd.NewReader(rc)
		require.NoError(t, err)
		opIt, itErr := jsonblob.Iterate(zr)
		opIt(func(_ *driver.UpdateOperation, recIt jsonblob.RecordIter) bool {
			recIt(func(v *claircore.Vulnerability, e *driver.EnrichmentRecord) bool {
				if v != nil {
					require.NotEmpty(t, v.Name)
					require.NotEmpty(t, v.Package.Name)
					require.NotEmpty(t, v.Dist.DID)
					require.NotEmpty(t, v.Dist.VersionID)
					require.NotContains(t, v.Package.Name, "synthetic")
					require.NotEqual(t, "synthetic", v.Dist.DID)
					if v.Package.Name == "nginx" {
						require.NotEmpty(t, v.FixedInVersion)
					}
					vulns[v.Name] = append(vulns[v.Name], vulnRecord{v.Package.Name, v.Dist.VersionID, v.FixedInVersion, v.Links})
				}
				if e != nil {
					var raw enrichmentJSON
					require.NoError(t, json.Unmarshal(e.Enrichment, &raw))
					require.NotEmpty(t, raw.ID)
					enrichments[raw.ID] = enrichmentRecord{firstDescription(raw.Descriptions), len(raw.References), highestScore(raw.Metrics)}
				}
				return true
			})
			return true
		})
		require.NoError(t, itErr())
		zr.Close()
		require.NoError(t, rc.Close())
	}

	for _, source := range []string{"alpine.json.zst", "debian.json.zst", "nvd.json.zst"} {
		require.True(t, seenSources[source], "missing source %s", source)
	}
	require.NotEmpty(t, enrichments)
	for cve, e := range enrichments {
		require.NotEmpty(t, e.description, "missing description for %s", cve)
		if !math.IsNaN(e.score) && e.score < 0 {
			t.Fatalf("invalid CVSS for %s", cve)
		}
	}

	var nginxImportant bool
	for cve, records := range vulns {
		for _, record := range records {
			if record.packageName == "nginx" && record.distribution == "3.9" {
				require.Contains(t, enrichments, cve, "missing nginx enrichment for %s", cve)
				require.NotEmpty(t, record.links, "missing nginx link for %s", cve)
				require.Greater(t, enrichments[cve].score, 0.0, "missing nginx CVSS for %s", cve)
			}
			if record.packageName == "nginx" && record.distribution == "3.9" && enrichments[cve].score >= 7.0 && record.fixed != "" {
				nginxImportant = true
			}
		}
	}
	require.True(t, nginxImportant, "nginx 3.9 must have a fixable Important-or-higher vulnerability")
}

type vulnRecord struct {
	packageName, distribution, fixed string
	links                            string
}
type enrichmentRecord struct {
	description string
	links       int
	score       float64
}
type enrichmentJSON struct {
	ID           string `json:"id"`
	Descriptions []struct {
		Value string `json:"value"`
	} `json:"descriptions"`
	References []struct {
		URL string `json:"url"`
	} `json:"references"`
	Metrics map[string][]struct {
		CVSSData struct {
			BaseScore float64 `json:"baseScore"`
		} `json:"cvssData"`
	} `json:"metrics"`
}

func firstDescription(descriptions []struct {
	Value string `json:"value"`
}) string {
	for _, description := range descriptions {
		if description.Value != "" {
			return description.Value
		}
	}
	return ""
}

func highestScore(metrics map[string][]struct {
	CVSSData struct {
		BaseScore float64 `json:"baseScore"`
	} `json:"cvssData"`
}) float64 {
	var highest float64
	for _, entries := range metrics {
		for _, entry := range entries {
			if entry.CVSSData.BaseScore > highest {
				highest = entry.CVSSData.BaseScore
			}
		}
	}
	return highest
}
