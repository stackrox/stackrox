package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestEvaluationFilter_ZeroValue(t *testing.T) {
	var f EvaluationFilter
	assert.False(t, f.IsNonDefault())
}

func TestEvaluationFilter_Apply_RemovesContainersByName(t *testing.T) {
	f := EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			var kept []*storage.Container
			var keptImgs []*storage.Image
			for i, c := range dep.GetContainers() {
				if c.GetName() != "excluded" {
					kept = append(kept, c)
					if i < len(imgs) {
						keptImgs = append(keptImgs, imgs[i])
					}
				}
			}
			if len(kept) == len(dep.GetContainers()) {
				return dep, imgs
			}
			cloned := dep.CloneVT()
			cloned.Containers = kept
			return cloned, keptImgs
		},
	}

	dep := &storage.Deployment{
		Containers: []*storage.Container{
			{Name: "app"},
			{Name: "excluded"},
			{Name: "sidecar"},
		},
	}
	imgs := []*storage.Image{
		{Id: "img-app"},
		{Id: "img-excluded"},
		{Id: "img-sidecar"},
	}

	assert.True(t, f.IsNonDefault())

	resultDep, resultImgs := f.Apply(dep, imgs)
	assert.Len(t, resultDep.GetContainers(), 2)
	assert.Equal(t, "app", resultDep.GetContainers()[0].GetName())
	assert.Equal(t, "sidecar", resultDep.GetContainers()[1].GetName())
	assert.Len(t, resultImgs, 2)
	assert.Equal(t, "img-app", resultImgs[0].GetId())
	assert.Equal(t, "img-sidecar", resultImgs[1].GetId())

	// Original not mutated.
	assert.Len(t, dep.GetContainers(), 3)
}

func TestEvaluationFilter_Apply_FiltersImageComponents(t *testing.T) {
	f := EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			result := make([]*storage.Image, len(imgs))
			for i, img := range imgs {
				if img == nil || img.GetScan() == nil {
					result[i] = img
					continue
				}
				var kept []*storage.EmbeddedImageScanComponent
				for _, c := range img.GetScan().GetComponents() {
					if c.GetName() != "excluded-pkg" {
						kept = append(kept, c)
					}
				}
				if len(kept) == len(img.GetScan().GetComponents()) {
					result[i] = img
					continue
				}
				cloned := img.CloneVT()
				cloned.Scan.Components = kept
				result[i] = cloned
			}
			return dep, result
		},
	}

	dep := &storage.Deployment{}
	img := &storage.Image{
		Id: "test-img",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "kept-pkg"},
				{Name: "excluded-pkg"},
				{Name: "another-kept"},
			},
		},
	}

	_, resultImgs := f.Apply(dep, []*storage.Image{img})
	assert.Len(t, resultImgs[0].GetScan().GetComponents(), 2)
	assert.Equal(t, "kept-pkg", resultImgs[0].GetScan().GetComponents()[0].GetName())
	assert.Equal(t, "another-kept", resultImgs[0].GetScan().GetComponents()[1].GetName())

	// Original not mutated.
	assert.Len(t, img.GetScan().GetComponents(), 3)
}

func TestEvaluationFilter_Apply_ChainingFilters(t *testing.T) {
	containerFilter := EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			var kept []*storage.Container
			var keptImgs []*storage.Image
			for i, c := range dep.GetContainers() {
				if c.GetName() != "excluded" {
					kept = append(kept, c)
					if i < len(imgs) {
						keptImgs = append(keptImgs, imgs[i])
					}
				}
			}
			if len(kept) == len(dep.GetContainers()) {
				return dep, imgs
			}
			cloned := dep.CloneVT()
			cloned.Containers = kept
			return cloned, keptImgs
		},
	}

	imageFilter := EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			result := make([]*storage.Image, len(imgs))
			for i, img := range imgs {
				if img == nil || img.GetScan() == nil {
					result[i] = img
					continue
				}
				var kept []*storage.EmbeddedImageScanComponent
				for _, c := range img.GetScan().GetComponents() {
					if c.GetName() != "excluded-pkg" {
						kept = append(kept, c)
					}
				}
				if len(kept) == len(img.GetScan().GetComponents()) {
					result[i] = img
					continue
				}
				cloned := img.CloneVT()
				cloned.Scan.Components = kept
				result[i] = cloned
			}
			return dep, result
		},
	}

	dep := &storage.Deployment{
		Containers: []*storage.Container{
			{Name: "app"},
			{Name: "excluded"},
		},
	}
	imgs := []*storage.Image{
		{Id: "app-img", Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "kept-pkg"},
				{Name: "excluded-pkg"},
			},
		}},
		{Id: "excluded-img"},
	}

	// Chain: container filter first, then image filter on survivors.
	resultDep, resultImgs := dep, imgs
	for _, f := range []EvaluationFilter{containerFilter, imageFilter} {
		resultDep, resultImgs = f.Apply(resultDep, resultImgs)
	}

	assert.Len(t, resultDep.GetContainers(), 1)
	assert.Equal(t, "app", resultDep.GetContainers()[0].GetName())
	assert.Len(t, resultImgs, 1)
	assert.Len(t, resultImgs[0].GetScan().GetComponents(), 1)
	assert.Equal(t, "kept-pkg", resultImgs[0].GetScan().GetComponents()[0].GetName())
}

func TestEvaluationFilter_Apply_NoOpFilter(t *testing.T) {
	f := EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			return dep, imgs
		},
	}

	dep := &storage.Deployment{
		Containers: []*storage.Container{{Name: "app"}},
	}
	imgs := []*storage.Image{{Id: "img"}}

	resultDep, resultImgs := f.Apply(dep, imgs)
	assert.True(t, dep == resultDep, "expected same deployment pointer")
	assert.True(t, &imgs[0] == &resultImgs[0], "expected same images slice")
}
