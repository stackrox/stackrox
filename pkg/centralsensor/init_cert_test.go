package centralsensor

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testClusterID  = "12b1af66-be55-4e54-948d-ac9c311ca4b2"
	otherClusterID = "d2842760-c0f5-4574-80a1-58c300c36375"
)

func TestInitCertClusterIDIsNilUUID(t *testing.T) {
	parsed, err := uuid.FromString(InitCertClusterID)
	require.NoError(t, err, "could not parse init cert cluster ID as UUID")
	assert.Equal(t, uuid.Nil, parsed, "expected init cert cluster ID to be the nil UUID")
}

func TestGetClusterID(t *testing.T) {
	cases := map[string]struct {
		explicitID, idFromCert string
		flagSettings           []bool
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
		// Feature flag disabled: no explicit ID, nil ID in cert (odd but okay)
		"no-explicit-and-nil-cert-id": {
			explicitID:   "",
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{false},
			expectedID:   InitCertClusterID,
		},
		// Feature flag disabled: nil ID set explicitly and in cert
		"nil-explicit-and-cert-id": {
			explicitID:   InitCertClusterID,
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{false},
			expectedID:   InitCertClusterID,
		},
		// Feature flag disabled: non-nil ID set explicitly in conjunction with nil cert (not recognized as wildcard)
		"explicit-and-nil-cert-id": {
			explicitID:   testClusterID,
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{false},
			expectError:  true,
		},
		// Feature flag enabled: explicit ID and wildcard cert
		"explicit-id-and-wildcard-cert": {
			explicitID:   testClusterID,
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{true},
			expectedID:   testClusterID,
		},
		// Feature flag enabled: explicit nil ID and wildcard cert
		"explicit-nil-id-and-wildcard-cert": {
			explicitID:   InitCertClusterID,
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{true},
			expectError:  true,
		},
		"no-explicit-id-and-wildcard-cert": {
			explicitID:   "",
			idFromCert:   InitCertClusterID,
			flagSettings: []bool{true},
			expectError:  true,
		},
	}

	for cName, c := range cases {
		flagValues := c.flagSettings
		if len(flagValues) == 0 {
			flagValues = []bool{false, true}
		}

		for _, flagValue := range flagValues {
			t.Run(fmt.Sprintf("%s/featureFlag=%t", cName, flagValue), func(t *testing.T) {
				if buildinfo.ReleaseBuild && flagValue != features.SensorInstallationExperience.Enabled() {
					t.Skip("cannot override feature flags on release builds")
				}

				ei := envisolator.NewEnvIsolator(t)
				defer ei.RestoreAll()

				ei.Setenv(features.SensorInstallationExperience.EnvVar(), strconv.FormatBool(flagValue))

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
}
