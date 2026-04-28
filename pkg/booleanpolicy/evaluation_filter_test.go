package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestIsNonDefaultFilter(t *testing.T) {
	cases := map[string]struct {
		filter   *storage.EvaluationFilter
		expected bool
	}{
		"nil filter": {
			filter:   nil,
			expected: false,
		},
		"empty filter": {
			filter:   &storage.EvaluationFilter{},
			expected: false,
		},
		"SKIP_NONE image layers": {
			filter: &storage.EvaluationFilter{
				SkipImageLayers: storage.SkipImageLayers_SKIP_NONE,
			},
			expected: false,
		},
		"skip init containers": {
			filter: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			},
			expected: true,
		},
		"skip base layers": {
			filter: &storage.EvaluationFilter{
				SkipImageLayers: storage.SkipImageLayers_SKIP_BASE,
			},
			expected: true,
		},
		"skip app layers": {
			filter: &storage.EvaluationFilter{
				SkipImageLayers: storage.SkipImageLayers_SKIP_APP,
			},
			expected: true,
		},
		"both container and image filters": {
			filter: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
				SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
			},
			expected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isNonDefaultFilter(tc.filter))
		})
	}
}

func TestFilterContainersBySkipTypes(t *testing.T) {
	regularContainer := &storage.Container{
		Name:  "app",
		Type:  storage.ContainerType_REGULAR,
		Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "app:latest"}},
	}
	initContainer := &storage.Container{
		Name:  "init",
		Type:  storage.ContainerType_INIT,
		Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "init:latest"}},
	}

	appImage := &storage.Image{Id: "app-img"}
	initImage := &storage.Image{Id: "init-img"}

	cases := map[string]struct {
		containers         []*storage.Container
		images             []*storage.Image
		filter             *storage.EvaluationFilter
		expectedContainers int
		expectedImageIDs   []string
	}{
		"no skip - all containers kept": {
			containers: []*storage.Container{regularContainer, initContainer},
			images:     []*storage.Image{appImage, initImage},
			filter:     &storage.EvaluationFilter{},
			expectedContainers: 2,
			expectedImageIDs:   []string{"app-img", "init-img"},
		},
		"skip init - only regular kept": {
			containers: []*storage.Container{regularContainer, initContainer},
			images:     []*storage.Image{appImage, initImage},
			filter: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			},
			expectedContainers: 1,
			expectedImageIDs:   []string{"app-img"},
		},
		"skip init - no init containers present": {
			containers: []*storage.Container{regularContainer},
			images:     []*storage.Image{appImage},
			filter: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			},
			expectedContainers: 1,
			expectedImageIDs:   []string{"app-img"},
		},
		"skip init - all init containers": {
			containers: []*storage.Container{initContainer},
			images:     []*storage.Image{initImage},
			filter: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			},
			expectedContainers: 0,
			expectedImageIDs:   nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dep := &storage.Deployment{Containers: tc.containers}
			filteredDep, filteredImgs := filterContainersBySkipTypes(dep, tc.images, tc.filter)

			assert.Len(t, filteredDep.GetContainers(), tc.expectedContainers)
			var imgIDs []string
			for _, img := range filteredImgs {
				imgIDs = append(imgIDs, img.GetId())
			}
			assert.Equal(t, tc.expectedImageIDs, imgIDs)

			// Original deployment should not be mutated.
			assert.Len(t, dep.GetContainers(), len(tc.containers))
		})
	}
}

