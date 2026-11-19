//go:build sql_integration

package service

import (
	"context"
	"testing"
	"time"

	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/globaldb"
	sacEnricher "github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/backgroundtasks"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
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

func TestDryRunJobScopeSurvivesRequestCancellation(t *testing.T) {
	db := pgtest.ForT(t)
	previousDB := globaldb.GetPostgres()
	globaldb.SetPostgresTest(t, db)
	t.Cleanup(func() { globaldb.SetPostgresTest(t, previousDB) })
	cluster := &storage.Cluster{Id: uuid.NewV4().String(), Name: "dry-run-cluster"}
	require.NoError(t, clusterStore.New(db).Upsert(sac.WithAllAccess(t.Context()), cluster))
	roles := []permissions.ResolvedRole{roletest.NewResolvedRole("dry-run-role",
		map[string]storage.Access{resources.Deployment.String(): storage.Access_READ_ACCESS},
		&storage.SimpleAccessScope{Id: "dry-run-scope", Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{cluster.GetName()},
		}})}
	ctrl := gomock.NewController(t)
	identity := identityMocks.NewMockIdentity(ctrl)
	identity.EXPECT().UID().AnyTimes().Return("dry-run-user")
	identity.EXPECT().Service().AnyTimes().Return(nil)
	identity.EXPECT().ExternalAuthProvider().AnyTimes().Return(nil)
	identity.EXPECT().Roles().AnyTimes().Return(roles)
	requestCtx, cancel := context.WithCancel(authn.ContextWithIdentity(t.Context(), identity, t))
	defer cancel()
	requestCtx, err := sacEnricher.GetEnricher().EnrichContext(requestCtx)
	require.NoError(t, err)

	type scopeResult struct {
		err     error
		allowed bool
		foreign bool
	}
	checked := make(chan scopeResult, 1)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	deployments.EXPECT().GetDeploymentIDs(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]string, error) {
		resource := permissions.ResourceWithAccess{Resource: resources.Deployment, Access: storage.Access_READ_ACCESS}
		_, scopeErr := sac.GlobalAccessScopeChecker(ctx).EffectiveAccessScope(resource)
		result := scopeResult{err: scopeErr}
		if scopeErr == nil {
			checker := sac.ForResource(resources.Deployment).ScopeChecker(ctx, storage.Access_READ_ACCESS)
			result.allowed = checker.IsAllowed(sac.ClusterScopeKey(cluster.GetId()), sac.NamespaceScopeKey("any"))
			result.foreign = checker.IsAllowed(sac.ClusterScopeKey(uuid.NewV4().String()), sac.NamespaceScopeKey("any"))
		}
		checked <- result
		return nil, scopeErr
	})
	manager := backgroundtasks.NewManager()
	service := &serviceImpl{
		deployments:            deployments,
		validator:              newPolicyValidator(nil),
		dryRunPolicyJobManager: manager,
	}
	policy := &storage.Policy{
		Name:            "Detached policy dry run",
		Severity:        storage.Severity_LOW_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"test"},
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{{PolicyGroups: []*storage.PolicyGroup{{
			FieldName: fieldnames.PrivilegedContainer,
			Values:    []*storage.PolicyValue{{Value: "true"}},
		}}}},
	}
	_, err = service.SubmitDryRunPolicyJob(requestCtx, policy)
	require.NoError(t, err)
	cancel()
	manager.Start()
	select {
	case result := <-checked:
		require.NoError(t, result.err)
		require.True(t, result.allowed)
		require.False(t, result.foreign)
	case <-time.After(5 * time.Second):
		t.Fatal("policy dry-run job did not check its access scope")
	}
}
