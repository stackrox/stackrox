package test

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
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
			img, err := utils.GenerateImageFromString(c.ImageString)
			assert.NoError(t, err)
			protoassert.Equal(t, c.ExpectedImage, img)
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
			assert.Equal(t, c.output, utils.ExtractImageDigest(c.input))
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
			img, err := utils.GenerateImageFromStringWithOverride(c.image, c.override)
			assert.NoError(t, err)
			protoassert.Equal(t, c.expectedName, img.GetName())
		})
	}
}

func TestStripCVEDescriptions(t *testing.T) {
	newImg := utils.StripCVEDescriptions(fixtures.GetImage())
	var hitOne bool
	for _, comp := range newImg.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			hitOne = true
			assert.Empty(t, vuln.GetSummary())
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
	assert.Equal(t, "qa", utils.ExtractOpenShiftProject(imgName))
}

func TestExtractOpenShiftProject_solelyRemote(t *testing.T) {
	imgName := &storage.ImageName{
		Remote: "stackrox/nginx",
	}
	assert.Equal(t, "stackrox", utils.ExtractOpenShiftProject(imgName))
}

func TestRemoveScheme(t *testing.T) {
	tcs := []struct {
		imageStr string
		want     string
	}{
		{"", ""},
		{"nginx:latest", "nginx:latest"},
		{"docker-pullable://rest-of-image", "rest-of-image"},
		{
			"crio://image-registry.openshift-image-registry.svc:5000/testdev/nginx:1.18.0@sha256:e90ac5331fe095cea01b121a3627174b2e33e06e83720e9a934c7b8ccc9c55a0",
			"image-registry.openshift-image-registry.svc:5000/testdev/nginx:1.18.0@sha256:e90ac5331fe095cea01b121a3627174b2e33e06e83720e9a934c7b8ccc9c55a0",
		},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			assert.Equal(t, tc.want, utils.RemoveScheme(tc.imageStr))
		})
	}
}

func TestNormalizeImageFullName(t *testing.T) {
	img, _ := utils.GenerateImageFromString("nginx@sha256:0000000000000000000000000000000000000000000000000000000000000000")
	fmt.Printf("\n%+v\n\n", img)
	tcs := []struct {
		name    string
		imgName *storage.ImageName
		digest  string
		want    string
	}{
		{
			"only tag",
			&storage.ImageName{Registry: "docker.io", Remote: "library/nginx", Tag: "latest"},
			"",
			"docker.io/library/nginx:latest",
		},
		{
			"only digest",
			&storage.ImageName{Registry: "docker.io", Remote: "library/nginx", Tag: ""},
			"sha256:0000000000000000000000000000000000000000000000000000000000000000",
			"docker.io/library/nginx@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"tag and digest (latest tag)",
			&storage.ImageName{Registry: "docker.io", Remote: "library/nginx", Tag: "latest"},
			"sha256:0000000000000000000000000000000000000000000000000000000000000000",
			"docker.io/library/nginx:latest@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"tag and digest (specific tag)",
			&storage.ImageName{Registry: "docker.io", Remote: "library/nginx", Tag: "v1.2.3"},
			"sha256:0000000000000000000000000000000000000000000000000000000000000000",
			"docker.io/library/nginx:v1.2.3@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"no tag or digest (malformed) do not modify fullname",
			&storage.ImageName{Registry: "docker.io", Remote: "library/nginx", Tag: "", FullName: "helloworld"},
			"",
			"helloworld",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.NormalizeImageFullName(tc.imgName, tc.digest)
			assert.Equal(t, tc.want, got.GetFullName())
		})
	}
}