func TestFilterImagesByLayer(t *testing.T) {
	baseComponent := &storage.EmbeddedImageScanComponent{
		Name:          "base-pkg",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
	}
	appComponent := &storage.EmbeddedImageScanComponent{
		Name:          "app-pkg",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
	}
	noLayerComponent := &storage.EmbeddedImageScanComponent{
		Name: "unknown-layer-pkg",
	}

	makeImage := func(components ...*storage.EmbeddedImageScanComponent) *storage.Image {
		return &storage.Image{
			Id: "test-img",
			Scan: &storage.ImageScan{
				Components: components,
			},
			BaseImageInfo: []*storage.BaseImageInfo{
				{MaxLayerIndex: 3},
			},
		}
	}

	cases := map[string]struct {
		image              *storage.Image
		skip               storage.SkipImageLayers
		expectedComponents []string
	}{
		"SKIP_NONE keeps all": {
			image:              makeImage(baseComponent, appComponent, noLayerComponent),
			skip:               storage.SkipImageLayers_SKIP_NONE,
			expectedComponents: []string{"base-pkg", "app-pkg", "unknown-layer-pkg"},
		},
		"SKIP_BASE keeps only app-layer and unknown components": {
			image:              makeImage(baseComponent, appComponent, noLayerComponent),
			skip:               storage.SkipImageLayers_SKIP_BASE,
			expectedComponents: []string{"app-pkg", "unknown-layer-pkg"},
		},
		"SKIP_APP keeps only base-layer and unknown components": {
			image:              makeImage(baseComponent, appComponent, noLayerComponent),
			skip:               storage.SkipImageLayers_SKIP_APP,
			expectedComponents: []string{"base-pkg", "unknown-layer-pkg"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			filter := &storage.EvaluationFilter{SkipImageLayers: tc.skip}
			result := filterImagesByLayer([]*storage.Image{tc.image}, filter)

			var names []string
			for _, c := range result[0].GetScan().GetComponents() {
				names = append(names, c.GetName())
			}
			assert.Equal(t, tc.expectedComponents, names)
		})
	}
}

func TestFilterImagesByLayer_NoBaseImageInfo(t *testing.T) {
	component := &storage.EmbeddedImageScanComponent{
		Name:          "some-pkg",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2},
	}
	img := &storage.Image{
		Id: "no-base-info",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{component},
		},
	}

	filter := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}
	result := filterImagesByLayer([]*storage.Image{img}, filter)

	assert.Len(t, result[0].GetScan().GetComponents(), 1,
		"without BaseImageInfo, image should be returned unmodified")
}

func TestFilterImagesByLayer_NilImage(t *testing.T) {
	filter := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}
	result := filterImagesByLayer([]*storage.Image{nil}, filter)
	assert.Nil(t, result[0])
}

func TestApplyEvaluationFilter(t *testing.T) {
	initContainer := &storage.Container{
		Name: "init",
		Type: storage.ContainerType_INIT,
	}
	regularContainer := &storage.Container{
		Name: "app",
		Type: storage.ContainerType_REGULAR,
	}

	baseComp := &storage.EmbeddedImageScanComponent{
		Name:          "base-pkg",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
	}
	appComp := &storage.EmbeddedImageScanComponent{
		Name:          "app-pkg",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
	}

	initImage := &storage.Image{Id: "init-img"}
	appImage := &storage.Image{
		Id: "app-img",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{baseComp, appComp},
		},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 3}},
	}

	ed := EnhancedDeployment{
		Deployment: &storage.Deployment{
			Containers: []*storage.Container{regularContainer, initContainer},
		},
		Images: []*storage.Image{appImage, initImage},
	}

	t.Run("default filter returns original", func(t *testing.T) {
		result := applyEvaluationFilter(ed, nil)
		assert.Equal(t, ed.Deployment, result.Deployment)
		assert.Equal(t, ed.Images, result.Images)
	})

	t.Run("skip init + skip base combined", func(t *testing.T) {
		filter := &storage.EvaluationFilter{
			SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		}
		result := applyEvaluationFilter(ed, filter)

		assert.Len(t, result.Deployment.GetContainers(), 1)
		assert.Equal(t, "app", result.Deployment.GetContainers()[0].GetName())

		assert.Len(t, result.Images, 1)
		comps := result.Images[0].GetScan().GetComponents()
		assert.Len(t, comps, 1)
		assert.Equal(t, "app-pkg", comps[0].GetName())
	})
}
