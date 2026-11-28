package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/role"
	storeMocks "github.com/stackrox/rox/central/role/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSeedDefaultPermissionSets(t *testing.T) {
	for name, tc := range map[string]struct {
		consoleErr error
		otherErr   error
	}{
		"success": {},
		"console name collision": {
			consoleErr: errox.AlreadyExists.New("name collision"),
		},
		"console storage error": {
			consoleErr: errors.New("database unavailable"),
		},
		"other default failure": {
			otherErr: errors.New("database unavailable"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
			console := getDefaultPermissionSet(accesscontrol.K8sCentralProxyUser)
			other := getDefaultPermissionSet(accesscontrol.Admin)
			store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{other}).Return(tc.otherErr)
			if tc.otherErr == nil {
				store.EXPECT().Upsert(ctx, console).Return(tc.consoleErr)
			}
			err := seedDefaultPermissionSets(ctx, store, []*storage.PermissionSet{console, other})
			if tc.otherErr != nil {
				require.ErrorIs(t, err, tc.otherErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSeedMetricsPermissionSet(t *testing.T) {
	for name, openshift := range map[string]string{
		"OpenShift Central":     "true",
		"non-OpenShift Central": "false",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.Openshift.EnvVar(), openshift)
			ctx := t.Context()
			store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
			gomock.InOrder(
				store.EXPECT().Exists(ctx, accesscontrol.OpenShiftPrometheusMetricsReaderID).Return(false, nil),
				store.EXPECT().Upsert(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, ps *storage.PermissionSet) error {
					assert.Equal(t, "b3070020-ecc3-4f34-a3f6-ad28ffc9b80e", ps.GetId())
					assert.Equal(t, "OpenShift Prometheus Metrics Reader", ps.GetName())
					assert.Empty(t, ps.GetResourceToAccess())
					assert.Equal(t, storage.Traits_IMPERATIVE, ps.GetTraits().GetOrigin())
					assert.False(t, role.IsDefaultPermissionSet(ps))
					assert.True(t, declarativeconfig.CanModifyResource(ctx, ps))
					require.NoError(t, role.ValidatePermissionSet(ps))
					require.True(t, ps.GetAccessConstraint() == storage.AccessConstraint_ACCESS_CONSTRAINT_READ &&
						ps.GetScopeConstraint() == storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE)
					return nil
				}),
				store.EXPECT().Exists(ctx, accesscontrol.OpenShiftPrometheusMetricsReaderID).Return(true, nil),
			)
			seedMetricsPermissionSet(ctx, store)
			seedMetricsPermissionSet(ctx, store)
		})
	}
}

func TestSeedMetricsPermissionSetPreservesExisting(t *testing.T) {
	ctx := t.Context()
	store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
	// Existing permissions and constraints are administrator-managed and must not be rewritten.
	store.EXPECT().Exists(ctx, accesscontrol.OpenShiftPrometheusMetricsReaderID).Return(true, nil)
	seedMetricsPermissionSet(ctx, store)
}

func TestSeedMetricsPermissionSetStorageFailures(t *testing.T) {
	for name, tc := range map[string]struct {
		readErr  error
		writeErr error
	}{
		"read failure":  {readErr: errors.New("database unavailable")},
		"write failure": {writeErr: errors.New("database unavailable")},
		"name collision": {
			writeErr: errox.AlreadyExists.New("name collision"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
			store.EXPECT().Exists(ctx, accesscontrol.OpenShiftPrometheusMetricsReaderID).Return(false, tc.readErr)
			if tc.readErr == nil {
				store.EXPECT().Upsert(ctx, gomock.Any()).Return(tc.writeErr)
			}
			seedMetricsPermissionSet(ctx, store)
		})
	}
}