func TestIsRedHatImageName(t *testing.T) {
	tcs := []struct {
		name     string
		imageStr string
		want     bool
	}{
		{
			name:     "images in registry.redhat.io are identified as Red Hat images",
			imageStr: "registry.redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
			want:     true,
		},
		{
			name:     "images in registry.access.redhat.com are identified as Red Hat images",
			imageStr: "registry.access.redhat.com/ubi8/openjdk-21@sha256:441897a1f691c7d4b3a67bb3e0fea83e18352214264cb383fd057bbbd5ed863c",
			want:     true,
		},
		{
			name:     "images in registry.connect.redhat.com are not identified as Red Hat images",
			imageStr: "registry.connect.redhat.com/nvidia-network-operator/nvidia-network-operator@sha256:2418015d00846dd0d7a8aca11927f1e89b4d8d525e6ae936360e3e3b3bd9e22f",
			want:     false,
		},
		{
			name:     "images in registry.marketplace.redhat.com are not identified as Red Hat images",
			imageStr: "registry.marketplace.redhat.com/rhm/seldonio/alibi-detect-server@sha256:4b0edf72477f54bdcb850457582f12bcb1338ca64dc94ebca056897402708306",
			want:     false,
		},
		{
			name:     "images in quay.io remote openshift-release-dev/ocp-v4.0-art-dev are identified as Red Hat images",
			imageStr: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c896b5d4b05343dfe94c0f75c9232a2a68044d0fa7a21b5f51ed796d23f1fcc5",
			want:     true,
		},
		{
			name:     "images in quay.io remote openshift-release-dev/ocp-release are identified as Red Hat images",
			imageStr: "quay.io/openshift-release-dev/ocp-release@sha256:3482dbdce3a6fb2239684d217bba6fc87453eff3bdb72f5237be4beb22a2160b",
			want:     true,
		},
		{
			name:     "images in third party registries are not identified as Red Hat images",
			imageStr: "docker.io/library/nginx:latest",
			want:     false,
		},
		{
			name:     "images in non-redhat quay.io remotes are not identified as Red Hat images",
			imageStr: "quay.io/kuadrant/kuadrant-operator:v0.3.1",
			want:     false,
		},
		{
			name:     "images in third party registries with quay.io Red Hat remote are not identified as Red Hat images",
			imageStr: "not-quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c896b5d4b05343dfe94c0f75c9232a2a68044d0fa7a21b5f51ed796d23f1fcc5",
			want:     false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			imgName, _, err := utils.GenerateImageNameFromString(tc.imageStr)
			assert.NoError(t, err)
			got := utils.IsRedHatImageName(imgName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsRedHatImage(t *testing.T) {
	tcs := []struct {
		name       string
		imageNames []string
		want       bool
	}{
		{
			name: "images with multiple names where any is Red Hat are identified as Red Hat images - first name is Red Hat",
			imageNames: []string{
				"registry.redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
				"not-redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
				"also-not-redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
			},
			want: true,
		},
		{
			name: "images with multiple names where any is Red Hat are identified as Red Hat images - last name is Red Hat",
			imageNames: []string{
				"not-redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
				"also-not-redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
				"registry.redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
			},
			want: true,
		},
		{
			name: "images with multiple names where none are Red Hat are not identified as Red Hat images",
			imageNames: []string{
				"docker.io/library/nginx:latest",
				"gcr.io/library/nginx:latest",
			},
			want: false,
		},
		{
			name: "returns true with single Red Hat name",
			imageNames: []string{
				"registry.redhat.io/openshift4/ose-csi-external-provisioner@sha256:395a5a4aa4cfe3a0093d2225ce2e67acdcec0fd894e4b61e30a750f22931448d",
			},
			want: true,
		},
		{
			name: "returns false with single non-Red Hat name",
			imageNames: []string{
				"docker.io/library/nginx:latest",
			},
			want: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var names []*storage.ImageName
			for _, nameStr := range tc.imageNames {
				name, _, err := utils.GenerateImageNameFromString(nameStr)
				assert.NoError(t, err)
				names = append(names, name)
			}

			img := &storage.Image{Names: names}
			got := utils.IsRedHatImage(img)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFillScanStatsV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cases := []struct {
		image                            *storage.ImageV2
		expectedCveCount                 int32
		expectedUnknownCveCount          int32
		expectedFixableUnknownCveCount   int32
		expectedCriticalCveCount         int32
		expectedFixableCriticalCveCount  int32
		expectedImportantCveCount        int32
		expectedFixableImportantCveCount int32
		expectedModerateCveCount         int32
		expectedFixableModerateCveCount  int32
		expectedLowCveCount              int32
		expectedFixableLowCveCount       int32
		expectedFixableCveCount          int32
	}{
		{
			image: &storage.ImageV2{
				Id:     utils.NewImageV2ID(&storage.ImageName{Registry: "reg", FullName: "reg"}, "sha"),
				Digest: "sha",
				Name:   &storage.ImageName{Registry: "reg", FullName: "reg"},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-1",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 1,
			expectedUnknownCveCount:          0,
			expectedFixableUnknownCveCount:   0,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  1,
			expectedImportantCveCount:        0,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         0,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              0,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          1,
		},
		{
			image: &storage.ImageV2{
				Id:     utils.NewImageV2ID(&storage.ImageName{Registry: "reg", FullName: "reg"}, "sha"),
				Digest: "sha",
				Name:   &storage.ImageName{Registry: "reg", FullName: "reg"},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
									Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 2,
			expectedUnknownCveCount:          1,
			expectedFixableUnknownCveCount:   1,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  1,
			expectedImportantCveCount:        0,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         0,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              0,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          2,
		},
		{
			image: &storage.ImageV2{
				Id:     utils.NewImageV2ID(&storage.ImageName{Registry: "reg", FullName: "reg"}, "sha"),
				Digest: "sha",
				Name:   &storage.ImageName{Registry: "reg", FullName: "reg"},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-1",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-2",
									Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-3",
									Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-4",
									Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-5",
									Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 5,
			expectedUnknownCveCount:          1,
			expectedFixableUnknownCveCount:   0,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  0,
			expectedImportantCveCount:        1,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         1,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              1,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          0,
		},
	}

	for _, c := range cases {
		t.Run(t.Name(), func(t *testing.T) {
			utils.FillScanStatsV2(c.image)
			assert.Equal(t, c.expectedCveCount, c.image.GetScanStats().GetCveCount())
			assert.Equal(t, c.expectedUnknownCveCount, c.image.GetScanStats().GetUnknownCveCount())
			assert.Equal(t, c.expectedFixableUnknownCveCount, c.image.GetScanStats().GetFixableUnknownCveCount())
			assert.Equal(t, c.expectedCriticalCveCount, c.image.GetScanStats().GetCriticalCveCount())
			assert.Equal(t, c.expectedFixableCriticalCveCount, c.image.GetScanStats().GetFixableCriticalCveCount())
			assert.Equal(t, c.expectedImportantCveCount, c.image.GetScanStats().GetImportantCveCount())
			assert.Equal(t, c.expectedFixableImportantCveCount, c.image.GetScanStats().GetFixableImportantCveCount())
			assert.Equal(t, c.expectedModerateCveCount, c.image.GetScanStats().GetModerateCveCount())
			assert.Equal(t, c.expectedFixableModerateCveCount, c.image.GetScanStats().GetFixableModerateCveCount())
			assert.Equal(t, c.expectedLowCveCount, c.image.GetScanStats().GetLowCveCount())
			assert.Equal(t, c.expectedFixableLowCveCount, c.image.GetScanStats().GetFixableLowCveCount())
			assert.Equal(t, c.expectedFixableCveCount, c.image.GetScanStats().GetFixableCveCount())
		})
	}
}
