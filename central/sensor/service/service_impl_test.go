package service

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
)

func TestGetCertExpiryStatus(t *testing.T) {
	type testCase struct {
		notBefore, notAfter time.Time
		expectedStatus      *storage.ClusterCertExpiryStatus
	}
	testCases := map[string]testCase{
		"should return nil when no dates": {
			expectedStatus: nil,
		},
		"should fill not before only if expiry is not set": {
			notBefore: time.Unix(1646870400, 0), // Thu Mar 10 2022 00:00:00 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertNotBefore: &types.Timestamp{
					Seconds: 1646870400,
				},
			},
		},
		"should fill expiry only if notbefore is not set": {
			notAfter: time.Unix(1646956799, 0), // Thu Mar 10 2022 23:59:59 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertExpiry: &types.Timestamp{
					Seconds: 1646956799,
				},
			},
		},
		"should fill status if both bounds are set": {
			notBefore: time.Unix(1646870400, 0), // Thu Mar 10 2022 00:00:00 GMT+0000
			notAfter:  time.Unix(1646956799, 0), // Thu Mar 10 2022 23:59:59 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertNotBefore: &types.Timestamp{
					Seconds: 1646870400,
				},
				SensorCertExpiry: &types.Timestamp{
					Seconds: 1646956799,
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			identity := service.WrapMTLSIdentity(mtls.IdentityFromCert(mtls.CertInfo{
				NotBefore: tc.notBefore,
				NotAfter:  tc.notAfter,
			}))
			result, err := getCertExpiryStatus(identity)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, result)
		})
	}
}
