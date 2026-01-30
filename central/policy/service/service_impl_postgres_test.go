//go:build sql_integration

package service

import (
	"context"
	"testing"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	networkPoliciesDatastore "github.com/stackrox/rox/central/networkpolicies/datastore"
	notifierDatastore "github.com/stackrox/rox/central/notifier/datastore"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	policyStore "github.com/stackrox/rox/central/policy/store"
	policyCategoryDatastore "github.com/stackrox/rox/central/policycategory/datastore"
	categoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	edgeDataStore "github.com/stackrox/rox/central/policycategoryedge/datastore"
	edgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicyServiceWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyServicePostgresSuite))
}

type PolicyServicePostgresSuite struct {
	suite.Suite
	mockCtrl          *gomock.Controller
	ctx               context.Context
	db                *pgtest.TestPostgres
	policies          policyDatastore.DataStore
	categories        policyCategoryDatastore.DataStore
	clusters          clusterDatastore.DataStore
	namespaces        namespaceDatastore.DataStore
	mitreVectorStore  mitreDataStore.AttackReadOnlyDataStore
	lifecycleManager  *lifecycleMocks.MockManager
	connectionManager *connectionMocks.MockManager
	tested            Service
}

func (s *PolicyServicePostgresSuite) SetupSuite() {
	s.T().Setenv("ROX_IMAGE_FLAVOR", "opensource")

	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())

	policyStorage := policyStore.New(s.db)

	notifierDS := notifierDatastore.GetTestPostgresDataStore(s.T(), s.db)

	categoryStorage := categoryPostgres.New(s.db)

	edgeStorage := edgePostgres.New(s.db)

	edgeDatastore := edgeDataStore.New(edgeStorage)

	s.categories = policyCategoryDatastore.New(categoryStorage, edgeDatastore)

	s.policies = policyDatastore.New(policyStorage, s.clusters, notifierDS, s.categories)

	var err error
	s.clusters, err = clusterDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	s.namespaces, err = namespaceDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	deployments, err := deploymentDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	networkPolicies, err := networkPoliciesDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	s.mitreVectorStore = mitreDataStore.NewMitreAttackStore()

	s.mockCtrl = gomock.NewController(s.T())

	s.lifecycleManager = lifecycleMocks.NewMockManager(s.mockCtrl)

	s.connectionManager = connectionMocks.NewMockManager(s.mockCtrl)

	s.tested = New(s.policies, s.clusters, deployments, s.namespaces, networkPolicies, notifierDS, s.mitreVectorStore, nil, s.lifecycleManager, nil, nil, s.connectionManager)
}

// TestPostPolicy tests posting and then immediately after putting the same policy, as this discovered a bug in the
// title casing of policy categories. This caused the policy as code workflow to create a new policy CR with a new policy
// category name that did not conform to "title" casing to fail (ROX-26676).
func (s *PolicyServicePostgresSuite) TestPutAfterPostPolicyWithInvalidCasing() {
	policy := &storage.Policy{
		Name:            "Test Policy",
		Description:     "Test Description",
		Categories:      []string{"Not a Real Category"},
		Severity:        storage.Severity_CRITICAL_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Remote",
						Values: []*storage.PolicyValue{
							{
								Value: "nginx",
							},
						},
					},
				},
			},
		},
	}

	s.lifecycleManager.EXPECT().UpsertPolicy(gomock.Any()).AnyTimes()

	s.connectionManager.EXPECT().PreparePoliciesAndBroadcast(gomock.Any()).AnyTimes()

	categoryCount, err := s.categories.Count(s.ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	s.Equal(0, categoryCount)

	postedPolicy, err := s.tested.PostPolicy(s.ctx, &v1.PostPolicyRequest{
		Policy:                 policy,
		EnableStrictValidation: true,
	})
	s.NotNil(postedPolicy)
	s.NoError(err)
	log.Infof("Posted policy: %v", postedPolicy)

	categories, err := s.categories.GetAllPolicyCategories(s.ctx)
	s.NoError(err)
	log.Infof("Categories: %s", categories)

	_, err = s.tested.PutPolicy(s.ctx, postedPolicy)
	s.NoError(err)

	count, err := s.policies.Count(s.ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	s.Equal(1, count)

	count, err = s.categories.Count(s.ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	s.Equal(1, count)
}

func (s *PolicyServicePostgresSuite) TestDryRunFetchesLabels() {
	// Insert test cluster with labels
	cluster := &storage.Cluster{
		Name:   "Test Cluster",
		Labels: map[string]string{"env": "prod"},
	}
	clusterID, err := s.clusters.AddCluster(s.ctx, cluster)
	s.NoError(err)

	// Insert test namespace with labels
	namespace := &storage.NamespaceMetadata{
		Id:          "22222222-2222-2222-2222-222222222222",
		Name:        "test-namespace",
		ClusterId:   clusterID,
		ClusterName: "Test Cluster",
		Labels:      map[string]string{"team": "backend"},
	}
	s.NoError(s.namespaces.AddNamespace(s.ctx, namespace))

	// Create a deployment so dry-run has something to process
	deployment := &storage.Deployment{
		Id:          "33333333-3333-3333-3333-333333333333",
		Name:        "test-deployment",
		ClusterId:   clusterID,
		NamespaceId: "22222222-2222-2222-2222-222222222222",
		Namespace:   "test-namespace",
	}
	deploymentDS, err := deploymentDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)
	s.NoError(deploymentDS.UpsertDeployment(s.ctx, deployment))

	// Create deploy-time policy
	policy := &storage.Policy{
		Name:            "Test Label Policy",
		Severity:        storage.Severity_HIGH_SEVERITY,
		Categories:      []string{"Test Category"},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageTag,
						Values:    []*storage.PolicyValue{{Value: "latest"}},
					},
				},
			},
		},
		PolicyVersion: policyversion.CurrentVersion().String(),
	}

	s.lifecycleManager.EXPECT().UpsertPolicy(gomock.Any()).AnyTimes()
	s.connectionManager.EXPECT().PreparePoliciesAndBroadcast(gomock.Any()).AnyTimes()

	// Run dry-run - this should fetch cluster and namespace labels from datastores
	resp, err := s.tested.DryRunPolicy(s.ctx, policy)
	s.NoError(err)
	s.NotNil(resp)

	// Cleanup: remove test data to avoid affecting other tests
	s.NoError(deploymentDS.RemoveDeployment(s.ctx, clusterID, "33333333-3333-3333-3333-333333333333"))
	s.NoError(s.namespaces.RemoveNamespace(s.ctx, "22222222-2222-2222-2222-222222222222"))
	s.NoError(s.clusters.RemoveCluster(s.ctx, clusterID, nil))
}
