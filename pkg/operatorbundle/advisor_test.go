package operatorbundle

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fmtSprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

func joined(msgs []string) string { return strings.Join(msgs, "\n") }

// --- in-memory fakes for the three interfaces ---

type fakeCatalog struct {
	installed  map[string]*Bundle // digest -> bundle
	candidates map[string][]Bundle
}

func (f *fakeCatalog) FindInstalledBundle(_ context.Context, digest string) (*Bundle, error) {
	return f.installed[digest], nil
}

func (f *fakeCatalog) FindCandidateBundles(_ context.Context, pkg, _, _ string) ([]Bundle, error) {
	return f.candidates[pkg], nil
}

type fakeScanner struct {
	byRef map[string]*ImageCVEs
	scans []string
}

func (f *fakeScanner) ScanImage(_ context.Context, reference string) (*ImageCVEs, error) {
	f.scans = append(f.scans, reference)
	return f.byRef[reference], nil
}

type fakeInstalled struct {
	byDigest map[string]*ImageCVEs
}

func (f *fakeInstalled) GetCVEsByDigest(_ context.Context, digest string) (*ImageCVEs, error) {
	return f.byDigest[digest], nil
}

type fakeDeployed struct {
	deployed map[string]struct{}
	calls    int
}

func (f *fakeDeployed) ListDeployedAmong(_ context.Context, digests []string) (map[string]struct{}, error) {
	f.calls++
	out := make(map[string]struct{})
	for _, d := range digests {
		if _, ok := f.deployed[d]; ok {
			out[d] = struct{}{}
		}
	}
	return out, nil
}

func TestAdvise(t *testing.T) {
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	const repo = "registry.redhat.io/albo/controller-rhel9"

	installedBundle := &Bundle{
		Package:      "aws-load-balancer-operator",
		ChannelName:  "stable-v1",
		Version:      "1.4.0",
		CSVName:      "aws-load-balancer-operator.v1.4.0",
		CreationDate: "2026-06-01",
		RelatedImages: []RelatedImage{
			{Image: repo, Digest: "sha256:old"},
		},
	}
	candidateBundle := Bundle{
		Package:     "aws-load-balancer-operator",
		ChannelName: "stable-v1",
		Version:     "1.4.3",
		CSVName:     "aws-load-balancer-operator.v1.4.3",
		RelatedImages: []RelatedImage{
			{Image: repo, Digest: "sha256:new"},
		},
	}

	catalog := &fakeCatalog{
		installed: map[string]*Bundle{
			"sha256:img-a": installedBundle,
			"sha256:img-b": installedBundle, // second image, same bundle
		},
		candidates: map[string][]Bundle{
			"aws-load-balancer-operator": {
				candidateBundle,
				{Package: "aws-load-balancer-operator", Version: "1.5.0"}, // ignored: higher minor
			},
		},
	}
	installedSrc := &fakeInstalled{byDigest: map[string]*ImageCVEs{
		"sha256:old": {Repository: repo, Digest: "sha256:old", CVEs: []CVE{
			cve("CVE-1", sev, "1.1"), cve("CVE-2", sev, ""),
		}},
	}}
	scanner := &fakeScanner{byRef: map[string]*ImageCVEs{
		repo + "@sha256:new": {Repository: repo, Digest: "sha256:new", CVEs: []CVE{
			cve("CVE-2", sev, ""), cve("CVE-3", sev, "2.0"),
		}},
	}}

	advisor := NewAdvisor(catalog, scanner, installedSrc)

	t.Run("groups digests sharing a bundle and diffs correctly", func(t *testing.T) {
		reports, unresolved, err := advisor.Advise(context.Background(), []string{"sha256:img-a", "sha256:img-b"})
		require.NoError(t, err)
		assert.Empty(t, unresolved)
		require.Len(t, reports, 1, "two images of the same bundle must produce one report")

		r := reports[0]
		require.True(t, r.HasUpdate())
		assert.Equal(t, "1.4.3", r.UpdateCandidate.Version)
		assert.ElementsMatch(t, []string{"sha256:img-a", "sha256:img-b"}, r.TriggeringDigests)

		require.Len(t, r.ImageDiffs, 1)
		d := r.ImageDiffs[0]
		assert.Equal(t, ImagePaired, d.Status)
		assert.Equal(t, []string{"CVE-1"}, cveIDs(d.Fixed))
		assert.Equal(t, []string{"CVE-2"}, cveIDs(d.StillActive))
		assert.Equal(t, []string{"CVE-3"}, cveIDs(d.New))

		// The candidate image is scanned by pinned digest reference.
		assert.Contains(t, scanner.scans, repo+"@sha256:new")
	})

	t.Run("unresolved digests are reported", func(t *testing.T) {
		reports, unresolved, err := advisor.Advise(context.Background(), []string{"sha256:unknown"})
		require.NoError(t, err)
		assert.Empty(t, reports)
		assert.Equal(t, []string{"sha256:unknown"}, unresolved)
	})

	t.Run("progress callback receives step-by-step messages", func(t *testing.T) {
		var messages []string
		a := NewAdvisor(catalog, scanner, installedSrc, WithProgress(func(format string, args ...any) {
			messages = append(messages, fmtSprintf(format, args...))
		}))
		_, _, err := a.Advise(context.Background(), []string{"sha256:img-a"})
		require.NoError(t, err)
		require.NotEmpty(t, messages)
		assert.Contains(t, joined(messages), "Resolving installed bundle")
		assert.Contains(t, joined(messages), "scanning")
	})

	t.Run("no update candidate yields report with reason", func(t *testing.T) {
		noCandidateCatalog := &fakeCatalog{
			installed:  map[string]*Bundle{"sha256:img-a": installedBundle},
			candidates: map[string][]Bundle{}, // none
		}
		a := NewAdvisor(noCandidateCatalog, scanner, installedSrc)
		reports, _, err := a.Advise(context.Background(), []string{"sha256:img-a"})
		require.NoError(t, err)
		require.Len(t, reports, 1)
		assert.False(t, reports[0].HasUpdate())
		assert.NotEmpty(t, reports[0].NoUpdateReason)
		assert.Empty(t, reports[0].ImageDiffs)
	})
}

