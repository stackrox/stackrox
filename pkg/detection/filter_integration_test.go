package detection

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

)

// ── local test helpers ────────────────────────────────────────────────────────

func iComp(name string, layerIdx int32) *storage.EmbeddedImageScanComponent {
	c := &storage.EmbeddedImageScanComponent{Name: name, Version: "1.0"}
	if layerIdx >= 0 {
		c.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: layerIdx}
	}
	return c
}

func iImgWithBaseInfo(maxLayer int32, components ...*storage.EmbeddedImageScanComponent) *storage.Image {
	img := &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "docker.io/test/img"},
		Scan: &storage.ImageScan{Components: components},
	}
	if maxLayer >= 0 {
		img.BaseImageInfo = []*storage.BaseImageInfo{{MaxLayerIndex: maxLayer}}
	}
	return img
}

func iDepWithContainerAndImage(img *storage.Image) *storage.Deployment {
	return &storage.Deployment{
		Id:         uuid.NewV4().String(),
		Containers: []*storage.Container{{Id: img.GetId(), Name: "c0"}},
	}
}

func iImgWithLayersAndComponents(maxLayer int32, layers []*storage.ImageLayer, components ...*storage.EmbeddedImageScanComponent) *storage.Image {
	return &storage.Image{
		Id:            uuid.NewV4().String(),
		Name:          &storage.ImageName{FullName: "docker.io/test/img"},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: maxLayer}},
		Metadata:      &storage.ImageMetadata{V1: &storage.V1Metadata{Layers: layers}},
		Scan:          &storage.ImageScan{Components: components},
	}
}

func iEnhancedDeployment(dep *storage.Deployment, images []*storage.Image) booleanpolicy.EnhancedDeployment {
	return booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     images,
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}
}

func iPolicy(filter *storage.EvaluationFilter, groups ...*storage.PolicyGroup) *storage.Policy {
	return &storage.Policy{
		PolicyVersion:     policyversion.CurrentVersion().String(),
		Name:              uuid.NewV4().String(),
		EventSource:       storage.EventSource_NOT_APPLICABLE,
		PolicySections:    []*storage.PolicySection{{PolicyGroups: groups}},
		EvaluationFilter: filter,
	}
}

func iGroup(fieldName, value string) *storage.PolicyGroup {
	return &storage.PolicyGroup{
		FieldName: fieldName,
		Values:    []*storage.PolicyValue{{Value: value}},
	}
}

func iBaseOnly() *storage.EvaluationFilter {
	return &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP}
}

func iAppOnly() *storage.EvaluationFilter {
	return &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}
}

func iMatchDep(t *testing.T, policy *storage.Policy, dep *storage.Deployment, images []*storage.Image) bool {
	t.Helper()
	m, err := booleanpolicy.BuildDeploymentMatcher(policy)
	require.NoError(t, err)
	v, err := m.MatchDeployment(nil, iEnhancedDeployment(dep, images))
	require.NoError(t, err)
	return len(v.AlertViolations) > 0
}

func iMatchImg(t *testing.T, policy *storage.Policy, img *storage.Image) bool {
	t.Helper()
	m, err := booleanpolicy.BuildImageMatcher(policy)
	require.NoError(t, err)
	v, err := m.MatchImage(nil, img)
	require.NoError(t, err)
	return len(v.AlertViolations) > 0
}

// ── component layer filter ────────────────────────────────────────────────────

func TestIntegration_ComponentLayerFilter(t *testing.T) {
	img := iImgWithBaseInfo(1, iComp("base-a", 0), iComp("base-b", 1), iComp("app", 2))
	dep := iDepWithContainerAndImage(img)
	images := []*storage.Image{img}

	for _, tc := range []struct {
		name      string
		comp      string
		filter    *storage.EvaluationFilter
		wantMatch bool
	}{
		{"base / BASE → match", "base-a", iBaseOnly(), true},
		{"base / APP → no match", "base-a", iAppOnly(), false},
		{"app / APP → match", "app", iAppOnly(), true},
		{"app / BASE → no match", "app", iBaseOnly(), false},
		{"app / no filter → match", "app", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := iPolicy(tc.filter, iGroup(fieldnames.ImageComponent, tc.comp+"="))
			assert.Equal(t, tc.wantMatch, iMatchDep(t, policy, dep, images))
		})
	}
}

func TestIntegration_MultiContainer(t *testing.T) {
	img0 := iImgWithBaseInfo(0, iComp("c", 0), iComp("c", 1))
	img1 := iImgWithBaseInfo(1, iComp("c", 0), iComp("c", 1))
	dep := &storage.Deployment{
		Id: uuid.NewV4().String(),
		Containers: []*storage.Container{
			{Id: img0.GetId(), Name: "c0"},
			{Id: img1.GetId(), Name: "c1"},
		},
	}
	policy := iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "c="))
	assert.True(t, iMatchDep(t, policy, dep, []*storage.Image{img0, img1}))
}

func TestIntegration_BuildTimeImageFilter(t *testing.T) {
	img := iImgWithBaseInfo(1, iComp("base", 0), iComp("app", 2))
	assert.True(t, iMatchImg(t, iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "base=")), img))
	assert.False(t, iMatchImg(t, iPolicy(iAppOnly(), iGroup(fieldnames.ImageComponent, "base=")), img))
	assert.True(t, iMatchImg(t, iPolicy(iAppOnly(), iGroup(fieldnames.ImageComponent, "app=")), img))
	assert.False(t, iMatchImg(t, iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "app=")), img))
}

// ── Dockerfile layer filter ───────────────────────────────────────────────────

