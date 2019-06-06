package m9to10

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestImageStitch(t *testing.T) {
	cases := []struct {
		image, expectedImage *storage.Image
		changed              bool
	}{
		{
			image:         &storage.Image{},
			expectedImage: &storage.Image{},
		},
		{
			image: &storage.Image{
				Scan: &storage.ImageScan{},
			},
			expectedImage: &storage.Image{
				Scan: &storage.ImageScan{},
			},
		},
		{
			image: &storage.Image{
				Metadata: &storage.ImageMetadata{},
			},
			expectedImage: &storage.Image{
				Metadata: &storage.ImageMetadata{},
			},
		},
		{
			changed: true,
			image: &storage.Image{
				Metadata: &storage.ImageMetadata{
					V1: &storage.V1Metadata{
						Layers: []*storage.ImageLayer{
							{
								DEPRECATEDComponents: []*storage.ImageScanComponent{
									{
										Name: "name1",
									},
									{
										Name: "name2",
									},
								},
							},
							{
								DEPRECATEDComponents: []*storage.ImageScanComponent{
									{
										Name: "name3",
									},
								},
							},
						},
					},
				},
				Scan: &storage.ImageScan{},
			},
			expectedImage: &storage.Image{
				Metadata: &storage.ImageMetadata{
					V1: &storage.V1Metadata{
						Layers: []*storage.ImageLayer{
							{},
							{},
						},
					},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.ImageScanComponent{
						{
							Name: "name1",
							HasLayerIndex: &storage.ImageScanComponent_LayerIndex{
								LayerIndex: 0,
							},
						},
						{
							Name: "name2",
							HasLayerIndex: &storage.ImageScanComponent_LayerIndex{
								LayerIndex: 0,
							},
						},
						{
							Name: "name3",
							HasLayerIndex: &storage.ImageScanComponent_LayerIndex{
								LayerIndex: 1,
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.changed, stitchImageComponents(c.image))
		assert.Equal(t, c.expectedImage, c.image)
	}
}
