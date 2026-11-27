package reportgenerator

import (
	"context"
	"strconv"
	"testing"

	baseImageRepoMocks "github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestFormatBaseImageNames(t *testing.T) {
	testCases := map[string]struct {
		infos      []*storage.BaseImageInfo
		configured map[string][]string
		expected   string
	}{
		"no base image": {},
		"single match": {
			infos:    []*storage.BaseImageInfo{{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-a/base:9"}},
			expected: "registry-a/base:9",
		},
		"same digest uses one deterministic name": {
			infos: []*storage.BaseImageInfo{
				{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-b/base:stable"},
				{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-a/base:9"},
				{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-a/base:9"},
			},
			expected: "registry-a/base:9",
		},
		"configured reference preferred over alphabetical first": {
			infos: []*storage.BaseImageInfo{
				{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-a/base:9"},
				{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-b:5000/base:stable"},
			},
			configured: map[string][]string{"registry-b:5000/base": {"stable"}},
			expected:   "registry-b:5000/base:stable",
		},
		"distinct bases joined in one cell": {
			infos: []*storage.BaseImageInfo{
				{BaseImageDigest: "sha256:second", BaseImageFullName: "registry-b/runtime:2"},
				{BaseImageDigest: "sha256:first", BaseImageFullName: "registry-a/base:9"},
			},
			expected: "registry-a/base:9; registry-b/runtime:2",
		},
		"unresolved base stays blank": {
			infos: []*storage.BaseImageInfo{
				{BaseImageFullName: "registry-a/base:9"},
				{BaseImageDigest: "sha256:base"},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatBaseImageNames(tc.infos, tc.configured))
		})
	}
}

func TestWithBaseImageNames(t *testing.T) {
	for name, flatten := range map[string]bool{"legacy image": false, "flattened image": true} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.FlattenImageData.EnvVar(), strconv.FormatBool(flatten))
			ctrl := gomock.NewController(t)
			imageStore := imageDSMocks.NewMockDataStore(ctrl)
			repoStore := baseImageRepoMocks.NewMockDataStore(ctrl)
			rg := &reportGeneratorImpl{imageDatastore: imageStore, baseImageRepositoryDatastore: repoStore}

			imageName := "registry.example.com/app:1"
			digest := "sha256:app"
			id := digest
			if flatten {
				id = imageUtils.NewImageV2ID(&storage.ImageName{FullName: imageName}, digest)
			}
			image := &storage.Image{
				Id:   digest,
				Name: &storage.ImageName{FullName: imageName},
				BaseImageInfo: []*storage.BaseImageInfo{
					{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-a/base:9"},
					{BaseImageDigest: "sha256:base", BaseImageFullName: "registry-b/base:stable"},
				},
			}
			imageStore.EXPECT().GetManyImageMetadata(gomock.Any(), []string{id}).Return([]*storage.Image{image}, nil).Times(1)
			repoStore.EXPECT().ListRepositories(gomock.Any()).Return([]*storage.BaseImageRepository{
				{RepositoryPath: "registry-b/base", TagPattern: "stable"},
			}, nil).Times(1)

			rows := []*ImageCVEQueryResponse{
				{Image: &imageName, ImageSHA: &digest},
				{Image: &imageName, ImageSHA: &digest},
			}
			cache := &baseImageNameCache{}
			require.NoError(t, rg.withBaseImageNames(context.Background(), rows, cache))
			for _, row := range rows {
				assert.Equal(t, "registry-b/base:stable", row.BaseImage)
			}
			require.NoError(t, rg.withBaseImageNames(context.Background(), rows, cache))
		})
	}
}
