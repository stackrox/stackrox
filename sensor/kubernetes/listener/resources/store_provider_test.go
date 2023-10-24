package resources

import (
	"testing"

	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stretchr/testify/suite"
)

const (
	resourceTypeA = "resource_type_A"
	resourceTypeB = "resource_type_B"
)

type providerSuite struct {
	suite.Suite
	provider *InMemoryStoreProvider
}

var _ suite.SetupTestSuite = (*providerSuite)(nil)

func (s *providerSuite) SetupTest() {
	s.provider = InitializeStore()
}

func Test_StoreProvider(t *testing.T) {
	suite.Run(t, new(providerSuite))
}

func (s *providerSuite) Test_ReconciliationStoreInitialization() {
	testCases := map[string]struct {
		resType       string
		expectedError bool
	}{
		deduper.TypeComplianceOperatorProfile.String(): {
			resType:       deduper.TypeComplianceOperatorProfile.String(),
			expectedError: false,
		},
		deduper.TypeComplianceOperatorResult.String(): {
			resType:       deduper.TypeComplianceOperatorResult.String(),
			expectedError: false,
		},
		deduper.TypeComplianceOperatorRule.String(): {
			resType:       deduper.TypeComplianceOperatorRule.String(),
			expectedError: false,
		},
		deduper.TypeComplianceOperatorScan.String(): {
			resType:       deduper.TypeComplianceOperatorScan.String(),
			expectedError: false,
		},
		deduper.TypeComplianceOperatorScanSettingBinding.String(): {
			resType:       deduper.TypeComplianceOperatorScanSettingBinding.String(),
			expectedError: false,
		},
		"Invalid type": {
			resType:       "Not_supported_type",
			expectedError: true,
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			_, err := s.provider.reconciliationStore.ReconcileDelete(tc.resType, "1", 1111)
			if tc.expectedError {
				s.Assert().Error(err)
			} else {
				s.Assert().NoError(err)
			}

		})
	}
}
