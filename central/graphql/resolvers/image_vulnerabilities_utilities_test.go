package resolvers

import (
	"context"
	"testing"

	cveFlatMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	"github.com/stackrox/rox/generated/storage"
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