// TestAdviseNarrowsToUsedImages verifies that only images used in the cluster (installed-bundle
// images the InstalledImageSource has) are scanned and diffed: a multi-version bundle ships many
// (repository, name) variants, but only the deployed ones should drive candidate scanning.
func TestAdviseNarrowsToUsedImages(t *testing.T) {
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	const (
		cniRepo = "registry.redhat.io/openshift-service-mesh/istio-cni-rhel9"
		pxyRepo = "registry.redhat.io/openshift-service-mesh/istio-proxyv2-rhel9"
		mgRepo  = "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel9"
		ztRepo  = "registry.redhat.io/openshift-service-mesh/istio-ztunnel-rhel9"
	)

	installedBundle := &Bundle{
		Package: "servicemeshoperator3", ChannelName: "stable", Version: "3.2.0",
		CSVName: "servicemeshoperator3.v3.2.0",
		RelatedImages: []RelatedImage{
			{Image: cniRepo, Name: "images_v1_27_3_cni", Digest: "sha256:cni-used"},      // used
			{Image: pxyRepo, Name: "images_v1_27_3_proxy", Digest: "sha256:pxy-used"},    // used
			{Image: mgRepo, Name: "images_v1_27_3_mustgather", Digest: "sha256:mg-used"}, // used, dropped in candidate
			{Image: cniRepo, Name: "images_v1_24_3_cni", Digest: "sha256:cni-unused"},    // not in cluster
			{Image: pxyRepo, Name: "images_v1_24_3_proxy", Digest: "sha256:pxy-unused"},  // not in cluster
		},
	}
	candidateBundle := Bundle{
		Package: "servicemeshoperator3", ChannelName: "stable", Version: "3.2.9",
		CSVName: "servicemeshoperator3.v3.2.9",
		RelatedImages: []RelatedImage{
			{Image: cniRepo, Name: "images_v1_27_3_cni", Digest: "sha256:cni-new"},
			{Image: pxyRepo, Name: "images_v1_27_3_proxy", Digest: "sha256:pxy-new"},
			{Image: cniRepo, Name: "images_v1_24_3_cni", Digest: "sha256:cni-unused-new"},
			{Image: pxyRepo, Name: "images_v1_24_3_proxy", Digest: "sha256:pxy-unused-new"},
			{Image: ztRepo, Name: "images_v1_28_0_ztunnel", Digest: "sha256:zt-new"}, // brand new, not used
		},
	}

	catalog := &fakeCatalog{
		installed:  map[string]*Bundle{"sha256:cni-used": installedBundle},
		candidates: map[string][]Bundle{"servicemeshoperator3": {candidateBundle}},
	}
	// Only the three deployed images are present in ACS.
	installedSrc := &fakeInstalled{byDigest: map[string]*ImageCVEs{
		"sha256:cni-used": {Repository: cniRepo, Digest: "sha256:cni-used", CVEs: []CVE{cve("CVE-CNI", sev, "")}},
		"sha256:pxy-used": {Repository: pxyRepo, Digest: "sha256:pxy-used", CVEs: []CVE{cve("CVE-PXY", sev, "1.1")}},
		"sha256:mg-used":  {Repository: mgRepo, Digest: "sha256:mg-used", CVEs: []CVE{cve("CVE-MG", sev, "")}},
	}}
	scanner := &fakeScanner{byRef: map[string]*ImageCVEs{
		cniRepo + "@sha256:cni-new": {Repository: cniRepo, Digest: "sha256:cni-new", CVEs: []CVE{cve("CVE-CNI", sev, "")}},
		pxyRepo + "@sha256:pxy-new": {Repository: pxyRepo, Digest: "sha256:pxy-new", CVEs: []CVE{}},
	}}

	advisor := NewAdvisor(catalog, scanner, installedSrc)
	reports, _, err := advisor.Advise(context.Background(), []string{"sha256:cni-used"})
	require.NoError(t, err)
	require.Len(t, reports, 1)

	// Only the two used images that also exist in the candidate are scanned — not the unused
	// 1.24.3 variants nor the brand-new ztunnel.
	assert.ElementsMatch(t, []string{cniRepo + "@sha256:cni-new", pxyRepo + "@sha256:pxy-new"}, scanner.scans,
		"only used images present in the candidate should be scanned")

	byKey := make(map[string]ImageDiff)
	for _, d := range reports[0].ImageDiffs {
		byKey[d.Repository+"|"+d.Name] = d
		assert.NotEqual(t, ImageAdded, d.Status, "unused candidate images must not appear as ADDED")
	}
	require.Len(t, reports[0].ImageDiffs, 3, "two paired used images + one removed (dropped) used image")
	assert.Equal(t, ImagePaired, byKey[cniRepo+"|images_v1_27_3_cni"].Status)
	assert.Equal(t, ImagePaired, byKey[pxyRepo+"|images_v1_27_3_proxy"].Status)
	// must-gather is used but dropped from the candidate -> REMOVED.
	assert.Equal(t, ImageRemoved, byKey[mgRepo+"|images_v1_27_3_mustgather"].Status)
}

