//go:build sql_integration

package service

import (
	"context"
	"testing"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	notifierDatastore "github.com/stackrox/rox/central/notifier/datastore"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/policy/search"
	policyStore "github.com/stackrox/rox/central/policy/store"
	policyCategoryDatastore "github.com/stackrox/rox/central/policycategory/datastore"
	categorySearch "github.com/stackrox/rox/central/policycategory/search"
	categoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	edgeDataStore "github.com/stackrox/rox/central/policycategoryedge/datastore"
	edgeSearch "github.com/stackrox/rox/central/policycategoryedge/search"
	edgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/postgres"
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
	db                postgres.DB
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

	pStore := policyStore.New(s.db)

	notifierDS := notifierDatastore.GetTestPostgresDataStore(s.T(), s.db)

	categoryStorage := categoryPostgres.New(s.db)
	categorySearcher := categorySearch.New(categoryStorage)

	edgeStorage := edgePostgres.New(s.db)
	edgeSearcher := edgeSearch.New(edgeStorage)

	edgeDatastore := edgeDataStore.New(edgeStorage, edgeSearcher)

	s.categories = policyCategoryDatastore.New(categoryStorage, categorySearcher, edgeDatastore)

	s.policies = policyDatastore.New(pStore, search.New(pStore), s.clusters, notifierDS, s.categories)

	var err error
	s.clusters, err = clusterDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	s.mitreVectorStore = mitreDataStore.NewMitreAttackStore()

	s.mockCtrl = gomock.NewController(s.T())

	s.lifecycleManager = lifecycleMocks.NewMockManager(s.mockCtrl)

	s.connectionManager = connectionMocks.NewMockManager(s.mockCtrl)

	s.tested = New(s.policies, s.clusters, nil, nil, notifierDS, s.mitreVectorStore, nil, s.lifecycleManager, nil, nil, s.connectionManager)
}

func (s *PolicyServicePostgresSuite) TearDownSuite() {
	s.db.Close()
}

func (s *PolicyServicePostgresSuite) TestPostPolicy() {
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
}
