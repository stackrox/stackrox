package centralsensor

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClusterID  = "12b1af66-be55-4e54-948d-ac9c311ca4b2"
	otherClusterID = "d2842760-c0f5-4574-80a1-58c300c36375"
)

func TestUserIssuedInitCertClusterIDIsNilUUID(t *testing.T) {
	parsed, err := uuid.FromString(RegisteredInitCertClusterID)
	require.NoError(t, err, "could not parse init cert cluster ID as UUID")
	assert.Equal(t, uuid.Nil, parsed, "expected init cert cluster ID to be the nil UUID")
}

func TestGetClusterID(t *testing.T) {
	cases := map[string]struct {
		explicitID, idFromCert string
		expectedID             string
		expectError            bool
	}{
		// No explicit ID, concrete ID in cert
		"no-explicit-id": {
			explicitID: "",
			idFromCert: testClusterID,
			expectedID: testClusterID,
		},
		// Same explicit and cert ID
		"explicit-id-no-wildcard": {
			explicitID: testClusterID,
			idFromCert: testClusterID,
			expectedID: testClusterID,
		},
		// Error case where an incorrect explicit ID is set for a non-wildcard cert
		"incorrect-id-no-wildcard": {
			explicitID:  otherClusterID,
			idFromCert:  testClusterID,
			expectError: true,
		},
		// Feature flag enabled: explicit ID and registered wildcard cert
		"explicit-id-and-registered-wildcard-cert": {
			explicitID: testClusterID,
			idFromCert: RegisteredInitCertClusterID,
			expectedID: testClusterID,
		},
		// Feature flag enabled: explicit nil ID and registered wildcard cert
		"explicit-nil-id-and-registered-wildcard-cert": {
			explicitID:  RegisteredInitCertClusterID,
			idFromCert:  RegisteredInitCertClusterID,
			expectError: true,
		},
		"no-explicit-id-and-registered-wildcard-cert": {
			explicitID:  "",
			idFromCert:  RegisteredInitCertClusterID,
			expectError: true,
		},
		// Feature flag enabled: explicit ID and ephemeral wildcard cert
		"explicit-id-and-ephemeral-wildcard-cert": {
			explicitID: testClusterID,
			idFromCert: EphemeralInitCertClusterID,
			expectedID: testClusterID,
		},
		// Feature flag enabled: explicit nil ID and ephemeral wildcard cert
		"explicit-nil-id-and-ephemeral-wildcard-cert": {
			explicitID:  RegisteredInitCertClusterID,
			idFromCert:  EphemeralInitCertClusterID,
			expectError: true,
		},
		"no-explicit-id-and-ephemeral-wildcard-cert": {
			explicitID:  "",
			idFromCert:  EphemeralInitCertClusterID,
			expectError: true,
		},
	}

	for cName, c := range cases {
		t.Run(cName, func(t *testing.T) {
			clusterID, err := GetClusterID(c.explicitID, c.idFromCert)
			if c.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.expectedID, clusterID)
			}
		})
	}
}
