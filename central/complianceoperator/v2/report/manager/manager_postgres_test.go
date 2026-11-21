//go:build sql_integration

package manager

import (
	"context"
	"testing"

	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/central/globaldb"
	sacEnricher "github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
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

func TestReportScopeSurvivesRequestCancellation(t *testing.T) {
	t.Setenv(features.ComplianceReporting.EnvVar(), "true")
	db := pgtest.ForT(t)
	previousDB := globaldb.GetPostgres()
	globaldb.SetPostgresTest(t, db)
	t.Cleanup(func() { globaldb.SetPostgresTest(t, previousDB) })
	cluster := &storage.Cluster{Id: uuid.NewV4().String(), Name: "report-cluster"}
	require.NoError(t, clusterStore.New(db).Upsert(sac.WithAllAccess(t.Context()), cluster))
	roles := []permissions.ResolvedRole{roletest.NewResolvedRole("report-role",
		map[string]storage.Access{resources.Compliance.String(): storage.Access_READ_ACCESS},
		&storage.SimpleAccessScope{Id: "report-scope", Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{cluster.GetName()},
		}})}
	identity := identityMocks.NewMockIdentity(gomock.NewController(t))
	identity.EXPECT().Service().AnyTimes().Return(nil)
	identity.EXPECT().ExternalAuthProvider().AnyTimes().Return(nil)
	identity.EXPECT().Roles().AnyTimes().Return(roles)
	requestCtx, cancel := context.WithCancel(authn.ContextWithIdentity(t.Context(), identity, t))
	defer cancel()
	requestCtx, err := sacEnricher.GetEnricher().EnrichContext(requestCtx)
	require.NoError(t, err)
	manager := &managerImpl{
		runningReportConfigs: make(map[string]*reportRequest),
		reportRequests:       make(chan *reportRequest, 1),
	}
	require.NoError(t, manager.SubmitReportRequest(requestCtx,
		&storage.ComplianceOperatorScanConfigurationV2{Id: "report-config"}, storage.ComplianceOperatorReportStatus_EMAIL))
	queued := <-manager.reportRequests
	cancel()
	resource := permissions.ResourceWithAccess{Resource: resources.Compliance, Access: storage.Access_READ_ACCESS}
	_, err = sac.GlobalAccessScopeChecker(requestCtx).EffectiveAccessScope(resource)
	require.ErrorIs(t, err, context.Canceled)
	_, err = sac.GlobalAccessScopeChecker(queued.ctx).EffectiveAccessScope(resource)
	require.NoError(t, err)
	checker := sac.ForResource(resources.Compliance).ScopeChecker(queued.ctx, storage.Access_READ_ACCESS)
	require.True(t, checker.IsAllowed(sac.ClusterScopeKey(cluster.GetId())))
	require.False(t, checker.IsAllowed(sac.ClusterScopeKey(uuid.NewV4().String())))
}
