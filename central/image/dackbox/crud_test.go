package dackbox

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConvertImagesToListImages(t *testing.T) {
	ts := timestamp.TimestampNow()
	var cases = []struct {
		input    *storage.Image
		expected *storage.ListImage
	}{
		{
			input: &storage.Image{
				Id: "sha",
				Name: &storage.ImageName{
					FullName: "name",
				},
			},
			expected: &storage.ListImage{
				Id:   "sha",
				Name: "name",
			},
		},
		{
			input: &storage.Image{
				Id: "sha",
				Name: &storage.ImageName{
					FullName: "name",
				},
				Metadata: &storage.ImageMetadata{
					V1: &storage.V1Metadata{
						Created: ts,
					},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{},
								{
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "hi",
									},
								},
							},
						},
					},
				},
			},
			expected: &storage.ListImage{
				Id:      "sha",
				Name:    "name",
				Created: ts,
				SetComponents: &storage.ListImage_Components{
					Components: 2,
				},
				SetCves: &storage.ListImage_Cves{
					Cves: 3,
				},
				SetFixable: &storage.ListImage_FixableCves{
					FixableCves: 1,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.input.GetName().GetFullName(), func(t *testing.T) {
			assert.Equal(t, c.expected, convertImageToListImage(c.input))
		})
	}
}
