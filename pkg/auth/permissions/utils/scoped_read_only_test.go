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

func TestValidateScopedReadOnlyPermissionSet(t *testing.T) {
	for name, tc := range map[string]struct {
		set     *storage.PermissionSet
		wantErr bool
	}{
		"missing": {wantErr: true},
		"constrained": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}},
		"missing read constraint": {set: &storage.PermissionSet{
			ScopeConstraint: storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		}, wantErr: true},
		"missing scope constraint": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
		}, wantErr: true},
		"stored write despite constraints": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_WRITE_ACCESS},
		}, wantErr: true},
		"stored global read despite constraints": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Administration": storage.Access_READ_ACCESS},
		}, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateScopedReadOnlyPermissionSet(tc.set)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
