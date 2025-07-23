//go:build sql_integration

package service

import (
	"context"
	"testing"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
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
	mitreVectorStore  mitreDataStore.AttackReadOnlyDataStore
	lifecycleManager  *lifecycleMocks.MockManager
	connectionManager *connectionMocks.MockManager
	tested            Service
}

func (s *PolicyServicePostgresSuite) SetupSuite() {

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

	s.mitreVectorStore = mitreDataStore.NewMitreAttackStore()

	s.mockCtrl = gomock.NewController(s.T())

	s.lifecycleManager = lifecycleMocks.NewMockManager(s.mockCtrl)

	s.connectionManager = connectionMocks.NewMockManager(s.mockCtrl)

	s.tested = New(s.policies, s.clusters, nil, nil, notifierDS, s.mitreVectorStore, nil, s.lifecycleManager, nil, nil, s.connectionManager)
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
