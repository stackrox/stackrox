package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

func TestValidateScopedReadOnlyPermissions(t *testing.T) {
	for name, tc := range map[string]struct {
		grants  map[string]storage.Access
		wantErr bool
	}{
		"empty": {},
		"scoped reads and denied global": {
			grants: map[string]storage.Access{
				"Deployment":     storage.Access_READ_ACCESS,
				"Node":           storage.Access_READ_ACCESS,
				"Administration": storage.Access_NO_ACCESS,
			},
		},
		"global read": {
			grants:  map[string]storage.Access{"Administration": storage.Access_READ_ACCESS},
			wantErr: true,
		},
		"unknown resource read": {
			grants:  map[string]storage.Access{"UnknownResource": storage.Access_READ_ACCESS},
			wantErr: true,
		},
		"scoped write": {
			grants:  map[string]storage.Access{"Deployment": storage.Access_READ_WRITE_ACCESS},
			wantErr: true,
		},
		"unknown access": {
			grants:  map[string]storage.Access{"Deployment": storage.Access(99)},
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateScopedReadOnlyPermissions(tc.grants)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
