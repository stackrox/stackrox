//go:build sql_integration

package custom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	sacEnricher "github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	identityMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type contextCapturingTracker struct {
	tracker.Tracker
	contexts chan context.Context
}

func (tr contextCapturingTracker) Gather(ctx context.Context) {
	tr.contexts <- ctx
}

func TestRunnerBackgroundScopeSurvivesRequestCancellation(t *testing.T) {
	db := pgtest.ForT(t)
	previousDB := globaldb.GetPostgres()
	globaldb.SetPostgresTest(t, db)
	t.Cleanup(func() { globaldb.SetPostgresTest(t, previousDB) })
	cluster := &storage.Cluster{Id: uuid.NewV4().String(), Name: "metrics-cluster"}
	require.NoError(t, clusterStore.New(db).Upsert(sac.WithAllAccess(t.Context()), cluster))
	scope := &storage.SimpleAccessScope{Id: "metrics-scope", Rules: &storage.SimpleAccessScope_Rules{
		IncludedClusters: []string{cluster.GetName()},
	}}
	roles := []permissions.ResolvedRole{roletest.NewResolvedRole("metrics-role",
		map[string]storage.Access{resources.Deployment.String(): storage.Access_READ_ACCESS}, scope)}
	identity := identityMocks.NewMockIdentity(gomock.NewController(t))
	identity.EXPECT().UID().Return(uuid.NewV4().String()).AnyTimes()
	identity.EXPECT().Service().Return(nil).AnyTimes()
	identity.EXPECT().ExternalAuthProvider().Return(nil).AnyTimes()
	identity.EXPECT().Roles().Return(roles).AnyTimes()
	requestCtx, cancel := context.WithCancel(authn.ContextWithIdentity(t.Context(), identity, t))
	defer cancel()
	requestCtx, err := sacEnricher.GetEnricher().EnrichContext(requestCtx)
	require.NoError(t, err)
	backgroundContexts := make(chan context.Context, 1)
	runner := trackerRunner{{Tracker: contextCapturingTracker{contexts: backgroundContexts}}}
	recorder := httptest.NewRecorder()
	runner.ServeHTTP(recorder, httptest.NewRequestWithContext(requestCtx, http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	cancel()
	resource := permissions.ResourceWithAccess{Resource: resources.Deployment, Access: storage.Access_READ_ACCESS}
	_, err = sac.GlobalAccessScopeChecker(requestCtx).EffectiveAccessScope(resource)
	require.ErrorIs(t, err, context.Canceled, "request SQL must still honor request cancellation")
	var backgroundCtx context.Context
	select {
	case backgroundCtx = <-backgroundContexts:
	case <-time.After(time.Second):
		t.Fatal("metrics collection did not start")
	}
	require.NoError(t, backgroundCtx.Err())
	require.Same(t, identity, authn.IdentityFromContextOrNil(backgroundCtx))
	_, err = sac.GlobalAccessScopeChecker(backgroundCtx).EffectiveAccessScope(resource)
	require.NoError(t, err, "background SQL must not retain the canceled request or its cached failure")
	checker := sac.ForResource(resources.Deployment).ScopeChecker(backgroundCtx, storage.Access_READ_ACCESS)
	require.True(t, checker.IsAllowed(sac.ClusterScopeKey(cluster.GetId()), sac.NamespaceScopeKey("any")))
	require.False(t, checker.IsAllowed(sac.ClusterScopeKey(uuid.NewV4().String()), sac.NamespaceScopeKey("any")))
}
