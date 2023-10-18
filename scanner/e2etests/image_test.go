package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/client"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vulnerability describes a vulnerability in a TestCase.
type Vulnerability struct {
	Name        string                 `json:"Name"`
	Description string                 `json:"Description"`
	Link        string                 `json:"Link"`
	Severity    string                 `json:"Severity"`
	Metadata    map[string]interface{} `json:"Metadata"`
	FixedBy     string                 `json:"FixedBy"`
}

// Vulnerability describes a feature in a TestCase.
type Feature struct {
	Name            string          `json:"Name"`
	NamespaceName   string          `json:"NamespaceName"`
	VersionFormat   string          `json:"VersionFormat"`
	Version         string          `json:"Version"`
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
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// TestCase represents on test case or scenario.
type TestCase struct {
	TestArgs
	TestWant
	// Test flags.
	OnlyCheckSpecifiedVulns  bool `json:"only_check_specified_vulns"`
	UncertifiedRhel          bool `json:"uncertified_rhel"`
	CheckProvidedExecutables bool `json:"check_provided_executables"`
}

// TestImage is the E2E test for container image scans.
func TestImage(t *testing.T) {
	ctx := context.Background()
	c, err := client.NewGRPCScannerClient(ctx,
		client.WithAddress(":8443"),
		client.WithoutTLSVerify)
	require.NoError(t, err)

	testCases, err := loadTestCases("testdata/image_tests.json")
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s/%s", tc.Namespace, tc.Image), func(t *testing.T) {
			ref, err := name.ParseReference(tc.Image)
			require.NoError(t, err)

			auth := tc.authConfig()
			d, err := indexer.GetDigestFromReference(ref, auth)
			require.NoError(t, err)

			vr, err := c.IndexAndScanImage(ctx, d, auth)
			require.NoError(t, err)

			expected := tc.TestWant
			converted := tc.convertReport(vr)
			defer func() {
				if t.Failed() && vr != nil {
					tc.logReport(t, vr)
				}
			}()
			assert.Equal(t, &expected, converted,
				"The converted vulnerability report did not contain the expected values.")
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

// namespace creates a test case namespace from a vulnerability report.
func namespace(vr *v4.VulnerabilityReport) string {
	if len(vr.GetContents().GetDistributions()) == 1 {
		d := vr.GetContents().GetDistributions()[0]
		return fmt.Sprintf("%s:%s", d.GetDid(), d.GetVersionId())
	}
	return "unknown"
}

func (tc *TestCase) authConfig() authn.Authenticator {
	return authn.FromConfig(authn.AuthConfig{
		Username: os.Getenv(tc.Username),
		Password: os.Getenv(tc.Password),
	})
}

func (tc *TestCase) convertReport(vr *v4.VulnerabilityReport) *TestWant {
	return &TestWant{
		Namespace: namespace(vr),
		Source:    tc.Source,
		Features:  tc.convertFeatures(vr),
	}
}

func (tc *TestCase) convertFeatures(vr *v4.VulnerabilityReport) []Feature {
	nv := func(n, v string) string {
		return fmt.Sprintf("%s-%s", n, v)
	}
	// Populate map with all features in the test case.
	feats := make(map[string]*Feature, len(tc.Features))
	for idx, f := range tc.Features {
		feats[nv(f.Name, f.Version)] = &tc.Features[idx]
	}
	// Convert every expected package in the report.
	ret := make([]Feature, 0, len(tc.Features))
	for _, p := range vr.GetContents().GetPackages() {
		f, ok := feats[nv(p.Name, p.Version)]
		if !ok {
			// We only care for features in the test case.
			continue
		}
		ret = append(ret, Feature{
			Name:            p.GetName(),
			NamespaceName:   namespace(vr),
			Version:         p.GetVersion(),
			Vulnerabilities: tc.convertVulns(vr, p, f),
			// TODO Pending fields not currently available in the vulnerability report.
			VersionFormat: f.VersionFormat,
			AddedBy:       f.AddedBy,
			Location:      f.Location,
			FixedBy:       f.FixedBy,
		})
	}
	return ret
}

func (tc *TestCase) convertVulns(vr *v4.VulnerabilityReport, pkg *v4.Package, feat *Feature) []Vulnerability {
	// Populate map with all vulnerabilities in the test case feature.
	featVulns := make(map[string]*Vulnerability)
	for idx, featVuln := range feat.Vulnerabilities {
		featVulns[featVuln.Name] = &feat.Vulnerabilities[idx]
	}
	// Convert all package vulnerabilities.
	var vulns []Vulnerability
	for _, id := range vr.GetPackageVulnerabilities()[pkg.GetId()].GetValues() {
		v := vr.GetVulnerabilities()[id]
		n, _, _ := strings.Cut(v.GetName(), " ")
		featVuln, ok := featVulns[n]
		if !ok && tc.OnlyCheckSpecifiedVulns {
			// Skip since we only check what is in the test features.
			continue
		}
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
		vulns = append(vulns, Vulnerability{
			Name:        n,
			Description: v.GetDescription(),
			Link:        link,
			Severity:    severity,
			FixedBy:     v.GetFixedInVersion(),
			// TODO Pending fields not currently available in the vulnerability report.
			Metadata: featVuln.Metadata,
		})
	}
	return vulns
}

func (tc *TestCase) logReport(t *testing.T, vr *v4.VulnerabilityReport) {
	vulns := func(p *v4.Package) (v []*v4.VulnerabilityReport_Vulnerability) {
		for _, id := range vr.GetPackageVulnerabilities()[p.GetId()].GetValues() {
			v = append(v, vr.GetVulnerabilities()[id])
		}
		return v
	}
	t.Log("Printing Packages and Vulnerabilities in Vulnerability Report:")
	for _, pkg := range vr.GetContents().GetPackages() {
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
}
