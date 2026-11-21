package booleanpolicy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/filter"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// ROX-35476: SkipImageLayers evaluation-filter violation-generation tests.
//
// These exercise the production filter.CompileEvaluationFilter +
// filter.(EvaluationFilter).Apply path (gated on features.EvaluationFilter AND
// features.ImageLayerFilter) ahead of the real image matcher, using hand-built
// in-memory fixtures. Assertions are on stable domain markers (CVE ids /
// component names) via substring matching on violation messages.
// ---------------------------------------------------------------------------

func TestBaseAppImageLayerDifferentiation(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")
	t.Setenv(features.ImageLayerFilter.EnvVar(), "true")
	suite.Run(t, new(baseAppImageLayerDiffTestSuite))
}

type baseAppImageLayerDiffTestSuite struct {
	suite.Suite
}

// imageLayerViolations runs the real evaluation-filter compile/apply pipeline on
// img for policy, then matches the (possibly filtered) image and returns the
// violation messages.
func imageLayerViolations(t testing.TB, img *storage.Image, policy *storage.Policy) []string {
	t.Helper()
	filtered := img
	for _, f := range filter.CompileEvaluationFilter(policy.GetEvaluationFilter()) {
		_, imgs := f.Apply(nil, []*storage.Image{filtered})
		require.Len(t, imgs, 1)
		filtered = imgs[0]
	}
	matcher, err := BuildImageMatcher(policy)
	require.NoError(t, err)
	v, err := matcher.MatchImage(nil, filtered)
	require.NoError(t, err)
	msgs := make([]string, 0, len(v.AlertViolations))
	for _, av := range v.AlertViolations {
		msgs = append(msgs, av.GetMessage())
	}
	return msgs
}

func assertContainsMarker(t testing.TB, msgs []string, marker string) {
	t.Helper()
	var found bool
	for _, m := range msgs {
		if strings.Contains(m, marker) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a violation mentioning %q, got %v", marker, msgs)
}

func assertNotContainsMarker(t testing.TB, msgs []string, marker string) {
	t.Helper()
	var found bool
	for _, m := range msgs {
		if strings.Contains(m, marker) {
			found = true
			break
		}
	}
	assert.False(t, found, "did not expect a violation mentioning %q, got %v", marker, msgs)
}

func fixableVuln(cve string, cvss float32) *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:        cve,
		Cvss:       cvss,
		Link:       "https://" + cve,
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v9.9"},
	}
}

func plainVuln(cve string, cvss float32) *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:  cve,
		Cvss: cvss,
		Link: "https://" + cve,
	}
}

func layeredComp(name, version string, layerIdx int32, vulns ...*storage.EmbeddedVulnerability) *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          name,
		Version:       version,
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: layerIdx},
		Vulns:         vulns,
	}
}

func unlayeredComp(name, version string, vulns ...*storage.EmbeddedVulnerability) *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    name,
		Version: version,
		Vulns:   vulns,
	}
}

// layeredImage builds an image whose base/app boundary is maxBaseLayer. A
// negative maxBaseLayer omits BaseImageInfo entirely (filter no-op case).
func layeredImage(maxBaseLayer int32, comps ...*storage.EmbeddedImageScanComponent) *storage.Image {
	img := &storage.Image{
		Id:   fmt.Sprintf("img-layerfilter-max%d-comps%d", maxBaseLayer, len(comps)),
		Name: &storage.ImageName{FullName: "docker.io/layerfilter"},
		Scan: &storage.ImageScan{Components: comps},
	}
	if maxBaseLayer >= 0 {
		img.BaseImageInfo = []*storage.BaseImageInfo{{MaxLayerIndex: maxBaseLayer}}
	}
	return img
}

func imageLayerPolicy(skip storage.SkipImageLayers, groups ...*storage.PolicyGroup) *storage.Policy {
	p := policyWithGroups(storage.EventSource_NOT_APPLICABLE, groups...)
	p.LifecycleStages = []storage.LifecycleStage{storage.LifecycleStage_BUILD}
	p.EvaluationFilter = &storage.EvaluationFilter{SkipImageLayers: skip}
	return p
}

