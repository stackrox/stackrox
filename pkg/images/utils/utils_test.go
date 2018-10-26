package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	var cases = []struct {
		ImageString   string
		ExpectedImage *v1.Image
	}{
		{
			ImageString: "nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &v1.Image{
				Id: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Name: &v1.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					FullName: "docker.io/library/nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				},
			},
		},
		{
			ImageString: "stackrox.io/main:1.0@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &v1.Image{
				Id: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Name: &v1.ImageName{
					Registry: "stackrox.io",
					Remote:   "main",
					Tag:      "1.0",
					FullName: "stackrox.io/main:1.0@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				},
			},
		},
		{
			ImageString: "nginx",
			ExpectedImage: &v1.Image{
				Name: &v1.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					FullName: "docker.io/library/nginx:latest",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.ImageString, func(t *testing.T) {
			assert.Equal(t, c.ExpectedImage, GenerateImageFromString(c.ImageString))
		})
	}
}

func TestExtractImageSha(t *testing.T) {
	var cases = []struct {
		input  string
		output string
	}{
		{
			input:  "docker-pullable://k8s.gcr.io/etcd-amd64@sha256:68235934469f3bc58917bcf7018bf0d3b72129e6303b0bef28186d96b2259317",
			output: "sha256:68235934469f3bc58917bcf7018bf0d3b72129e6303b0bef28186d96b2259317",
		},
		{
			input:  "docker://sha256:041b6144416e6e9c540d1fb4883ebc1b6fe4baf09d066d8311c0109755baae96",
			output: "sha256:041b6144416e6e9c540d1fb4883ebc1b6fe4baf09d066d8311c0109755baae96",
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.output, ExtractImageSha(c.input))
		})
	}
}
