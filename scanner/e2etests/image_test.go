package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vulnerability describes a vulnerability in a TestCase.
type Vulnerability struct {
	Name        string         `json:"Name"`
	Description string         `json:"Description"`
	Link        string         `json:"Link"`
	Severity    string         `json:"Severity"`
	Metadata    map[string]any `json:"Metadata"`
	FixedBy     string         `json:"FixedBy"`
}

// Vulnerability describes a feature in a TestCase.
type Feature struct {
	Name            string          `json:"Name"`
	VersionFormat   string          `json:"VersionFormat"`
	Version         string          `json:"Version"`
	Arch            string          `json:"Arch"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
	AddedBy         string          `json:"AddedBy"`
	Location        string          `json:"Location"`
	FixedBy         string          `json:"FixedBy"`
}

// TestWant describes the information wanted from the scan results in a TestCase.
type TestWant struct {
	Namespace string    `json:"namespace"`
	Source    string    `json:"source"`
	Features  []Feature `json:"expected_features"`
}

// TestArgs describes the arguments (i.e. test input) necessary to exercise the test case.
type TestArgs struct {
	Image    string `json:"image"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// TestCase represents on test case or scenario.
type TestCase struct {
	TestArgs
	TestWant
	// Test flags.
	DisabledReason          string `json:"disabled_reason"`
	OnlyCheckSpecifiedVulns bool   `json:"only_check_specified_vulns"`
}

// TestImage is the E2E test for container image scans.
func TestImage(t *testing.T) {
	indexerAddr := os.Getenv("SCANNER_E2E_INDEXER_ADDRESS")
	if indexerAddr == "" {
		indexerAddr = ":8443"
	}
	matcherAddr := os.Getenv("SCANNER_E2E_MATCHER_ADDRESS")
	if matcherAddr == "" {
		matcherAddr = ":8443"
	}
	debug := os.Getenv("SCANNER_E2E_DEBUG") == "1"
	t.Logf("Indexer Address: %s", indexerAddr)
	t.Logf("Matcher Address: %s", matcherAddr)
	t.Logf("Debug: %v", debug)
	ctx := context.Background()
	c, err := client.NewGRPCScanner(ctx,
		client.WithIndexerAddress(indexerAddr),
		client.WithMatcherAddress(matcherAddr),
		client.SkipTLSVerification)
	require.NoError(t, err)

	testCases, err := loadTestCases("testdata/image_tests.json")
	require.NoError(t, err, "parsing testdata/image_tests.json")

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s/%s", tc.Namespace, tc.Image), func(t *testing.T) {
			if tc.DisabledReason != "" {
				t.Skipf("%s disabled: reason: %q", t.Name(), tc.DisabledReason)
			}
			ref, err := name.ParseReference(tc.Image)
			require.NoError(t, err)

			auth := tc.authConfig()
			d, err := indexer.GetDigestFromReference(ref, auth)
			require.NoError(t, err)

			opt := client.ImageRegistryOpt{}
			vr, err := c.IndexAndScanImage(ctx, d, auth, opt)
			require.NoError(t, err)

			expected := tc.TestWant
			actual := tc.mapFillReport(t, vr)
			if debug {
				defer func() {
					if t.Failed() && vr != nil {
						tc.logReport(t, vr)
					}
				}()
			}
			assert.Equal(t, &expected, actual,
				"The vulnerability report did not contain the expected values.")
		})
	}
}

func loadTestCases(s string) ([]*TestCase, error) {
	f, err := os.Open(s)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(f.Close)
	decoder := json.NewDecoder(f)
	var cases []*TestCase
	err = decoder.Decode(&cases)
	if err != nil {
		return nil, err
	}
	return cases, nil
}

// mapNamespace creates a test case namespace from a vulnerability report.
func mapNamespace(vr *v4.VulnerabilityReport) string {
	if len(vr.GetContents().GetDistributions()) == 1 {
		d := vr.GetContents().GetDistributions()[0]
		return fmt.Sprintf("%s:%s", d.GetDid(), d.GetVersionId())
	}
	return "unknown"
}

