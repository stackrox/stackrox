package utils

import (
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
			input:  "docker-pullable://registry.k8s.io/etcd-amd64@sha256:68235934469f3bc58917bcf7018bf0d3b72129e6303b0bef28186d96b2259317",
			output: "sha256:68235934469f3bc58917bcf7018bf0d3b72129e6303b0bef28186d96b2259317",
		},
		{
			input:  "docker://sha256:041b6144416e6e9c540d1fb4883ebc1b6fe4baf09d066d8311c0109755baae96",
			output: "sha256:041b6144416e6e9c540d1fb4883ebc1b6fe4baf09d066d8311c0109755baae96",
		},
		{
			input:  "docker-pullable://registry.k8s.io/etcd-amd64@sha512:4cc8f2b59644e88f744c5d889a9082b9c3e6c03c549c703d1ec5613ecb308beae9b0d0c268ef6c5efdc1606d0e918a211276c3ae5d5fa7c7e903b6f2237f2383",
			output: "sha512:4cc8f2b59644e88f744c5d889a9082b9c3e6c03c549c703d1ec5613ecb308beae9b0d0c268ef6c5efdc1606d0e918a211276c3ae5d5fa7c7e903b6f2237f2383",
		},
		{
			input:  "docker://sha512:36fb26cde46557cf26a79d8fe53e704416c18afe667103fe58d84180d8a3e33244cd10baabeaeb0eb7541760ab776e3db2dee5e15a9ad26b0966703889c4eb45",
			output: "sha512:36fb26cde46557cf26a79d8fe53e704416c18afe667103fe58d84180d8a3e33244cd10baabeaeb0eb7541760ab776e3db2dee5e15a9ad26b0966703889c4eb45",
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

func TestExtractOpenShiftProject_fullName(t *testing.T) {
	imgName := &storage.ImageName{
		Registry: "image-registry.openshift-image-registry.svc:5000",
		Remote:   "qa/nginx",
		Tag:      "1.18.0",
		FullName: "image-registry.openshift-image-registry.svc:5000/qa/nginx:1.18.0",
	}
	assert.Equal(t, "qa", ExtractOpenShiftProject(imgName))
}

func TestExtractOpenShiftProject_solelyRemote(t *testing.T) {
	imgName := &storage.ImageName{
		Remote: "stackrox/nginx",
	}
	assert.Equal(t, "stackrox", ExtractOpenShiftProject(imgName))
}

func TestUniqueImageNames(t *testing.T) {
	a := []*storage.ImageName{
		{
			Registry: "docker.io",
			Remote:   "something",
			Tag:      "latest",
			FullName: "docker.io/something:latest",
		},
		{
			Registry: "docker.io",
			Remote:   "someone",
			Tag:      "latest",
			FullName: "docker.io/someone:latest",
		},
	}
	b := []*storage.ImageName{
		{
			Registry: "docker.io",
			Remote:   "somewhere",
			Tag:      "latest",
			FullName: "docker.io/somewhere:latest",
		},
		{
			Registry: "docker.io",
			Remote:   "someone",
			Tag:      "latest",
			FullName: "docker.io/someone:latest",
		},
	}

	res := UniqueImageNames(a, b)

	assert.Equal(t, append(a, b[0]), res)
}
