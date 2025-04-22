package datastore

import (
	"context"
	"errors"
	"testing"

	searchMocks "github.com/stackrox/rox/central/alert/datastore/internal/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/datastore/internal/store/mocks"
	_ "github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/alerttest"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake = errors.New("fake error")
)

func TestAlertDataStore(t *testing.T) {
	suite.Run(t, new(alertDataStoreTestSuite))
}

type alertDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *alertDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)

	s.dataStore = New(s.storage, s.searcher, platformmatcher.Singleton())
}

func (s *alertDataStoreTestSuite) TestSearchAlerts() {
	s.searcher.EXPECT().SearchAlerts(s.hasReadCtx, &v1.Query{}).Return([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchAlerts(s.hasReadCtx, &v1.Query{})

	s.Equal(errFake, err)
	protoassert.SlicesEqual(s.T(), []*v1.SearchResult{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawAlerts() {
	s.searcher.EXPECT().SearchRawAlerts(s.hasReadCtx, &v1.Query{}, true).Return([]*storage.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(s.hasReadCtx, &v1.Query{}, true)

	s.Equal(errFake, err)
	protoassert.SlicesEqual(s.T(), []*storage.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearch() {
	s.searcher.EXPECT().Search(s.hasReadCtx, &v1.Query{}, true).Return([]search.Result{{ID: alerttest.FakeAlertID}}, nil)

	result, err := s.dataStore.Search(s.hasReadCtx, &v1.Query{}, true)
	s.NoError(err)
	s.ElementsMatch([]search.Result{{ID: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchResolved() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.searcher.EXPECT().Search(s.hasReadCtx, &v1.Query{}, false).Return([]search.Result{{ID: alerttest.FakeAlertID}}, nil)

	result, err := s.dataStore.Search(s.hasReadCtx, &v1.Query{}, false)
	s.NoError(err)
	s.ElementsMatch([]search.Result{{ID: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawResolvedAlerts() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.searcher.EXPECT().SearchRawAlerts(s.hasReadCtx, &v1.Query{}, false).Return([]*storage.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(s.hasReadCtx, &v1.Query{}, false)

	s.Equal(errFake, err)
	protoassert.SlicesEqual(s.T(), []*storage.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchResolvedListAlerts() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.searcher.EXPECT().SearchListAlerts(s.hasReadCtx, &v1.Query{}, false).Return([]*storage.ListAlert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchListAlerts(s.hasReadCtx, &v1.Query{}, false)
	s.Equal(errFake, err)
	protoassert.SlicesEqual(s.T(), []*storage.ListAlert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchListAlerts() {
	s.searcher.EXPECT().SearchListAlerts(s.hasReadCtx, &v1.Query{}, true).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.SearchListAlerts(s.hasReadCtx, &v1.Query{}, true)

	s.Equal(errFake, err)
	protoassert.SlicesEqual(s.T(), alerttest.NewFakeListAlertSlice(), result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Success() {
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Count(s.hasReadCtx, expectedQ, true).Return(1, nil)

	result, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.NoError(err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestCountAlertsResolved_Success() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.searcher.EXPECT().Count(s.hasReadCtx, &v1.Query{}, false).Return(1, nil)

	result, err := s.dataStore.Count(s.hasReadCtx, &v1.Query{}, false)
	s.NoError(err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Error() {
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Count(s.hasReadCtx, expectedQ, true).Return(0, errFake)

	_, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{fakeAlert}).Return(errFake)

	err := s.dataStore.UpsertAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenUpsertFails() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{fakeAlert}).Return(errFake)

	err := s.dataStore.UpsertAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleBatch() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(storage.ViolationState_RESOLVED, fakeAlert.GetState())
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleBatchWhenStorageFails() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, []int{0}, errFake)

	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, alerttest.FakeAlertID)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleBatchWhenTheAlertWasNotFoundInStorage() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{fakeAlert.GetId()}).Return(nil, []int{0}, nil)

	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, fakeAlert.GetId())

	s.NoError(err)
}

func (s *alertDataStoreTestSuite) TestKeyIndexing() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{fakeAlert.GetId()}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil)
	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasWriteCtx, fakeAlert.GetId())
	s.NoError(err)
}

func (s *alertDataStoreTestSuite) TestGetByQuery() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetByQueryFn(s.hasReadCtx, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, query *v1.Query, fn func(*storage.Alert) error) error {
			return fn(fakeAlert)
		}).Times(1)
	err := s.dataStore.WalkByQuery(s.hasWriteCtx, search.EmptyQuery(), func(a *storage.Alert) error {
		protoassert.Equal(s.T(), fakeAlert, a)
		return nil
	})
	s.Require().NoError(err)

	s.storage.EXPECT().GetByQueryFn(s.hasWriteCtx, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, query *v1.Query, fn func(*storage.Alert) error) error {
			return fn(fakeAlert)
		}).Times(1)
	err = s.dataStore.WalkByQuery(s.hasWriteCtx, search.EmptyQuery(), func(a *storage.Alert) error {
		protoassert.Equal(s.T(), fakeAlert, a)
		return nil
	})
	s.Require().NoError(err)
}

func (s *alertDataStoreTestSuite) TestUpsert_PlatformComponentAndEntityTypeAssignment() {
	s.T().Setenv(features.PlatformComponents.EnvVar(), "true")
	if !features.PlatformComponents.Enabled() {
		s.T().Skip("Skip test when ROX_PLATFORM_COMPONENTS disabled")
		s.T().SkipNow()
	}
	// Case: Resource alert
	alert := &storage.Alert{
		Id:     "id",
		Entity: &storage.Alert_Resource_{Resource: &storage.Alert_Resource{}},
	}
	expectedAlert := &storage.Alert{
		Id:                "id",
		Entity:            &storage.Alert_Resource_{Resource: &storage.Alert_Resource{}},
		EntityType:        storage.Alert_RESOURCE,
		PlatformComponent: false,
	}

	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{expectedAlert}).Return(nil).Times(1)
	err := s.dataStore.UpsertAlert(s.hasWriteCtx, alert)
	s.Require().NoError(err)

	// Case: Container image alert
	alert = &storage.Alert{
		Id:     "id",
		Entity: &storage.Alert_Image{Image: &storage.ContainerImage{}},
	}
	expectedAlert = &storage.Alert{
		Id:                "id",
		Entity:            &storage.Alert_Image{Image: &storage.ContainerImage{}},
		EntityType:        storage.Alert_CONTAINER_IMAGE,
		PlatformComponent: false,
	}

	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{expectedAlert}).Return(nil).Times(1)
	err = s.dataStore.UpsertAlert(s.hasWriteCtx, alert)
	s.Require().NoError(err)

	// Case: Deployment alert not matching platform rules
	alert = &storage.Alert{
		Id:     "id",
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{Namespace: "my-namespace"}},
	}
	expectedAlert = &storage.Alert{
		Id:                "id",
		Entity:            &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{Namespace: "my-namespace"}},
		EntityType:        storage.Alert_DEPLOYMENT,
		PlatformComponent: false,
	}

	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{expectedAlert}).Return(nil).Times(1)
	err = s.dataStore.UpsertAlert(s.hasWriteCtx, alert)
	s.Require().NoError(err)

	// Case: Deployment alert matching platform rules
	alert = &storage.Alert{
		Id:     "id",
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{Namespace: "openshift-123"}},
	}
	expectedAlert = &storage.Alert{
		Id:                "id",
		Entity:            &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{Namespace: "openshift-123"}},
		EntityType:        storage.Alert_DEPLOYMENT,
		PlatformComponent: true,
	}

	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{expectedAlert}).Return(nil).Times(1)
	err = s.dataStore.UpsertAlert(s.hasWriteCtx, alert)
	s.Require().NoError(err)
}

func TestAlertDataStoreWithSAC(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(alertDataStoreWithSACTestSuite))
}

type alertDataStoreWithSACTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *alertDataStoreWithSACTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)
	s.dataStore = New(s.storage, s.searcher, platformmatcher.Singleton())
}

func (s *alertDataStoreWithSACTestSuite) TestAddAlertEnforced() {
	s.storage.EXPECT().Upsert(gomock.Any(), alerttest.NewFakeAlert()).Times(0)
	err := s.dataStore.UpsertAlert(s.hasReadCtx, alerttest.NewFakeAlert())

	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *alertDataStoreWithSACTestSuite) TestMarkAlertStaleBatchEnforced() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetMany(gomock.Any(), []string{alerttest.FakeAlertID}).Return([]*storage.Alert{fakeAlert}, nil, nil)
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Times(0)

	_, err := s.dataStore.MarkAlertsResolvedBatch(s.hasReadCtx, alerttest.FakeAlertID)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	s.Equal(storage.ViolationState_ACTIVE, fakeAlert.GetState())
}
