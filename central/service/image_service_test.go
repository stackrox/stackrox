package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertImagesToListImages(t *testing.T) {
	ts := timestamp.TimestampNow()
	var cases = []struct {
		input    *v1.Image
		expected *v1.ListImage
	}{
		{
			input: &v1.Image{
				Name: &v1.ImageName{
					Sha:      "sha",
					FullName: "name",
				},
			},
			expected: &v1.ListImage{
				Sha:  "sha",
				Name: "name",
			},
		},
		{
			input: &v1.Image{
				Name: &v1.ImageName{
					Sha:      "sha",
					FullName: "name",
				},
				Metadata: &v1.ImageMetadata{
					Created: ts,
				},
				Scan: &v1.ImageScan{
					Components: []*v1.ImageScanComponent{
						{
							Vulns: []*v1.Vulnerability{
								{},
							},
						},
						{
							Vulns: []*v1.Vulnerability{
								{},
								{},
							},
						},
					},
				},
			},
			expected: &v1.ListImage{
				Sha:     "sha",
				Name:    "name",
				Created: ts,
				SetComponents: &v1.ListImage_Components{
					Components: 2,
				},
				SetCves: &v1.ListImage_Cves{
					Cves: 3,
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
