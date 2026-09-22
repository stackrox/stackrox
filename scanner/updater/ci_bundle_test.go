package updater

import (
	"archive/zip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// TestCIMinimalBundle validates the checked-in bundle through the same
// jsonblob reader used by the updater importer. Alpine vulnerability records
// intentionally carry Unknown severity; Scanner V4 applies the authoritative
// NVD enrichment record when it materializes the effective severity.
func TestCIMinimalBundle(t *testing.T) {
	bundlePath := filepath.Join("ci", "bundles", "ci-minimal", "vulnerabilities.zip")
	if candidate := os.Getenv("CI_MINIMAL_BUNDLE_PATH"); candidate != "" {
		bundlePath = candidate
	}
	r, err := zip.OpenReader(bundlePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Close())
	})

	seenSources := map[string]bool{}
	vulns := make(map[string][]vulnRecord)
	enrichments := make(map[string]enrichmentRecord)
	cvePattern := regexp.MustCompile(`CVE-[0-9]{4}-[0-9]+|RH[BS]A-[0-9]{4}:[0-9]+|ALAS[0-9]*-[0-9]{4}-[0-9]+|GO-[0-9]{4}-[0-9]+|GHSA-[a-z0-9-]+`)
	values, err := os.ReadFile(filepath.Join("..", "..", "deploy", "common", "ci-values.yaml"))
	require.NoError(t, err)
	var ci struct {
		Customize map[string]struct {
			EnvVars map[string]string `json:"envVars"`
		} `json:"customize"`
	}
	require.NoError(t, yaml.Unmarshal(values, &ci))
	allowlist := strings.Split(ci.Customize["scanner-v4-matcher"].EnvVars["SCANNER_V4_MATCHER_VULN_BUNDLE_ALLOWLIST"], ",")

	for _, f := range r.File {
		require.Contains(t, map[string]bool{
			"alpine.json.zst": true, "debian.json.zst": true, "nvd.json.zst": true,
			"ubuntu.json.zst": true, "osv.json.zst": true, "manual.json.zst": true,
			"rhel-vex.json.zst": true, "stackrox-rhel-csaf.json.zst": true,
			"aws.json.zst": true, "oracle.json.zst": true, "photon.json.zst": true,
		}, f.Name)
		require.NotContains(t, f.Name, "synthetic")
		seenSources[f.Name] = true
		require.Contains(t, allowlist, strings.TrimSuffix(f.Name, ".json.zst"), "CI must import every selected source")

		rc, err := f.Open()
		require.NoError(t, err)
		zr, err := zstd.NewReader(rc)
		require.NoError(t, err)
		opIt, itErr := jsonblob.Iterate(zr)
		opIt(func(_ *driver.UpdateOperation, recIt jsonblob.RecordIter) bool {
			recIt(func(v *claircore.Vulnerability, e *driver.EnrichmentRecord) bool {
				if v != nil {
					require.NotEmpty(t, v.Name)
					require.NotNil(t, v.Package)
					require.NotEmpty(t, v.Package.Name)
					var distribution string
					if v.Dist != nil {
						require.NotEmpty(t, v.Dist.DID)
						require.NotEmpty(t, v.Dist.VersionID)
						require.NotEqual(t, "synthetic", v.Dist.DID)
						distribution = v.Dist.VersionID
					} else {
						require.NotNil(t, v.Repo, "language and RHEL VEX records require repository identity")
						require.NotEmpty(t, v.Repo.Name)
					}
					require.NotContains(t, v.Package.Name, "synthetic")
					if v.Package.Name == "nginx" && f.Name == "alpine.json.zst" {
						require.NotEmpty(t, v.FixedInVersion)
					}
					identities := v.Name + " " + v.Links
					for _, alias := range v.Aliases {
						if alias.Valid() {
							identities += " " + alias.String()
						}
					}
					for _, cve := range cvePattern.FindAllString(identities, -1) {
						vulns[cve] = append(vulns[cve], vulnRecord{v.Package.Name, distribution, v.FixedInVersion, v.Links, f.Name, v.Invert})
					}
				}
				if e != nil {
					require.NotEmpty(t, e.Tags)
					if f.Name == "stackrox-rhel-csaf.json.zst" {
						var raw struct {
							Name string `json:"name"`
						}
						require.NoError(t, json.Unmarshal(e.Enrichment, &raw))
						require.NotEmpty(t, raw.Name)
						require.Contains(t, e.Tags, raw.Name)
						return true
					}
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

	for _, source := range []string{"ubuntu.json.zst", "osv.json.zst", "manual.json.zst", "rhel-vex.json.zst", "stackrox-rhel-csaf.json.zst"} {
		require.True(t, seenSources[source], "missing QA source %s", source)
	}
	for _, cve := range []string{"CVE-2017-5638", "CVE-2025-15467", "CVE-2021-33910", "CVE-2023-4911", "CVE-2025-11468", "CVE-2022-3219"} {
		require.NotEmpty(t, vulns[cve], "QA requires matching vulnerability records for %s, not just enrichment", cve)
		require.Contains(t, enrichments, cve, "missing QA enrichment")
	}
	var scannerCases []struct {
		DisabledReason string `json:"disabled_reason"`
		Features       []struct {
			Vulnerabilities []struct{ Name string }
		} `json:"expected_features"`
	}
	corpus, err := os.ReadFile(filepath.Join("..", "e2etests", "testdata", "image_tests.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(corpus, &scannerCases))
	for _, tc := range scannerCases {
		if tc.DisabledReason != "" {
			continue
		}
		for _, feature := range tc.Features {
			for _, vulnerability := range feature.Vulnerabilities {
				require.NotEmpty(t, vulns[vulnerability.Name], "missing native data required by Scanner E2E: %s", vulnerability.Name)
			}
		}
	}
	for name, tc := range map[string]struct{ cve, pkg, distribution, source string }{
		"struts":         {"CVE-2017-5638", "org.apache.struts:struts2-core", "", "osv.json.zst"},
		"ubuntu systemd": {"CVE-2021-33910", "systemd", "16.04", "ubuntu.json.zst"},
		"ubuntu libc":    {"CVE-2023-4911", "libc6", "22.04", "ubuntu.json.zst"},
		"ubuntu gpgv":    {"CVE-2022-3219", "gpgv", "22.04", "ubuntu.json.zst"},
		"rhel openssl":   {"CVE-2025-15467", "openssl-libs", "", "rhel-vex.json.zst"},
		"rhel python":    {"CVE-2025-11468", "python3", "", "rhel-vex.json.zst"},
		"amazon nss":     {"ALAS2-2024-2442", "nss-sysinit", "2", "aws.json.zst"},
		"oracle gcrypt":  {"CVE-2021-33560", "libgcrypt", "8", "oracle.json.zst"},
		"photon curl":    {"CVE-2023-38546", "curl", "3.0", "photon.json.zst"},
		"go advisory":    {"GO-2025-3487", "golang.org/x/crypto", "", "osv.json.zst"},
	} {
		t.Run(name, func(t *testing.T) {
			var found bool
			for _, record := range vulns[tc.cve] {
				found = found || (!record.inverted && record.packageName == tc.pkg &&
					record.distribution == tc.distribution && record.source == tc.source)
			}
			require.True(t, found, "missing matching identity for %s", tc.cve)
		})
	}
}

type vulnRecord struct {
	packageName, distribution, fixed string
	links, source                    string
	inverted                         bool
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