// TestAdviseRunningOnly verifies that WithRunningOnly further restricts the analysis to images
// referenced by a running deployment: a scanned-but-not-deployed image is excluded from both
// scanning and the diff, and the deployed source is queried once per bundle (batched).
func TestAdviseRunningOnly(t *testing.T) {
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	const (
		cniRepo = "registry.redhat.io/openshift-service-mesh/istio-cni-rhel9"
		pxyRepo = "registry.redhat.io/openshift-service-mesh/istio-proxyv2-rhel9"
		opRepo  = "registry.redhat.io/openshift-service-mesh/istio-rhel9-operator"
	)
	installedBundle := &Bundle{
		Package: "servicemeshoperator3", ChannelName: "stable", Version: "3.2.0",
		CSVName: "servicemeshoperator3.v3.2.0",
		RelatedImages: []RelatedImage{
			{Image: cniRepo, Name: "images_v1_27_3_cni", Digest: "sha256:cni"},
			{Image: pxyRepo, Name: "images_v1_27_3_proxy", Digest: "sha256:pxy"},
			{Image: opRepo, Name: "sail_operator", Digest: "sha256:op"},
		},
	}
	candidateBundle := Bundle{
		Package: "servicemeshoperator3", ChannelName: "stable", Version: "3.2.9",
		CSVName: "servicemeshoperator3.v3.2.9",
		RelatedImages: []RelatedImage{
			{Image: cniRepo, Name: "images_v1_27_3_cni", Digest: "sha256:cni-new"},
			{Image: pxyRepo, Name: "images_v1_27_3_proxy", Digest: "sha256:pxy-new"},
			{Image: opRepo, Name: "sail_operator", Digest: "sha256:op-new"},
		},
	}
	catalog := &fakeCatalog{
		installed:  map[string]*Bundle{"sha256:op": installedBundle},
		candidates: map[string][]Bundle{"servicemeshoperator3": {candidateBundle}},
	}
	// All three images are scanned/present in ACS.
	installedSrc := &fakeInstalled{byDigest: map[string]*ImageCVEs{
		"sha256:cni": {Repository: cniRepo, Digest: "sha256:cni", CVEs: []CVE{cve("CVE-CNI", sev, "")}},
		"sha256:pxy": {Repository: pxyRepo, Digest: "sha256:pxy", CVEs: []CVE{cve("CVE-PXY", sev, "")}},
		"sha256:op":  {Repository: opRepo, Digest: "sha256:op", CVEs: []CVE{cve("CVE-OP", sev, "")}},
	}}
	scanner := &fakeScanner{byRef: map[string]*ImageCVEs{
		pxyRepo + "@sha256:pxy-new": {Repository: pxyRepo, Digest: "sha256:pxy-new", CVEs: []CVE{}},
		opRepo + "@sha256:op-new":   {Repository: opRepo, Digest: "sha256:op-new", CVEs: []CVE{}},
	}}
	// Only proxy and operator are actually running; cni is scanned but not deployed.
	deployed := &fakeDeployed{deployed: map[string]struct{}{"sha256:pxy": {}, "sha256:op": {}}}

	advisor := NewAdvisor(catalog, scanner, installedSrc, WithRunningOnly(deployed))
	reports, _, err := advisor.Advise(context.Background(), []string{"sha256:op"})
	require.NoError(t, err)
	require.Len(t, reports, 1)

	assert.Equal(t, 1, deployed.calls, "deployed source is queried once per bundle (batched)")
	assert.ElementsMatch(t, []string{pxyRepo + "@sha256:pxy-new", opRepo + "@sha256:op-new"}, scanner.scans,
		"only running images are scanned; cni (scanned-but-not-deployed) is excluded")

	names := make(map[string]bool)
	for _, d := range reports[0].ImageDiffs {
		names[d.Name] = true
	}
	require.Len(t, reports[0].ImageDiffs, 2)
	assert.True(t, names["images_v1_27_3_proxy"])
	assert.True(t, names["sail_operator"])
	assert.False(t, names["images_v1_27_3_cni"], "cni must be excluded by running-only")
}