// Group A: CVSS "> 7" splits base/app CVEs by SkipImageLayers.
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_CVSS() {
	img := layeredImage(3,
		layeredComp("base-lib", "1.0", 1, plainVuln("CVE-BASE-HIGH", 9)),
		layeredComp("app-lib", "2.0", 5, plainVuln("CVE-APP-HIGH", 9)),
	)
	for _, tc := range []struct {
		name    string
		skip    storage.SkipImageLayers
		present []string
		absent  []string
	}{
		{"SKIP_NONE keeps both", storage.SkipImageLayers_SKIP_NONE, []string{"CVE-BASE-HIGH", "CVE-APP-HIGH"}, nil},
		{"SKIP_BASE keeps app only", storage.SkipImageLayers_SKIP_BASE, []string{"CVE-APP-HIGH"}, []string{"CVE-BASE-HIGH"}},
		{"SKIP_APP keeps base only", storage.SkipImageLayers_SKIP_APP, []string{"CVE-BASE-HIGH"}, []string{"CVE-APP-HIGH"}},
	} {
		suite.Run(tc.name, func() {
			policy := imageLayerPolicy(tc.skip, policyGroupWithSingleKeyValue(fieldnames.CVSS, "> 7", false))
			msgs := imageLayerViolations(suite.T(), img, policy)
			for _, m := range tc.present {
				assertContainsMarker(suite.T(), msgs, m)
			}
			for _, m := range tc.absent {
				assertNotContainsMarker(suite.T(), msgs, m)
			}
		})
	}
}

// Group B: Fixable "true" splits base/app, plus a CVE-field variant where
// SKIP_APP suppresses an otherwise-matching app CVE entirely (zero violations).
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_Fixable() {
	img := layeredImage(3,
		layeredComp("base-lib", "1.0", 1, fixableVuln("CVE-BASE-FIX", 9)),
		layeredComp("app-lib", "2.0", 5, fixableVuln("CVE-APP-FIX", 9)),
	)
	for _, tc := range []struct {
		name    string
		skip    storage.SkipImageLayers
		present []string
		absent  []string
	}{
		{"SKIP_NONE keeps both", storage.SkipImageLayers_SKIP_NONE, []string{"CVE-BASE-FIX", "CVE-APP-FIX"}, nil},
		{"SKIP_BASE keeps app only", storage.SkipImageLayers_SKIP_BASE, []string{"CVE-APP-FIX"}, []string{"CVE-BASE-FIX"}},
		{"SKIP_APP keeps base only", storage.SkipImageLayers_SKIP_APP, []string{"CVE-BASE-FIX"}, []string{"CVE-APP-FIX"}},
	} {
		suite.Run("Fixable "+tc.name, func() {
			policy := imageLayerPolicy(tc.skip, policyGroupWithSingleKeyValue(fieldnames.Fixable, "true", false))
			msgs := imageLayerViolations(suite.T(), img, policy)
			for _, m := range tc.present {
				assertContainsMarker(suite.T(), msgs, m)
			}
			for _, m := range tc.absent {
				assertNotContainsMarker(suite.T(), msgs, m)
			}
		})
	}

	// CVE-field variant: a policy pinned to the app CVE id yields zero
	// violations once the app layer is skipped, and one otherwise.
	for _, tc := range []struct {
		name          string
		skip          storage.SkipImageLayers
		wantViolation bool
	}{
		{"SKIP_APP suppresses the app CVE entirely", storage.SkipImageLayers_SKIP_APP, false},
		{"SKIP_BASE keeps the app CVE", storage.SkipImageLayers_SKIP_BASE, true},
		{"SKIP_NONE keeps the app CVE", storage.SkipImageLayers_SKIP_NONE, true},
	} {
		suite.Run("CVE field "+tc.name, func() {
			policy := imageLayerPolicy(tc.skip, policyGroupWithSingleKeyValue(fieldnames.CVE, "CVE-APP-FIX", false))
			msgs := imageLayerViolations(suite.T(), img, policy)
			if tc.wantViolation {
				assert.Len(suite.T(), msgs, 1, "expected exactly the app CVE violation, got %v", msgs)
				assertContainsMarker(suite.T(), msgs, "CVE-APP-FIX")
			} else {
				assert.Empty(suite.T(), msgs, "expected the app CVE to be filtered out, got %v", msgs)
			}
		})
	}
}

