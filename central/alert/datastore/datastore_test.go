package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/alert/convert"
	indexMocks "github.com/stackrox/rox/central/alert/datastore/internal/index/mocks"
	searchMocks "github.com/stackrox/rox/central/alert/datastore/internal/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/alerttest"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
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
	indexer   *indexMocks.MockIndexer
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
			sac.ResourceScopeKeys(resources.Role)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.storage.EXPECT().GetTxnCount().Return(uint64(1), nil)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.indexer.EXPECT().GetTxnCount().Return(uint64(1))
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)

	var err error
	s.dataStore, err = New(s.storage, s.indexer, s.searcher)
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
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Search(s.hasReadCtx, expectedQ).Return([]search.Result{
		{ID: alerttest.FakeAlertID},
	}, nil)

	result, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.NoError(err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Error() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().Search(s.hasReadCtx, expectedQ).Return(nil, errFake)

	_, err := s.dataStore.CountAlerts(s.hasReadCtx)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Return(errFake)

	err := s.dataStore.AddAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenTheIndexerFails() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlert() {
	s.storage.EXPECT().GetAlert(alerttest.NewFakeAlert().Id).Return(nil, true, nil)
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Return(errFake)

	err := s.dataStore.UpdateAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlertWhenTheIndexerFails() {
	s.storage.EXPECT().GetAlert(alerttest.NewFakeAlert().Id).Return(nil, true, nil)
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(s.hasWriteCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStale() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	s.storage.EXPECT().UpdateAlert(gomock.Any()).Return(nil)
	s.indexer.EXPECT().AddListAlert(gomock.Any()).Return(nil)

	err := s.dataStore.MarkAlertStale(s.hasWriteCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(storage.ViolationState_RESOLVED, fakeAlert.GetState())
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenStorageFails() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, errFake)

	err := s.dataStore.MarkAlertStale(s.hasWriteCtx, alerttest.FakeAlertID)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenTheAlertWasNotFoundInStorage() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, nil)

	err := s.dataStore.MarkAlertStale(s.hasWriteCtx, alerttest.FakeAlertID)

	s.EqualError(err, fmt.Sprintf("alert with id '%s' does not exist", alerttest.FakeAlertID))
}

func TestAlertDataStoreWithSAC(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(alertDataStoreWithSACTestSuite))
}

type alertDataStoreWithSACTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore
	indexer   *indexMocks.MockIndexer
	searcher  *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *alertDataStoreWithSACTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)
	var err error
	s.dataStore, err = New(s.storage, s.indexer, s.searcher)
	s.NoError(err)
}

func (s *alertDataStoreWithSACTestSuite) TestAddAlertEnforced() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Times(0)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Times(0)

	err := s.dataStore.AddAlert(s.hasReadCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreWithSACTestSuite) TestUpdateAlertEnforced() {
	s.storage.EXPECT().GetAlert(alerttest.NewFakeAlert().Id).Times(0)
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Times(0)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Times(0)

	err := s.dataStore.UpdateAlert(s.hasReadCtx, alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreWithSACTestSuite) TestMarkAlertStaleEnforced() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Times(0)
	s.storage.EXPECT().UpdateAlert(gomock.Any()).Times(0)
	s.indexer.EXPECT().AddListAlert(gomock.Any()).Times(0)

	err := s.dataStore.MarkAlertStale(s.hasReadCtx, alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(storage.ViolationState_RESOLVED, fakeAlert.GetState())
}