func TestIntegration_DockerfileLayerFilter(t *testing.T) {
	img := &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "docker.io/test/img"},
		Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{Instruction: "FROM", Value: "base:latest"},
				{Instruction: "RUN", Value: "base_cmd"},
				{Instruction: "COPY", Value: "app/ /app"},
				{Instruction: "RUN", Value: "app_cmd"},
			},
		}},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 1}},
	}
	dep := iDepWithContainerAndImage(img)
	images := []*storage.Image{img}

	assert.True(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.DockerfileLine, "RUN=base_cmd")), dep, images), "base RUN visible under BASE")
	assert.False(t, iMatchDep(t, iPolicy(iAppOnly(), iGroup(fieldnames.DockerfileLine, "RUN=base_cmd")), dep, images), "base RUN hidden under APP")
	assert.True(t, iMatchDep(t, iPolicy(iAppOnly(), iGroup(fieldnames.DockerfileLine, "RUN=app_cmd")), dep, images), "app RUN visible under APP")
	assert.False(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.DockerfileLine, "RUN=app_cmd")), dep, images), "app RUN hidden under BASE")
	assert.False(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.DockerfileLine, "COPY=app/ /app")), dep, images), "COPY hidden under BASE")
	assert.True(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.DockerfileLine, "FROM=base:latest")), dep, images), "FROM visible under BASE")
}

// ── REQ-12 ────────────────────────────────────────────────────────────────────

func TestIntegration_REQ12_NoBaseImageInfo(t *testing.T) {
	img := iImgWithBaseInfo(-1, iComp("x", 2))
	dep := iDepWithContainerAndImage(img)
	images := []*storage.Image{img}
	assert.False(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "x=")), dep, images))
	assert.True(t, iMatchDep(t, iPolicy(iAppOnly(), iGroup(fieldnames.ImageComponent, "x=")), dep, images))
}

func TestIntegration_REQ12_NilLayerIndex(t *testing.T) {
	img := iImgWithBaseInfo(1, iComp("x", -1))
	dep := iDepWithContainerAndImage(img)
	images := []*storage.Image{img}
	assert.False(t, iMatchDep(t, iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "x=")), dep, images))
	assert.True(t, iMatchDep(t, iPolicy(iAppOnly(), iGroup(fieldnames.ImageComponent, "x=")), dep, images))
}

// ── Cache consistency ─────────────────────────────────────────────────────────

func TestIntegration_CacheConsistency(t *testing.T) {
	img := iImgWithBaseInfo(1, iComp("base", 0), iComp("app", 2))
	dep := iDepWithContainerAndImage(img)
	ed := iEnhancedDeployment(dep, []*storage.Image{img})

	baseM, err := booleanpolicy.BuildDeploymentMatcher(iPolicy(iBaseOnly(), iGroup(fieldnames.ImageComponent, "base=")))
	require.NoError(t, err)
	appM, err := booleanpolicy.BuildDeploymentMatcher(iPolicy(iAppOnly(), iGroup(fieldnames.ImageComponent, "app=")))
	require.NoError(t, err)

	var cache booleanpolicy.CacheReceptacle
	for i := 0; i < 3; i++ {
		v, err := baseM.MatchDeployment(&cache, ed)
		require.NoError(t, err)
		assert.NotEmpty(t, v.AlertViolations, "iteration %d BASE", i)

		v, err = appM.MatchDeployment(&cache, ed)
		require.NoError(t, err)
		assert.NotEmpty(t, v.AlertViolations, "iteration %d APP", i)
	}

	allM, err := booleanpolicy.BuildDeploymentMatcher(iPolicy(nil, iGroup(fieldnames.ImageComponent, "base=")))
	require.NoError(t, err)
	v, err := allM.MatchDeployment(&cache, ed)
	require.NoError(t, err)
	assert.NotEmpty(t, v.AlertViolations, "unfiltered policy sees base via shared cache")
}

// ── Cross-branch logical connection ──────────────────────────────────────────

func TestIntegration_CrossBranchLogicalConnection(t *testing.T) {
	img := iImgWithLayersAndComponents(
		1,
		[]*storage.ImageLayer{
			{Instruction: "FROM", Value: "base:latest"},
			{Instruction: "RUN", Value: "install-base"},
			{Instruction: "COPY", Value: ". /app"},
			{Instruction: "RUN", Value: "install-app"},
		},
		iComp("base-lib", 0),
		iComp("app-lib", 2),
	)
	dep := iDepWithContainerAndImage(img)
	images := []*storage.Image{img}

	crossPolicy := func(layer, comp string, filter *storage.EvaluationFilter) *storage.Policy {
		return iPolicy(filter, iGroup(fieldnames.DockerfileLine, layer), iGroup(fieldnames.ImageComponent, comp))
	}

	assert.True(t, iMatchDep(t, crossPolicy("RUN=install-base", "base-lib=", iBaseOnly()), dep, images), "BASE: base RUN + base lib → match")
	assert.False(t, iMatchDep(t, crossPolicy("RUN=install-base", "app-lib=", iBaseOnly()), dep, images), "BASE: app-lib pruned → no match")
	assert.True(t, iMatchDep(t, crossPolicy("RUN=install-app", "app-lib=", iAppOnly()), dep, images), "APP: app RUN + app lib → match")
	assert.False(t, iMatchDep(t, crossPolicy("RUN=install-app", "base-lib=", iAppOnly()), dep, images), "APP: base-lib pruned → no match")
	assert.True(t, iMatchDep(t, crossPolicy("RUN=install-base", "app-lib=", nil), dep, images), "no filter: loose conjunction fires")
}
