package custom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	identityMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func metricsIdentity(t *testing.T, tokenID, clusterID, resource string) authn.Identity {
	t.Helper()
	identity := identityMocks.NewMockIdentity(gomock.NewController(t))
	identity.EXPECT().UID().AnyTimes().Return(tokenID)
	identity.EXPECT().Roles().AnyTimes().Return([]permissions.ResolvedRole{&tokens.InternalRole{
		RoleName: "internal role",
		Clusters: tokens.ClusterScopes{clusterID: {"*"}},
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS: {resource},
		},
	}})
	return identity
}

func TestRegistryKeyForMetricsToken(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	first := metricsIdentity(t, "token-one", "cluster-one", "Deployment")
	renewed := metricsIdentity(t, "token-two", "cluster-one", "Deployment")
	otherCluster := metricsIdentity(t, "token-three", "cluster-two", "Deployment")
	changedPermissions := metricsIdentity(t, "token-four", "cluster-one", "Image")
	assert.Equal(t, registryKeyForIdentity(ctx, first), registryKeyForIdentity(ctx, renewed))
	assert.NotEqual(t, registryKeyForIdentity(ctx, first), registryKeyForIdentity(ctx, otherCluster))
	assert.NotEqual(t, registryKeyForIdentity(ctx, first), registryKeyForIdentity(ctx, changedPermissions))
	assert.Equal(t, renewed.UID(), registryKeyForIdentity(sac.WithNoAccess(ctx), renewed))
}

func TestRunnerReusesMetricsRegistryAfterTokenRotation(t *testing.T) {
	first := metricsIdentity(t, "token-one", "cluster-one", "Deployment")
	renewed := metricsIdentity(t, "token-two", "cluster-one", "Deployment")
	otherCluster := metricsIdentity(t, "token-three", "cluster-two", "Deployment")
	const metric = "metrics_rotation_test"
	registryKey := registryKeyForIdentity(sac.WithAllAccess(context.Background()), first)
	registry, err := metrics.GetCustomRegistry(registryKey)
	require.NoError(t, err)
	t.Cleanup(func() { metrics.DeleteCustomRegistry(registryKey) })
	require.NoError(t, registry.RegisterMetric(metric, "test metrics cache isolation", nil))
	registry.SetTotal(metric, nil, 1)
	runner := trackerRunner{}
	for _, test := range []struct {
		name     string
		identity authn.Identity
		want     string
		denied   bool
	}{
		{name: "first token", identity: first, want: "rox_central_" + metric + " 1"},
		{name: "renewed token", identity: renewed, want: "rox_central_" + metric + " 1"},
		{name: "cluster removed before renewal", identity: renewed, denied: true},
		{name: "other cluster", identity: otherCluster},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := authn.ContextWithIdentity(context.Background(), test.identity, t)
			if test.denied {
				ctx = sac.WithNoAccess(ctx)
			} else {
				ctx = sac.WithAllAccess(ctx)
			}
			rec := httptest.NewRecorder()
			runner.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/metrics", nil))
			require.Equal(t, http.StatusOK, rec.Code)
			if test.want == "" {
				assert.NotContains(t, rec.Body.String(), "rox_central_"+metric)
			} else {
				assert.Contains(t, rec.Body.String(), test.want)
			}
		})
	}
	t.Cleanup(func() {
		metrics.DeleteCustomRegistry(registryKeyForIdentity(sac.WithAllAccess(context.Background()), otherCluster))
	})
	t.Cleanup(func() { metrics.DeleteCustomRegistry(renewed.UID()) })
}
