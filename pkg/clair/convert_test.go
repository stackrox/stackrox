package clair

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
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
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	clairFeatures, protoComponents := mock.GetTestFeatures()
	assert.Equal(t, protoComponents, ConvertFeatures(nil, clairFeatures, ""))
}

func TestVersionFormatCompleteness(t *testing.T) {
	assert.Equal(t, len(VersionFormatsToSource), int(component.SentinelEndSourceType-component.UnsetSourceType-1))
}

func componentWithLayerIndex(name string, idx int32) *storage.EmbeddedImageScanComponent {
	c := &storage.EmbeddedImageScanComponent{
		Name: name,

		Vulns:       []*storage.EmbeddedVulnerability{},
		Executables: []*storage.EmbeddedImageScanComponent_Executable{},
	}
	if idx != -1 {
		c.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
			LayerIndex: idx,
		}
	}
	return c
}

func TestConvertFeaturesWithLayerIndexes(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	var cases = []struct {
		name               string
		metadata           *storage.ImageMetadata
		features           []clairV1.Feature
		expectedComponents []*storage.EmbeddedImageScanComponent
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
			features: []clairV1.Feature{
				{
					Name:    "a-name",
					AddedBy: "A",
				},
				{
					Name:    "b-name",
					AddedBy: "B",
				},
			},
			expectedComponents: []*storage.EmbeddedImageScanComponent{
				componentWithLayerIndex("a-name", 0),
				componentWithLayerIndex("b-name", 1),
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
			features: []clairV1.Feature{
				{
					Name:    "b-name",
					AddedBy: "B",
				},
			},
			expectedComponents: []*storage.EmbeddedImageScanComponent{
				componentWithLayerIndex("b-name", 1),
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
			features: []clairV1.Feature{
				{
					Name:    "b-name",
					AddedBy: "B",
				},
			},
			expectedComponents: []*storage.EmbeddedImageScanComponent{
				componentWithLayerIndex("b-name", 1),
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
			features: []clairV1.Feature{
				{
					Name:    "b-name",
					AddedBy: "B",
				},
			},
			expectedComponents: []*storage.EmbeddedImageScanComponent{
				componentWithLayerIndex("b-name", 2),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img := &storage.Image{
				Metadata: c.metadata,
			}
			convertedComponents := ConvertFeatures(img, c.features, "")
			require.Equal(t, len(c.expectedComponents), len(convertedComponents))
			for i := range convertedComponents {
				assert.Equal(t, c.expectedComponents[i], convertedComponents[i])
			}
		})
	}
}

func TestConvertTime(t *testing.T) {
	cases := []struct {
		input  string
		output *types.Timestamp
	}{
		{
			input:  "",
			output: nil,
		},
		{
			input:  "malformed",
			output: nil,
		},
		{
			input: "2018-02-07T23:29Z",
			output: &types.Timestamp{
				Seconds: 1518046140,
			},
		},
		{
			input: "2019-01-20T00:00:00Z",
			output: &types.Timestamp{
				Seconds: 1547942400,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.output, ConvertTime(c.input))
		})
	}
}
