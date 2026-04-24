package resolvers

import (
	"context"
	"testing"

	cveFlatMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
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

	cases := map[string]struct {
		resolver         *imageCVEV2Resolver
		expectedSeverity string
		expectedCvss     float64
		expectedNvdCvss  float64
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
			expectedCvss:     9.8,
			expectedNvdCvss:  9.1,
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
			expectedCvss:     7.5,
			expectedNvdCvss:  6.9,
		},
	}

	ctx := context.Background()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectedSeverity, tc.resolver.Severity(ctx))
			assert.InDelta(t, tc.expectedCvss, tc.resolver.Cvss(ctx), 0.001)
			assert.InDelta(t, tc.expectedNvdCvss, tc.resolver.Nvdcvss(ctx), 0.001)
		})
	}
}
