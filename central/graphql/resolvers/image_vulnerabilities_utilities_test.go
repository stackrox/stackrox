package resolvers

import (
	"context"
	"testing"

	imageCVEV2DSMocks "github.com/stackrox/rox/central/cve/image/v2/datastore/mocks"
	cveFlatMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestImageCVEV2Resolver_PrefersFlatData(t *testing.T) {
	ctrl := gomock.NewController(t)

	critical := storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	important := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY

	flatData := cveFlatMocks.NewMockCveFlat(ctrl)
	flatData.EXPECT().GetSeverity().Return(&critical).AnyTimes()
	flatData.EXPECT().GetTopCVSS().Return(float32(9.8)).AnyTimes()
	flatData.EXPECT().GetTopNVDCVSS().Return(float32(9.1)).AnyTimes()

	zeroFlatData := cveFlatMocks.NewMockCveFlat(ctrl)
	zeroFlatData.EXPECT().GetSeverity().Return(&critical).AnyTimes()
	zeroFlatData.EXPECT().GetTopCVSS().Return(float32(0)).AnyTimes()
	zeroFlatData.EXPECT().GetTopNVDCVSS().Return(float32(0)).AnyTimes()

	cases := map[string]struct {
		resolver         *imageCVEV2Resolver
		expectedSeverity string
		expectedCvss     *float64
		expectedNvdCvss  *float64
	}{
		"with flatData, aggregated values are returned": {
			resolver: &imageCVEV2Resolver{
				data: &storage.ImageCVEV2{
					Severity: important,
					Cvss:     7.5,
					Nvdcvss:  6.9,
				},
				flatData: flatData,
			},
			expectedSeverity: critical.String(),
			expectedCvss:     new(9.8),
			expectedNvdCvss:  new(9.1),
		},
		"missing scores in stored data": {
			resolver:         &imageCVEV2Resolver{data: &storage.ImageCVEV2{Severity: important}},
			expectedSeverity: important.String(),
		},
		"missing flat scores take precedence over populated stored scores": {
			resolver: &imageCVEV2Resolver{
				data:     &storage.ImageCVEV2{Severity: important, Cvss: 7.5, Nvdcvss: 6.9},
				flatData: zeroFlatData,
			},
			expectedSeverity: critical.String(),
		},
		"only CVSS is available": {
			resolver:         &imageCVEV2Resolver{data: &storage.ImageCVEV2{Severity: important, Cvss: 7.5}},
			expectedSeverity: important.String(),
			expectedCvss:     new(7.5),
		},
		"only NVD CVSS is available": {
			resolver:         &imageCVEV2Resolver{data: &storage.ImageCVEV2{Severity: important, Nvdcvss: 6.9}},
			expectedSeverity: important.String(),
			expectedNvdCvss:  new(6.9),
		},
		"without flatData, denormalized values are returned": {
			resolver: &imageCVEV2Resolver{
				data: &storage.ImageCVEV2{
					Severity: important,
					Cvss:     7.5,
					Nvdcvss:  6.9,
				},
			},
			expectedSeverity: important.String(),
			expectedCvss:     new(7.5),
			expectedNvdCvss:  new(6.9),
		},
	}

	ctx := context.Background()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectedSeverity, tc.resolver.Severity(ctx))
			cvss := tc.resolver.Cvss(ctx)
			if tc.expectedCvss == nil {
				assert.Nil(t, cvss)
			} else {
				require.NotNil(t, cvss)
				assert.InDelta(t, *tc.expectedCvss, *cvss, 0.001)
			}
			nvdCVSS := tc.resolver.Nvdcvss(ctx)
			if tc.expectedNvdCvss == nil {
				assert.Nil(t, nvdCVSS)
			} else {
				require.NotNil(t, nvdCVSS)
				assert.InDelta(t, *tc.expectedNvdCvss, *nvdCVSS, 0.001)
			}
		})
	}
}

