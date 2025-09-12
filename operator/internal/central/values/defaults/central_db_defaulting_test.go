package defaults

import (
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestCentralDBDefaulting(t *testing.T) {
	centralSpecWithDefaultedPVCClaimName := &platform.CentralSpec{
		Central: &platform.CentralComponentSpec{
			DB: &platform.CentralDBSpec{
				Persistence: &platform.DBPersistence{
					PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
						ClaimName: ptr.To("central-db"),
					},
				},
			},
		},
	}
	tests := map[string]struct {
		spec             *platform.CentralSpec
		expectedDefaults *platform.CentralSpec
		expectedError    require.ErrorAssertionFunc
	}{
		"nil spec should apply defaults": {
			spec:             nil,
			expectedDefaults: centralSpecWithDefaultedPVCClaimName,
		},
		"spec with nil Central should apply defaults": {
			spec:             &platform.CentralSpec{},
			expectedDefaults: centralSpecWithDefaultedPVCClaimName,
		},
		"spec with Central but no DB should apply defaults": {
			spec: &platform.CentralSpec{
				Central: &platform.CentralComponentSpec{},
			},
			expectedDefaults: centralSpecWithDefaultedPVCClaimName,
		},
		"spec with Central and DB but no ConnectionString should apply defaults": {
			spec: &platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					DB: &platform.CentralDBSpec{},
				},
			},
			expectedDefaults: centralSpecWithDefaultedPVCClaimName,
		},
		"external DB with connection string should not apply defaults": {
			spec: &platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					DB: &platform.CentralDBSpec{
						ConnectionStringOverride: ptr.To("postgresql://external:5432/db"),
					},
				},
			},
			expectedDefaults: &platform.CentralSpec{},
		},
		"external DB with connection string and persistence should error": {
			spec: &platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					DB: &platform.CentralDBSpec{
						ConnectionStringOverride: ptr.To("postgresql://external:5432/db"),
						Persistence:              &platform.DBPersistence{},
					},
				},
			},
			expectedError: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "if a connection string is provided, no persistence settings must be supplied")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defaults := &platform.CentralSpec{}

			err := centralDBPersistenceDefaulting(logr.Discard(), nil, nil, tt.spec, defaults)
			if tt.expectedError != nil {
				assert.Nil(t, tt.expectedDefaults, "expected defaults should not be specified if error is")
				tt.expectedError(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedDefaults, defaults, "defaults should match expected values")
		})
	}
}