func (tc *TestCase) authConfig() authn.Authenticator {
	return &authn.Basic{
		Username: os.Getenv(tc.Username),
		Password: os.Getenv(tc.Password),
	}
}

func (tc *TestCase) mapFillReport(t *testing.T, vr *v4.VulnerabilityReport) *TestWant {
	return &TestWant{
		Namespace: mapNamespace(vr),
		Source:    tc.Source,
		Features:  tc.mapFillFeatures(t, vr),
	}
}

// mapFillFeatures creates a features slice by converting values found in the
// vulnerability report.
func (tc *TestCase) mapFillFeatures(t *testing.T, vr *v4.VulnerabilityReport) []Feature {
	type VA struct {
		V string
		A string
	}
	// Populate map with all packages in the report.
	pkgs := make(map[string]map[VA]*v4.Package)
	for _, p := range vr.GetContents().GetPackages() {
		versions, ok := pkgs[p.Name]
		if !ok {
			versions = make(map[VA]*v4.Package)
			pkgs[p.Name] = versions
		}
		va := VA{V: p.Version, A: p.Arch}
		if _, ok := versions[va]; ok {
			t.Logf("Ignoring a duplicated package-version found in the vulnerability report: %v", va)
			continue
		}
		versions[va] = p
	}
	// Convert every expected package found in the report.
	var ret []Feature
	for idx := range tc.Features {
		f := &tc.Features[idx]
		versions, ok := pkgs[f.Name]
		if !ok {
			t.Logf("Expected package name not found in the vulnerability report: %s", f.Name)
			var c []string
			for n := range pkgs {
				if strings.Contains(n, f.Name) {
					c = append(c, n)
				}
			}
			if len(c) > 0 {
				t.Logf("Potential candidates: %v", c)
			} else {
				t.Log("No potential candidates found.")
			}
			continue
		}
		va := VA{V: f.Version, A: f.Arch}
		p, ok := versions[va]
		if !ok {
			t.Logf("Package %q with {version arch} %v not found in the vulnerability report", f.Name, va)
			t.Logf("Available versions: %+#v", versions)
			continue
		}
		if p != nil {
			env := environment(t, vr, p)
			ret = append(ret, Feature{
				Name:            p.GetName(),
				Version:         p.GetVersion(),
				Arch:            p.GetArch(),
				Vulnerabilities: tc.mapFillVulns(t, vr.GetVulnerabilities(), vr.GetPackageVulnerabilities()[p.GetId()].GetValues(), f),
				FixedBy:         p.GetFixedInVersion(),
				AddedBy:         env.GetIntroducedIn(),
				Location:        env.GetPackageDb(),
			})
		}
	}
	return ret
}

func environment(t *testing.T, vr *v4.VulnerabilityReport, pkg *v4.Package) *v4.Environment {
	envList, ok := vr.GetContents().GetEnvironments()[pkg.GetId()]
	if !ok {
		return nil
	}
	envs := envList.GetEnvironments()
	if len(envs) == 0 {
		return nil
	}
	var pkgDB, introIn string
	for i, e := range envs {
		if i == 0 {
			pkgDB, introIn = e.GetPackageDb(), e.GetIntroducedIn()
			continue
		}
		if pkgDB != e.GetPackageDb() || introIn != e.GetIntroducedIn() {
			t.Logf("Ignoring environment #%d for package %q: %v", i, pkg.GetName(), e)
		}
	}
	return envs[0]
}

