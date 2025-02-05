package service

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/pkg/errors"
	iiDSMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/pkg/features"
	grpctestutils "github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	grpctestutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestEnsureTLSAndReturnAddr(t *testing.T) {
	for _, testCase := range []struct {
		endpoint    string
		expectedOut string
		errExpected bool
	}{
		{
			endpoint: "scanner.stackrox", errExpected: true,
		},
		{
			endpoint: "http://scanner.stackrox", errExpected: true,
		},
		{
			endpoint: "https://scanner.stackrox", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox:8080", expectedOut: "scanner.stackrox:8080",
		},
		{
			endpoint: "https://scanner.stackrox/", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox/ping", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox:8080/", expectedOut: "scanner.stackrox:8080",
		},
		{
			endpoint: "https://scanner.stackrox:8080/ping", expectedOut: "scanner.stackrox:8080",
		},
	} {
		c := testCase
		t.Run(c.endpoint, func(t *testing.T) {
			got, err := ensureTLSAndReturnAddr(c.endpoint)
			if c.errExpected {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.expectedOut, got)
		})
	}
}

func TestGetScannerV4CertExpiry(t *testing.T) {
	expiryCur := time.Now()
	expiryOld := time.Date(2000, 01, 01, 1, 1, 1, 1, time.UTC)

	successExpiryFunc := genGetExpiryFunc(map[mtls.Subject]*time.Time{
		mtls.ScannerV4IndexerSubject: &expiryOld,
		mtls.ScannerV4MatcherSubject: &expiryCur,
	})

	matcherSuccessExpiryFunc := genGetExpiryFunc(map[mtls.Subject]*time.Time{
		mtls.ScannerV4MatcherSubject: &expiryCur,
	})

	indexerSuccessExpiryFunc := genGetExpiryFunc(map[mtls.Subject]*time.Time{
		mtls.ScannerV4IndexerSubject: &expiryOld,
	})

	errorExpiryFunc := genGetExpiryFunc(nil)

	allScannerConfigs := map[mtls.Subject]*tls.Config{
		mtls.ScannerV4IndexerSubject: {},
		mtls.ScannerV4MatcherSubject: {},
	}

	noScannerConfigs := map[mtls.Subject]*tls.Config{}

	// For readability.
	feature := true
	errorExpected := true
	getIntegrationError := true
	integrationExists := true

	testCases := []struct {
		desc                string
		featureEnabled      bool
		errorExpected       bool
		scannerConfigs      map[mtls.Subject]*tls.Config
		getIntegrationError bool
		integrationExists   bool
		expiryFunc          func(context.Context, mtls.Subject, *tls.Config, string) (*time.Time, error)
		expiryExpected      *time.Time
	}{
		{
			"error if feature disabled",
			!feature, errorExpected, allScannerConfigs, !getIntegrationError, integrationExists, successExpiryFunc, &expiryOld,
		},
		{
			"error if scanner config missing",
			feature, errorExpected, noScannerConfigs, !getIntegrationError, integrationExists, successExpiryFunc, &expiryOld,
		},
		{
			"error if failure getting scanner v4 integration",
			feature, errorExpected, allScannerConfigs, getIntegrationError, integrationExists, successExpiryFunc, &expiryOld,
		},
		{
			"error if no scanner v4 integration",
			feature, errorExpected, allScannerConfigs, !getIntegrationError, !integrationExists, successExpiryFunc, &expiryOld,
		},
		{
			"error if fail contacting both indexer and matcher",
			feature, errorExpected, allScannerConfigs, !getIntegrationError, integrationExists, errorExpiryFunc, &expiryOld,
		},
		{
			"success if indexer and matcher able to be contacted",
			feature, !errorExpected, allScannerConfigs, !getIntegrationError, integrationExists, successExpiryFunc, &expiryOld,
		},
		{
			"success if contacted matcher but failed contacting indexer",
			feature, !errorExpected, allScannerConfigs, !getIntegrationError, integrationExists, matcherSuccessExpiryFunc, &expiryCur,
		},
		{
			"success if contacted indexer but failed contacting matcher",
			feature, !errorExpected, allScannerConfigs, !getIntegrationError, integrationExists, indexerSuccessExpiryFunc, &expiryOld,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.featureEnabled {
				testutils.MustUpdateFeature(t, features.ScannerV4, true)
			} else {
				testutils.MustUpdateFeature(t, features.ScannerV4, false)
			}

			ctrl := gomock.NewController(t)
			iiDSMock := iiDSMocks.NewMockDataStore(ctrl)
			if tc.getIntegrationError {
				iiDSMock.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("fake")).AnyTimes()
			} else if !tc.integrationExists {
				iiDSMock.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, nil).AnyTimes()
			} else {
				iiDSMock.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(iiStore.DefaultScannerV4Integration, true, nil).AnyTimes()
			}

			s := &serviceImpl{
				imageIntegrations: iiDSMock,
				scannerConfigs:    tc.scannerConfigs,
				expiryFunc:        tc.expiryFunc,
			}
			actual, err := s.getScannerV4CertExpiry(context.Background())
			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expiryExpected == nil {
					assert.Nil(t, actual.Expiry)
				} else {
					expectedExpiry, err := protocompat.ConvertTimeToTimestampOrError(*tc.expiryExpected)
					require.NoError(t, err)
					assert.Equal(t, expectedExpiry.AsTime(), actual.Expiry.AsTime())
				}
			}
		})
	}
}

func genGetExpiryFunc(expiries map[mtls.Subject]*time.Time) func(context.Context, mtls.Subject, *tls.Config, string) (*time.Time, error) {
	return func(_ context.Context, subject mtls.Subject, _ *tls.Config, _ string) (*time.Time, error) {
		expiry, ok := expiries[subject]
		if !ok {
			return nil, errors.New("fake")
		}

		return expiry, nil
	}
}
