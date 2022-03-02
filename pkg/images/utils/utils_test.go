package utils

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	var cases = []struct {
		ImageString   string
		ExpectedImage *storage.ContainerImage
	}{
		{
			ImageString: "nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &storage.ContainerImage{
				Id: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Name: &storage.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					FullName: "docker.io/library/nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				},
				NotPullable: false,
			},
		},
		{
			ImageString: "stackrox.io/main:1.0@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &storage.ContainerImage{
				Id: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Name: &storage.ImageName{
					Registry: "stackrox.io",
					Remote:   "main",
					Tag:      "1.0",
					FullName: "stackrox.io/main:1.0@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				},
				NotPullable: false,
			},
		},
		{
			ImageString: "nginx",
			ExpectedImage: &storage.ContainerImage{
				Name: &storage.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					FullName: "docker.io/library/nginx:latest",
				},
				NotPullable: false,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.ImageString, func(t *testing.T) {
			img, err := GenerateImageFromString(c.ImageString)
			assert.NoError(t, err)
			assert.Equal(t, c.ExpectedImage, img)
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
			assert.Equal(t, c.output, ExtractImageDigest(c.input))
		})
	}
}

func TestGenerateImageFromStringWithOverride(t *testing.T) {
	cases := []struct {
		name         string
		image        string
		override     string
		expectedName *storage.ImageName
	}{
		{
			name:  "no remote - no override",
			image: "nginx:latest",
			expectedName: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "docker.io/library/nginx:latest",
			},
		},
		{
			name:  "no registry - no override",
			image: "library/nginx:latest",
			expectedName: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "docker.io/library/nginx:latest",
			},
		},
		{
			name:  "full registry - no override",
			image: "docker.io/library/nginx:latest",
			expectedName: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "docker.io/library/nginx:latest",
			},
		},
		{
			name:     "full registry - not docker - override",
			image:    "quay.io/library/nginx:latest",
			override: "override.io",
			expectedName: &storage.ImageName{
				Registry: "quay.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "quay.io/library/nginx:latest",
			},
		},
		{
			name:     "no remote - override",
			image:    "nginx:latest",
			override: "override.io",
			expectedName: &storage.ImageName{
				Registry: "override.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "override.io/library/nginx:latest",
			},
		},
		{
			name:     "no registry - override",
			image:    "library/nginx:latest",
			override: "override.io",
			expectedName: &storage.ImageName{
				Registry: "override.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "override.io/library/nginx:latest",
			},
		},
		{
			name:     "full registry - override",
			image:    "docker.io/library/nginx:latest",
			override: "override.io",
			expectedName: &storage.ImageName{
				Registry: "override.io",
				Remote:   "library/nginx",
				Tag:      "latest",
				FullName: "override.io/library/nginx:latest",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img, err := GenerateImageFromStringWithOverride(c.image, c.override)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedName, img.Name)
		})
	}
}

func TestStripCVEDescriptions(t *testing.T) {
	newImg := StripCVEDescriptions(fixtures.GetImage())
	var hitOne bool
	for _, comp := range newImg.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			hitOne = true
			assert.Empty(t, vuln.Summary)
		}
	}
	// Validate that we at least removed one summary
	assert.True(t, hitOne)
}

func TestDropImageTagAndDigest(t *testing.T) {
	cases := map[string]struct {
		image         string
		expectedImage string
		expectedError error
	}{
		"Image with Tag": {
			image:         "docker.io/stackrox/rox:tag",
			expectedImage: "docker.io/stackrox/rox",
			expectedError: nil,
		},
		"Image with Digest": {
			image:         "docker.io/stackrox/rox@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "docker.io/stackrox/rox",
			expectedError: nil,
		},
		"Image with Tag and Digest": {
			image:         "docker.io/stackrox/rox:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "docker.io/stackrox/rox",
			expectedError: nil,
		},
		"Image with no tag or digest": {
			image:         "docker.io/stackrox/rox",
			expectedImage: "docker.io/stackrox/rox",
			expectedError: nil,
		},
		"Image with no tag or digest and no domain": {
			image:         "stackrox/rox",
			expectedImage: "stackrox/rox",
			expectedError: nil,
		},
		"No registry with tag": {
			image:         "nginx:tag",
			expectedImage: "nginx",
			expectedError: nil,
		},
		"No registry with sha": {
			image:         "nginx@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "nginx",
			expectedError: nil,
		},
		"No registry with tag and sha": {
			image:         "nginx:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "nginx",
			expectedError: nil,
		},
		"No registry": {
			image:         "nginx",
			expectedImage: "nginx",
			expectedError: nil,
		},
		"Invalid image": {
			image:         "invalid image",
			expectedError: errors.New("invalid image name 'invalid image': invalid reference format"),
		},
		"docker.io and library with tag and sha": {
			image:         "docker.io/library/nginx:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "docker.io/library/nginx",
			expectedError: nil,
		},
		"no domain, library with tag and sha": {
			image:         "library/nginx:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "library/nginx",
			expectedError: nil,
		},
		"stackrox.io domain with tag and sha": {
			image:         "stackrox.io/path/main:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "stackrox.io/path/main",
			expectedError: nil,
		},
		"quay.io domain with tag and sha": {
			image:         "quay.io/path/main:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "quay.io/path/main",
			expectedError: nil,
		},
		"docker.io domain, no repository with tag and sha": {
			image:         "docker.io/nginx:tag@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993",
			expectedImage: "docker.io/nginx",
			expectedError: nil,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resImage, gotError := DropImageTagAndDigest(c.image)

			if c.expectedError == nil {
				assert.NoError(t, gotError, "expected a valid image but got error")
				assert.Equal(t, c.expectedImage, resImage, "Expected image %s but got %s", c.expectedImage, resImage)
			} else {
				assert.Error(t, gotError)
				assert.Equal(t, c.expectedError.Error(), gotError.Error())
			}
		})
	}
}
