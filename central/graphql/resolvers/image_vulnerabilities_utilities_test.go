package resolvers

import (
	"context"
	"testing"

	cveFlatMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func floatPtr(f float64) *float64 {
	return &f
}

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
	zeroFlatData.EXPECT().GetTopCVSS().Return(float32(0.0)).AnyTimes()
	zeroFlatData.EXPECT().GetTopNVDCVSS().Return(float32(0.0)).AnyTimes()

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
			expectedCvss:     floatPtr(9.8),
			expectedNvdCvss:  floatPtr(9.1),
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
			expectedCvss:     floatPtr(7.5),
			expectedNvdCvss:  floatPtr(6.9),
		},
		"zero CVSS returns nil": {
			resolver: &imageCVEV2Resolver{
				data: &storage.ImageCVEV2{
					Severity: important,
					Cvss:     0.0,
					Nvdcvss:  0.0,
				},
			},
			expectedSeverity: important.String(),
			expectedCvss:     nil,
			expectedNvdCvss:  nil,
		},
		"zero CVSS in flatData returns nil": {
			resolver: &imageCVEV2Resolver{
				data: &storage.ImageCVEV2{
					Severity: important,
					Cvss:     7.5,
					Nvdcvss:  6.9,
				},
				flatData: zeroFlatData,
			},
			expectedSeverity: critical.String(),
			expectedCvss:     nil,
			expectedNvdCvss:  nil,
		},
	}

	ctx := context.Background()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectedSeverity, tc.resolver.Severity(ctx))
			cvss := tc.resolver.Cvss(ctx)
			nvdCvss := tc.resolver.Nvdcvss(ctx)
			if tc.expectedCvss == nil {
				assert.Nil(t, cvss)
			} else {
				assert.InDelta(t, *tc.expectedCvss, *cvss, 0.001)
			}
			if tc.expectedNvdCvss == nil {
				assert.Nil(t, nvdCvss)
			} else {
				assert.InDelta(t, *tc.expectedNvdCvss, *nvdCvss, 0.001)
			}
		})
	}
}
