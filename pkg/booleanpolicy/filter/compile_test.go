package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
)

func TestCompileEvaluationFilter(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")
	cases := map[string]struct {
		proto    *storage.EvaluationFilter
		expected int
	}{
		"nil proto": {
			proto:    nil,
			expected: 0,
		},
		"empty proto": {
			proto:    &storage.EvaluationFilter{},
			expected: 0,
		},
		"SKIP_NONE image layers only": {
			proto: &storage.EvaluationFilter{
				SkipImageLayers: storage.SkipImageLayers_SKIP_NONE,
			},
			expected: 0,
		},
		"skip init containers only": {
			proto: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			},
			expected: 1,
		},
		"skip base layers only": {
			proto: &storage.EvaluationFilter{
				SkipImageLayers: storage.SkipImageLayers_SKIP_BASE,
			},
			expected: 1,
		},
		"both container and image filters": {
			proto: &storage.EvaluationFilter{
				SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
				SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
			},
			expected: 2,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			filters := CompileEvaluationFilter(tc.proto)
			assert.Len(t, filters, tc.expected)
			for _, f := range filters {
				assert.True(t, f.IsNonDefault())
			}
		})
	}
}

func TestCompileEvaluationFilter_CombinedApply(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")
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

	dep := &storage.Deployment{
		Containers: []*storage.Container{regularContainer, initContainer},
	}
	images := []*storage.Image{appImage, initImage}

	t.Run("nil proto returns no filters", func(t *testing.T) {
		filters := CompileEvaluationFilter(nil)
		assert.Nil(t, filters)
	})

	t.Run("skip init + skip base combined", func(t *testing.T) {
		filters := CompileEvaluationFilter(&storage.EvaluationFilter{
			SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		})
		assert.Len(t, filters, 2)

		resultDep, resultImgs := dep, images
		for _, f := range filters {
			resultDep, resultImgs = f.Apply(resultDep, resultImgs)
		}

		assert.Len(t, resultDep.GetContainers(), 1)
		assert.Equal(t, "app", resultDep.GetContainers()[0].GetName())

		assert.Len(t, resultImgs, 1)
		comps := resultImgs[0].GetScan().GetComponents()
		assert.Len(t, comps, 1)
		assert.Equal(t, "app-pkg", comps[0].GetName())
	})
}