// Group C: Image Component (two values in one group) splits base/app by name.
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_ImageComponent() {
	img := layeredImage(3,
		layeredComp("base-lib", "1.0", 1, plainVuln("CVE-BASE-COMP", 9)),
		layeredComp("app-lib", "2.0", 5, plainVuln("CVE-APP-COMP", 9)),
	)
	componentGroup := &storage.PolicyGroup{
		FieldName: fieldnames.ImageComponent,
		Values: []*storage.PolicyValue{
			{Value: "base-lib=1.0"},
			{Value: "app-lib=2.0"},
		},
	}
	for _, tc := range []struct {
		name    string
		skip    storage.SkipImageLayers
		present []string
		absent  []string
	}{
		{"SKIP_NONE keeps both", storage.SkipImageLayers_SKIP_NONE, []string{"base-lib", "app-lib"}, nil},
		{"SKIP_BASE keeps app only", storage.SkipImageLayers_SKIP_BASE, []string{"app-lib"}, []string{"base-lib"}},
		{"SKIP_APP keeps base only", storage.SkipImageLayers_SKIP_APP, []string{"base-lib"}, []string{"app-lib"}},
	} {
		suite.Run(tc.name, func() {
			policy := imageLayerPolicy(tc.skip, componentGroup)
			msgs := imageLayerViolations(suite.T(), img, policy)
			for _, m := range tc.present {
				assertContainsMarker(suite.T(), msgs, m)
			}
			for _, m := range tc.absent {
				assertNotContainsMarker(suite.T(), msgs, m)
			}
		})
	}
}

// Group D: a missing BaseImageInfo makes the filter a no-op even under SKIP_BASE.
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_MissingBaseImageInfo() {
	img := layeredImage(-1,
		layeredComp("base-lib", "1.0", 1, plainVuln("CVE-BASE-HIGH", 9)),
		layeredComp("app-lib", "2.0", 5, plainVuln("CVE-APP-HIGH", 9)),
	)
	policy := imageLayerPolicy(storage.SkipImageLayers_SKIP_BASE, policyGroupWithSingleKeyValue(fieldnames.CVSS, "> 7", false))
	msgs := imageLayerViolations(suite.T(), img, policy)
	assertContainsMarker(suite.T(), msgs, "CVE-BASE-HIGH")
	assertContainsMarker(suite.T(), msgs, "CVE-APP-HIGH")
}

// Group E: boundary is inclusive to base (idx == maxBaseLayer counts as base).
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_BoundaryInclusive() {
	img := layeredImage(3,
		layeredComp("boundary-lib", "1.0", 3, plainVuln("CVE-BOUND-HIGH", 9)),
	)
	for _, tc := range []struct {
		name    string
		skip    storage.SkipImageLayers
		present bool
	}{
		{"SKIP_APP keeps the boundary (base) layer", storage.SkipImageLayers_SKIP_APP, true},
		{"SKIP_BASE drops the boundary (base) layer", storage.SkipImageLayers_SKIP_BASE, false},
	} {
		suite.Run(tc.name, func() {
			policy := imageLayerPolicy(tc.skip, policyGroupWithSingleKeyValue(fieldnames.CVSS, "> 7", false))
			msgs := imageLayerViolations(suite.T(), img, policy)
			if tc.present {
				assertContainsMarker(suite.T(), msgs, "CVE-BOUND-HIGH")
			} else {
				assertNotContainsMarker(suite.T(), msgs, "CVE-BOUND-HIGH")
			}
		})
	}
}

// Group F: a component with no layer index is always kept, regardless of skip.
func (suite *baseAppImageLayerDiffTestSuite) TestImageLayerFilter_NoLayerIndex() {
	img := layeredImage(3,
		unlayeredComp("free-lib", "1.0", plainVuln("CVE-NOLAYER-HIGH", 9)),
	)
	for _, skip := range []storage.SkipImageLayers{storage.SkipImageLayers_SKIP_BASE, storage.SkipImageLayers_SKIP_APP} {
		suite.Run(skip.String(), func() {
			policy := imageLayerPolicy(skip, policyGroupWithSingleKeyValue(fieldnames.CVSS, "> 7", false))
			msgs := imageLayerViolations(suite.T(), img, policy)
			assertContainsMarker(suite.T(), msgs, "CVE-NOLAYER-HIGH")
		})
	}
}

// Group G: with features.ImageLayerFilter off, the filter is inert even when the
// policy requests SKIP_BASE; both CVEs remain.
func TestImageLayerFilter_FlagOff(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")
	t.Setenv(features.ImageLayerFilter.EnvVar(), "false")

	img := layeredImage(3,
		layeredComp("base-lib", "1.0", 1, plainVuln("CVE-BASE-HIGH", 9)),
		layeredComp("app-lib", "2.0", 5, plainVuln("CVE-APP-HIGH", 9)),
	)
	policy := imageLayerPolicy(storage.SkipImageLayers_SKIP_BASE, policyGroupWithSingleKeyValue(fieldnames.CVSS, "> 7", false))
	msgs := imageLayerViolations(t, img, policy)
	assertContainsMarker(t, msgs, "CVE-BASE-HIGH")
	assertContainsMarker(t, msgs, "CVE-APP-HIGH")
}
