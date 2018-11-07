package datastore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	indexMocks "github.com/stackrox/rox/central/alert/index/mocks"
	searchMocks "github.com/stackrox/rox/central/alert/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/store/mocks"
	"github.com/stackrox/rox/central/alerttest"
	"github.com/stackrox/rox/generated/api/v1"
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

	dataStore DataStore
	storage   *storeMocks.MockStore
	indexer   *indexMocks.MockIndexer
	searcher  *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *alertDataStoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)
	s.dataStore = New(s.storage, s.indexer, s.searcher)
}

func (s *alertDataStoreTestSuite) TestSearchAlerts() {
	s.searcher.EXPECT().SearchAlerts(&v1.Query{}).Return([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawAlerts() {
	s.searcher.EXPECT().SearchRawAlerts(&v1.Query{}).Return([]*v1.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchListAlerts() {
	s.searcher.EXPECT().SearchListAlerts(&v1.Query{}).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.SearchListAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal(alerttest.NewFakeListAlertSlice(), result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.EXPECT().SearchListAlerts(expectedQ).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.CountAlerts()

	s.Equal(errFake, err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenTheIndexerFails() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlert() {
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlertWhenTheIndexerFails() {
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStale() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	s.storage.EXPECT().UpdateAlert(gomock.Any()).Return(nil)
	s.indexer.EXPECT().AddAlert(gomock.Any()).Return(nil)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(v1.ViolationState_RESOLVED, fakeAlert.GetState())
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenStorageFails() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, errFake)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenTheAlertWasNotFoundInStorage() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, nil)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)

	s.EqualError(err, fmt.Sprintf("alert with id '%s' does not exist", alerttest.FakeAlertID))
}
