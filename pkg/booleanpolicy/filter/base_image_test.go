package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestImageLayerFilter(t *testing.T) {
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
			f := newImageLayerFilter(tc.skip)
			assert.NotNil(t, f)
			assert.True(t, f.IsNonDefault())

			dep := &storage.Deployment{}
			_, resultImgs := f.Apply(dep, []*storage.Image{tc.image})

			var names []string
			for _, c := range resultImgs[0].GetScan().GetComponents() {
				names = append(names, c.GetName())
			}
			assert.Equal(t, tc.expectedComponents, names)
		})
	}
}

func TestImageLayerFilter_NoBaseImageInfo(t *testing.T) {
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

	f := newImageLayerFilter(storage.SkipImageLayers_SKIP_BASE)
	assert.NotNil(t, f)

	dep := &storage.Deployment{}
	_, resultImgs := f.Apply(dep, []*storage.Image{img})

	assert.Len(t, resultImgs[0].GetScan().GetComponents(), 1,
		"without BaseImageInfo, image should be returned unmodified")
}

func TestImageLayerFilter_SkipNone(t *testing.T) {
	f := newImageLayerFilter(storage.SkipImageLayers_SKIP_NONE)
	assert.Nil(t, f)
}

func TestImageLayerFilter_DockerfileLayers(t *testing.T) {
	img := &storage.Image{
		Id: "test-img",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{},
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{Instruction: "FROM", Value: "ubuntu:22.04"},
					{Instruction: "RUN", Value: "apt-get update"},
					{Instruction: "RUN", Value: "apt-get install -y curl"},
					{Instruction: "COPY", Value: "app.py /app/"},
					{Instruction: "RUN", Value: "pip install flask"},
				},
			},
		},
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 2},
		},
	}

	cases := map[string]struct {
		skip               storage.SkipImageLayers
		expectedLayerCount int
		expectedFirstInstr string
	}{
		"SKIP_BASE keeps only app layers (index > 2)": {
			skip:               storage.SkipImageLayers_SKIP_BASE,
			expectedLayerCount: 2,
			expectedFirstInstr: "COPY",
		},
		"SKIP_APP keeps only base layers (index <= 2)": {
			skip:               storage.SkipImageLayers_SKIP_APP,
			expectedLayerCount: 3,
			expectedFirstInstr: "FROM",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newImageLayerFilter(tc.skip)
			assert.NotNil(t, f)

			dep := &storage.Deployment{}
			_, resultImgs := f.Apply(dep, []*storage.Image{img})

			layers := resultImgs[0].GetMetadata().GetV1().GetLayers()
			assert.Len(t, layers, tc.expectedLayerCount)
			if tc.expectedLayerCount > 0 {
				assert.Equal(t, tc.expectedFirstInstr, layers[0].GetInstruction())
			}
		})
	}
}

func TestImageLayerFilter_NilImage(t *testing.T) {
	f := newImageLayerFilter(storage.SkipImageLayers_SKIP_BASE)
	assert.NotNil(t, f)

	dep := &storage.Deployment{}
	_, resultImgs := f.Apply(dep, []*storage.Image{nil})
	assert.Nil(t, resultImgs[0])
}
