package datastore

import (
	"context"
	"errors"
	"testing"

	searchMocks "github.com/stackrox/rox/central/alert/datastore/internal/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/datastore/internal/store/mocks"
	_ "github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/alerttest"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	t.Parallel()
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

	var err error
	s.dataStore, err = New(s.storage, s.searcher)
	s.Require().NoError(err)
}

func (s *alertDataStoreTestSuite) TestSearchAlerts() {
	s.searcher.EXPECT().SearchAlerts(s.hasReadCtx, &v1.Query{}).Return([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchAlerts(s.hasReadCtx, &v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawAlerts() {
	s.searcher.EXPECT().SearchRawAlerts(s.hasReadCtx, &v1.Query{}).Return([]*storage.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(s.hasReadCtx, &v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*storage.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchListAlerts() {
	s.searcher.EXPECT().SearchListAlerts(s.hasReadCtx, &v1.Query{}).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.SearchListAlerts(s.hasReadCtx, &v1.Query{})

	s.Equal(errFake, err)
	s.Equal(alerttest.NewFakeListAlertSlice(), result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Success() {
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Count(s.hasReadCtx, expectedQ).Return(1, nil)

	result, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.NoError(err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Error() {
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Count(s.hasReadCtx, expectedQ).Return(0, errFake)

	_, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Alert{fakeAlert}).Return(errFake)

	err := s.dataStore.UpsertAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenTheIndexerFails() {
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
	var err error
	s.dataStore, err = New(s.storage, s.searcher)
	s.NoError(err)
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
