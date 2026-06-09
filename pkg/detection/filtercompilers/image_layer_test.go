package filtercompilers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stretchr/testify/assert"
)

func mkComp(name string, layerIdx int32) *storage.EmbeddedImageScanComponent {
	c := &storage.EmbeddedImageScanComponent{Name: name, Version: "1.0"}
	if layerIdx >= 0 {
		c.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: layerIdx}
	}
	return c
}

const maxLayer = int32(1) // layers 0,1 → base; 2+ → app

func TestIsBaseComponent(t *testing.T) {
	for name, tc := range map[string]struct {
		comp  *storage.EmbeddedImageScanComponent
		hasMax bool
		want  bool
	}{
		"layer 0 (base)": {mkComp("c", 0), true, true},
		"layer 1 (base)": {mkComp("c", 1), true, true},
		"layer 2 (app)": {mkComp("c", 2), true, false},
		"no baseinfo (REQ-12)": {mkComp("c", 0), false, false},
		"nil layerIdx (REQ-12)": {mkComp("c", -1), true, false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isBaseComponent(tc.comp, maxLayer, tc.hasMax))
		})
	}
}

func TestIsBaseLayer(t *testing.T) {
	for name, tc := range map[string]struct {
		i      int
		hasMax bool
		want   bool
	}{
		"i=0 (base)": {0, true, true},
		"i=1 (base)": {1, true, true},
		"i=2 (app)": {2, true, false},
		"no baseinfo (REQ-12)": {0, false, false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isBaseLayer(tc.i, maxLayer, tc.hasMax))
		})
	}
}

func TestKeepComponent(t *testing.T) {
	for name, tc := range map[string]struct {
		comp   *storage.EmbeddedImageScanComponent
		hasMax bool
		skip   storage.SkipImageLayers
		want   bool
	}{
		"base / SKIP_APP → keep":  {mkComp("c", 0), true, storage.SkipImageLayers_SKIP_APP, true},
		"base / SKIP_BASE → skip": {mkComp("c", 0), true, storage.SkipImageLayers_SKIP_BASE, false},
		"app / SKIP_BASE → keep":  {mkComp("c", 2), true, storage.SkipImageLayers_SKIP_BASE, true},
		"app / SKIP_APP → skip":   {mkComp("c", 2), true, storage.SkipImageLayers_SKIP_APP, false},
		"no baseinfo / SKIP_BASE → keep (treat as app)": {mkComp("c", 0), false, storage.SkipImageLayers_SKIP_BASE, true},
		"no baseinfo / SKIP_APP → skip (treat as app)":  {mkComp("c", 0), false, storage.SkipImageLayers_SKIP_APP, false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, keepComponent(tc.comp, maxLayer, tc.hasMax, tc.skip))
		})
	}
}

func TestKeepLayerByIndex(t *testing.T) {
	for name, tc := range map[string]struct {
		i      int
		hasMax bool
		skip   storage.SkipImageLayers
		want   bool
	}{
		"base / SKIP_APP → keep":  {0, true, storage.SkipImageLayers_SKIP_APP, true},
		"base / SKIP_BASE → skip": {0, true, storage.SkipImageLayers_SKIP_BASE, false},
		"app / SKIP_BASE → keep":  {2, true, storage.SkipImageLayers_SKIP_BASE, true},
		"app / SKIP_APP → skip":   {2, true, storage.SkipImageLayers_SKIP_APP, false},
		"no baseinfo / SKIP_BASE → keep (treat as app)": {0, false, storage.SkipImageLayers_SKIP_BASE, true},
		"no baseinfo / SKIP_APP → skip (treat as app)":  {0, false, storage.SkipImageLayers_SKIP_APP, false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, keepLayerByIndex(tc.i, maxLayer, tc.hasMax, tc.skip))
		})
	}
}

func TestImageForPath(t *testing.T) {
	img0 := &storage.Image{Name: &storage.ImageName{FullName: "img0"}}
	img1 := &storage.Image{Name: &storage.ImageName{FullName: "img1"}}
	images := []*storage.Image{img0, img1}

	path := func(steps ...pathutil.Step) *pathutil.Path {
		return pathutil.NewPath(steps...)
	}

	t.Run("container 0", func(t *testing.T) {
		p := path(pathutil.FieldStep("Containers"), pathutil.IndexStep(0), pathutil.FieldStep("Image"))
		img, ok := imageForPath(images, p)
		assert.True(t, ok)
		assert.Same(t, img0, img)
	})
	t.Run("container 1", func(t *testing.T) {
		p := path(pathutil.FieldStep("Containers"), pathutil.IndexStep(1), pathutil.FieldStep("Image"))
		img, ok := imageForPath(images, p)
		assert.True(t, ok)
		assert.Same(t, img1, img)
	})
	t.Run("out-of-range index", func(t *testing.T) {
		p := path(pathutil.FieldStep("Containers"), pathutil.IndexStep(5), pathutil.FieldStep("Image"))
		_, ok := imageForPath(images, p)
		assert.False(t, ok)
	})
	t.Run("no Containers step, single image → fallback", func(t *testing.T) {
		p := path(pathutil.FieldStep("Scan"), pathutil.FieldStep("Components"))
		img, ok := imageForPath(images[:1], p)
		assert.True(t, ok)
		assert.Same(t, img0, img)
	})
	t.Run("no Containers step, multiple images → not found", func(t *testing.T) {
		p := path(pathutil.FieldStep("Scan"), pathutil.FieldStep("Components"))
		_, ok := imageForPath(images, p)
		assert.False(t, ok)
	})
}
