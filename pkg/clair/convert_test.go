package clair

import (
	"testing"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertVulnerability(t *testing.T) {
	clairVulns, protoVulns := mock.GetTestVulns()
	for i, vuln := range clairVulns {
		assert.Equal(t, protoVulns[i], ConvertVulnerability(vuln))
	}
}

func TestConvertFeatures(t *testing.T) {
	clairFeatures, protoComponents := mock.GetTestFeatures()
	assert.Equal(t, protoComponents, ConvertFeatures(clairFeatures))
}

func TestPopulateLayersWithScan(t *testing.T) {

	var cases = []struct {
		name           string
		metadata       *storage.ImageMetadata
		envelope       *clairV1.LayerEnvelope
		expectedLayers []*storage.ImageLayer
	}{
		{
			name: "Nil metadata",
		},
		{
			name:     "Empty metadata",
			metadata: &storage.ImageMetadata{},
		},
		{
			name: "v1 metadata with equal vulns and layers - no empty",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{{}, {}},
				},
				LayerShas: []string{"A", "B"},
			},
			envelope: &clairV1.LayerEnvelope{
				Layer: &clairV1.Layer{
					Features: []clairV1.Feature{
						{
							Name:    "a-name",
							AddedBy: "A",
						},
						{
							Name:    "b-name",
							AddedBy: "B",
						},
					},
				},
			},
			expectedLayers: []*storage.ImageLayer{
				{
					Components: []*storage.ImageScanComponent{{Name: "a-name"}},
				},
				{
					Components: []*storage.ImageScanComponent{{Name: "b-name"}},
				},
			},
		},
		{
			name: "v1 metadata with fewer vulns than layers - no empty",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{{}, {}},
				},
				LayerShas: []string{"A", "B"},
			},
			envelope: &clairV1.LayerEnvelope{
				Layer: &clairV1.Layer{
					Features: []clairV1.Feature{
						{
							Name:    "b-name",
							AddedBy: "B",
						},
					},
				},
			},
			expectedLayers: []*storage.ImageLayer{
				{
					Components: nil,
				},
				{
					Components: []*storage.ImageScanComponent{{Name: "b-name"}},
				},
			},
		},
		{
			name: "v2 metadata with fewer vulns than layers - no empty",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{{}, {}},
				},
				V2:        &storage.V2Metadata{},
				LayerShas: []string{"A", "B"},
			},
			envelope: &clairV1.LayerEnvelope{
				Layer: &clairV1.Layer{
					Features: []clairV1.Feature{
						{
							Name:    "b-name",
							AddedBy: "B",
						},
					},
				},
			},
			expectedLayers: []*storage.ImageLayer{
				{
					Components: nil,
				},
				{
					Components: []*storage.ImageScanComponent{{Name: "b-name"}},
				},
			},
		},
		{
			name: "v2 metadata with empty layers",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{{Empty: true}, {}, {}},
				},
				V2:        &storage.V2Metadata{},
				LayerShas: []string{"A", "B"},
			},
			envelope: &clairV1.LayerEnvelope{
				Layer: &clairV1.Layer{
					Features: []clairV1.Feature{
						{
							Name:    "b-name",
							AddedBy: "B",
						},
					},
				},
			},
			expectedLayers: []*storage.ImageLayer{
				{},
				{},
				{
					Components: []*storage.ImageScanComponent{{Name: "b-name"}},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img := &storage.Image{Metadata: c.metadata}
			PopulateLayersWithScan(img, c.envelope)

			require.Len(t, img.Metadata.GetV1().GetLayers(), len(c.expectedLayers))
			for i, el := range c.expectedLayers {
				v1Components := img.GetMetadata().GetV1().GetLayers()[i].Components
				for j, comp := range el.Components {
					assert.Equal(t, comp.GetName(), v1Components[j].GetName())
				}
			}
		})
	}

}