func (tc *TestCase) mapFillVulns(t *testing.T, vr map[string]*v4.VulnerabilityReport_Vulnerability, vrIDs []string, feat *Feature) []Vulnerability {
	vrVulns := make(map[string]*v4.VulnerabilityReport_Vulnerability)
	for _, id := range vrIDs {
		v := vr[id]
		if _, ok := vrVulns[v.GetName()]; ok {
			continue
		}
		vrVulns[v.GetName()] = v
	}
	var vulns []Vulnerability
	for idx := range feat.Vulnerabilities {
		featVuln := &feat.Vulnerabilities[idx]
		// If not found, convert an empty vulnerability case.
		vrVuln, ok := vrVulns[featVuln.Name]
		if !ok {
			t.Logf("Missing vulnerability in the report: %q", featVuln.Name)
			continue
		}
		// We delete the vuln from the VR vulns map because we want to check after this
		// loop if anything was not matched (based on tc.OnlyCheckSpecifiedVulns).
		delete(vrVulns, featVuln.Name)
		vulns = append(vulns, tc.mapFillVuln(vrVuln, featVuln))
	}
	if tc.OnlyCheckSpecifiedVulns {
		return vulns
	}
	// See if there is any remaining vulnerabilities, and complain about them.
	if len(vrVulns) > 0 {
		t.Logf("Some vulnerabilities were not matched, and we want to match all of them")
		for _, v := range vrVulns {
			t.Logf("- Name: %q\n  ID: %q", v.GetName(), v.GetId())
			vulns = append(vulns, tc.mapFillVuln(v, &Vulnerability{}))
		}
	}
	return vulns
}

func (tc *TestCase) mapFillVuln(v *v4.VulnerabilityReport_Vulnerability, featVuln *Vulnerability) Vulnerability {
	// Link is by default the first on the list, but we check if the expected link is
	// somewhere else.
	link, _, _ := strings.Cut(v.GetLink(), " ")
	for _, l := range strings.Split(v.GetLink(), " ") {
		if l == featVuln.Link {
			link = l
		}
	}
	// Convert severity.
	severity, ok := map[v4.VulnerabilityReport_Vulnerability_Severity]string{
		v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL:    "Critical",
		v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT:   "Important",
		v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE:    "Moderate",
		v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW:         "Low",
		v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED: "Unspecified",
	}[v.GetNormalizedSeverity()]
	if !ok {
		severity = "Unknown"
	}
	var metadata map[string]any
	// TODO Use the CVSS score source type provided by the report when
	//      provided.
	var metadataKey string
	for k := range featVuln.Metadata {
		metadataKey = k
		break
	}
	f64 := func(f float32) float64 {
		return math.Round(float64(f)*100) / 100
	}
	if v.GetCvss() != nil {
		scores := map[string]any{}
		if v.GetCvss().GetV2() != nil {
			scores["CVSSv2"] = map[string]any{
				"Score":   f64(v.GetCvss().GetV2().GetBaseScore()),
				"Vectors": v.GetCvss().GetV2().GetVector(),
			}
		}
		if v.GetCvss().GetV3() != nil {
			scores["CVSSv3"] = map[string]any{
				"Score":   f64(v.GetCvss().GetV3().GetBaseScore()),
				"Vectors": v.GetCvss().GetV3().GetVector(),
			}
		}
		metadata = map[string]any{metadataKey: scores}
	}
	return Vulnerability{
		Name:        v.GetName(),
		Description: v.GetDescription(),
		Link:        link,
		Severity:    severity,
		FixedBy:     v.GetFixedInVersion(),

		Metadata: metadata,
	}
}

func (tc *TestCase) logReport(t *testing.T, vr *v4.VulnerabilityReport) {
	t.Log("Printing Packages and Vulnerabilities in Vulnerability Report:")
	for _, pkg := range vr.GetContents().GetPackages() {
		tc.logPackageAndVulns(t, vr, pkg)
	}
}

func (tc *TestCase) logPackageAndVulns(t *testing.T, vr *v4.VulnerabilityReport, pkg *v4.Package) {
	vulns := func(p *v4.Package) (v []*v4.VulnerabilityReport_Vulnerability) {
		for _, id := range vr.GetPackageVulnerabilities()[p.GetId()].GetValues() {
			v = append(v, vr.GetVulnerabilities()[id])
		}
		return v
	}
	s, err := json.MarshalIndent(struct {
		Package         *v4.Package
		Vulnerabilities []*v4.VulnerabilityReport_Vulnerability
	}{
		Package:         pkg,
		Vulnerabilities: vulns(pkg),
	}, "\t", " ")
	if err != nil {
		t.Logf("json marshalling failed: %v", err)
	}
	t.Logf("NameVersion: %s-%s:\n\n\t%s\n\n", pkg.GetName(), pkg.GetVersion(), s)
}
