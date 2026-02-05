package matcher

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMatchWithBaseImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDS := mocks.NewMockDataStore(ctrl)
	m := New(mockDS)
	ctx := context.Background()

	testCreatedTime := protoconv.ConvertTimeToTimestamp(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

	tcs := []struct {
		desc      string
		imgLayers []string
		mockSetup func()
		expected  []*storage.BaseImageInfo
	}{
		{
			desc:      "Match found: Base image layers are returned out of order",
			imgLayers: []string{"layer-A", "layer-B", "layer-C"},
			mockSetup: func() {
				mockDS.EXPECT().
					ListCandidateBaseImages(gomock.Any(), "layer-A").
					Return([]*storage.BaseImage{
						{
							Id:             "base-1",
							Repository:     "rhel",
							Tag:            "8",
							ManifestDigest: "sha-base",
							Created:        testCreatedTime,
							Layers: []*storage.BaseImageLayer{
								{LayerDigest: "layer-B", Index: 1}, // Index 1 comes first in slice
								{LayerDigest: "layer-A", Index: 0}, // Index 0 comes second
							},
						},
					}, nil)
			},
			expected: []*storage.BaseImageInfo{
				{
					BaseImageId:       "base-1",
					BaseImageFullName: "rhel:8",
					BaseImageDigest:   "sha-base",
					Created:           testCreatedTime,
				},
			},
		},
		{
			desc:      "No match: Layers match content but not order (index check)",
			imgLayers: []string{"layer-A", "layer-B", "layer-C"},
			mockSetup: func() {
				mockDS.EXPECT().
					ListCandidateBaseImages(gomock.Any(), "layer-A").
					Return([]*storage.BaseImage{
						{
							Id: "wrong-order-base",
							Layers: []*storage.BaseImageLayer{
								{LayerDigest: "layer-A", Index: 1}, // A is at Index 1
								{LayerDigest: "layer-B", Index: 0}, // B is at Index 0
							},
						},
					}, nil)
			},
			expected: nil, // Should fail because imgLayers[0] (A) != sorted layers[0] (B)
		},
		{
			desc:      "Multiple candidates: One matches, one does not",
			imgLayers: []string{"L1", "L2", "L3"},
			mockSetup: func() {
				mockDS.EXPECT().
					ListCandidateBaseImages(gomock.Any(), "L1").
					Return([]*storage.BaseImage{
						{
							Id: "match",
							Layers: []*storage.BaseImageLayer{
								{LayerDigest: "L1", Index: 0},
								{LayerDigest: "L2", Index: 1},
							},
						},
						{
							Id: "no-match-different-content",
							Layers: []*storage.BaseImageLayer{
								{LayerDigest: "L1", Index: 0},
								{LayerDigest: "LX", Index: 1},
							},
						},
					}, nil)
			},
			expected: []*storage.BaseImageInfo{
				{BaseImageId: "match"},
			},
		},
		{
			desc:      "Self-match prevention: Exact layer count and content",
			imgLayers: []string{"L1", "L2"},
			mockSetup: func() {
				mockDS.EXPECT().
					ListCandidateBaseImages(gomock.Any(), "L1").
					Return([]*storage.BaseImage{
						{
							Id: "identical",
							Layers: []*storage.BaseImageLayer{
								{LayerDigest: "L1", Index: 0},
								{LayerDigest: "L2", Index: 1},
							},
						},
					}, nil)
			},
			expected: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			tc.mockSetup()

			actual, _ := m.MatchWithBaseImages(ctx, tc.imgLayers)

			if tc.expected == nil {
				assert.Empty(t, actual)
			} else {
				require.Equal(t, len(tc.expected), len(actual))
				for i := range tc.expected {
					assert.Equal(t, tc.expected[i].GetBaseImageId(), actual[i].GetBaseImageId())
					if tc.expected[i].GetBaseImageFullName() != "" {
						assert.Equal(t, tc.expected[i].GetBaseImageFullName(), actual[i].GetBaseImageFullName())
					}
					if tc.expected[i].GetCreated() != nil {
						assert.Equal(t, tc.expected[i].GetCreated(), actual[i].GetCreated())
					}
				}
			}
		})
	}
}
