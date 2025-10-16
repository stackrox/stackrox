package clair

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertVulnerability(t *testing.T) {
	clairVulns, protoVulns := mock.GetTestVulns()
	for i, vuln := range clairVulns {
		protoassert.Equal(t, protoVulns[i], ConvertVulnerability(vuln))
	}
}

func TestConvertFeatures(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	clairFeatures, protoComponents := mock.GetTestFeatures()
	protoassert.SlicesEqual(t, protoComponents, ConvertFeatures(nil, clairFeatures, ""))
}

func TestVersionFormatCompleteness(t *testing.T) {
	assert.Equal(t, len(VersionFormatsToSource), int(component.SentinelEndSourceType-component.UnsetSourceType-1))
}

func componentWithLayerIndex(name string, idx int32) *storage.EmbeddedImageScanComponent {
	c := &storage.EmbeddedImageScanComponent{}
	c.SetName(name)

	c.SetVulns([]*storage.EmbeddedVulnerability{})
	c.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	if idx != -1 {
		c.SetLayerIndex(idx)
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
			metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Layers: []*storage.ImageLayer{{}, {}},
				}.Build(),
				LayerShas: []string{"A", "B"},
			}.Build(),
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
			metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Layers: []*storage.ImageLayer{{}, {}},
				}.Build(),
				LayerShas: []string{"A", "B"},
			}.Build(),
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
			metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Layers: []*storage.ImageLayer{{}, {}},
				}.Build(),
				V2:        &storage.V2Metadata{},
				LayerShas: []string{"A", "B"},
			}.Build(),
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
			metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Layers: []*storage.ImageLayer{storage.ImageLayer_builder{Empty: true}.Build(), {}, {}},
				}.Build(),
				V2:        &storage.V2Metadata{},
				LayerShas: []string{"A", "B"},
			}.Build(),
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
			img := &storage.Image{}
			img.SetMetadata(c.metadata)
			convertedComponents := ConvertFeatures(img, c.features, "")
			require.Equal(t, len(c.expectedComponents), len(convertedComponents))
			for i := range convertedComponents {
				protoassert.Equal(t, c.expectedComponents[i], convertedComponents[i])
			}
		})
	}
}