func TestImageCVEV2Resolver_IsFixable(t *testing.T) {
	groupedCVEIDs := []string{"cve-1#comp-1", "cve-1#comp-2"}
	fixableQuery := search.NewQueryBuilder().
		AddExactMatches(search.CVEID, groupedCVEIDs...).
		AddBools(search.Fixable, true).
		ProtoQuery()

	cases := map[string]struct {
		occurrenceFixable bool
		withFlatData      bool
		fixableCount      int
		expected          bool
	}{
		"without flatData, a fixable occurrence is fixable": {
			occurrenceFixable: true,
			expected:          true,
		},
		"without flatData, a non-fixable occurrence is not fixable": {
			occurrenceFixable: false,
			expected:          false,
		},
		"with flatData, fixable when any grouped occurrence is fixable": {
			occurrenceFixable: false,
			withFlatData:      true,
			fixableCount:      1,
			expected:          true,
		},
		"with flatData, not fixable when no grouped occurrence is fixable": {
			occurrenceFixable: true,
			withFlatData:      true,
			fixableCount:      0,
			expected:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// Without flatData, gomock fails the test on any datastore call.
			dataStore := imageCVEV2DSMocks.NewMockDataStore(ctrl)
			resolver := &imageCVEV2Resolver{
				root: &Resolver{ImageCVEV2DataStore: dataStore},
				data: &storage.ImageCVEV2{Id: "cve-1#comp-1", IsFixable: tc.occurrenceFixable},
			}
			if tc.withFlatData {
				flatData := cveFlatMocks.NewMockCveFlat(ctrl)
				flatData.EXPECT().GetCVEIDs().Return(groupedCVEIDs).AnyTimes()
				resolver.flatData = flatData
				dataStore.EXPECT().
					Count(gomock.Any(), protomock.GoMockMatcherEqualMessage(fixableQuery)).
					Return(tc.fixableCount, nil)
			}

			actual, err := resolver.IsFixable(context.Background(), RawQuery{})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestImageCVEV2Resolver_ComponentScopedLookups(t *testing.T) {
	componentScopeCtx := scoped.Context(context.Background(), scoped.Scope{
		IDs:   []string{"comp-1"},
		Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
	})
	stored := &storage.ImageCVEV2{
		Id:         "cve-1#comp-1",
		HasFixedBy: &storage.ImageCVEV2_FixedBy{FixedBy: "1.2"},
		Advisory:   &storage.Advisory{Name: "RHSA-2030:0001"},
	}

	cases := map[string]struct {
		ctx              context.Context
		flatDataCVEIDs   []string
		expectedCVEIDs   []string
		expectedFixedBy  string
		expectedAdvisory string
	}{
		"without flatData, the occurrence ID is looked up in component scope": {
			ctx:              componentScopeCtx,
			expectedCVEIDs:   []string{"cve-1#comp-1"},
			expectedFixedBy:  "1.2",
			expectedAdvisory: "RHSA-2030:0001",
		},
		"with flatData, the grouped CVE IDs are looked up in component scope": {
			ctx:              componentScopeCtx,
			flatDataCVEIDs:   []string{"cve-1#comp-1", "cve-1#comp-2"},
			expectedCVEIDs:   []string{"cve-1#comp-1", "cve-1#comp-2"},
			expectedFixedBy:  "1.2",
			expectedAdvisory: "RHSA-2030:0001",
		},
		"without flatData and without component scope, nothing is looked up": {
			ctx: context.Background(),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			dataStore := imageCVEV2DSMocks.NewMockDataStore(ctrl)
			resolver := &imageCVEV2Resolver{
				root: &Resolver{ImageCVEV2DataStore: dataStore},
				data: &storage.ImageCVEV2{Id: "cve-1#comp-1"},
			}
			if tc.flatDataCVEIDs != nil {
				flatData := cveFlatMocks.NewMockCveFlat(ctrl)
				flatData.EXPECT().GetCVEIDs().Return(tc.flatDataCVEIDs).AnyTimes()
				resolver.flatData = flatData
			}
			if tc.expectedCVEIDs != nil {
				query := search.NewQueryBuilder().AddExactMatches(search.CVEID, tc.expectedCVEIDs...).ProtoQuery()
				dataStore.EXPECT().
					SearchRawImageCVEs(gomock.Any(), protomock.GoMockMatcherEqualMessage(query)).
					Return([]*storage.ImageCVEV2{stored}, nil).
					Times(2)
			}

			fixedBy, err := resolver.FixedByVersion(tc.ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFixedBy, fixedBy)

			advisory, err := resolver.Advisory(tc.ctx)
			require.NoError(t, err)
			if tc.expectedAdvisory == "" {
				assert.Nil(t, advisory)
				return
			}
			require.NotNil(t, advisory)
			assert.Equal(t, tc.expectedAdvisory, advisory.Name(tc.ctx))
		})
	}
}
