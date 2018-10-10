package datastore

import (
	"errors"
	"fmt"
	"testing"

	indexMocks "github.com/stackrox/rox/central/alert/index/mocks"
	searchMocks "github.com/stackrox/rox/central/alert/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/store/mocks"
	"github.com/stackrox/rox/central/alerttest"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/mock"
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
	storage   *storeMocks.Store
	indexer   *indexMocks.Indexer
	searcher  *searchMocks.Searcher
}

func (s *alertDataStoreTestSuite) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)
	s.dataStore = New(s.storage, s.indexer, s.searcher)
}

func (s *alertDataStoreTestSuite) TestSearchAlerts() {
	s.searcher.On("SearchAlerts", &v1.Query{}).Return([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawAlerts() {
	s.searcher.On("SearchRawAlerts", &v1.Query{}).Return([]*v1.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchListAlerts() {
	s.searcher.On("SearchListAlerts", &v1.Query{}).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.SearchListAlerts(&v1.Query{})

	s.Equal(errFake, err)
	s.Equal(alerttest.NewFakeListAlertSlice(), result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String()).ProtoQuery()
	s.searcher.On("SearchListAlerts", expectedQ).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.CountAlerts()

	s.Equal(errFake, err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	s.storage.On("AddAlert", alerttest.NewFakeAlert()).Return(nil)
	s.indexer.On("AddAlert", alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
	s.indexer.AssertExpectations(s.T())
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenTheIndexerFails() {
	s.storage.On("AddAlert", alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlert() {
	s.storage.On("UpdateAlert", alerttest.NewFakeAlert()).Return(nil)
	s.indexer.On("AddAlert", alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
	s.indexer.AssertExpectations(s.T())
}

func (s *alertDataStoreTestSuite) TestUpdateAlertWhenTheIndexerFails() {
	s.storage.On("UpdateAlert", alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStale() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	s.storage.On("UpdateAlert", mock.Anything).Return(nil)
	s.indexer.On("AddAlert", mock.Anything).Return(nil)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(v1.ViolationState_RESOLVED, fakeAlert.GetState())

	s.storage.AssertExpectations(s.T())
	s.indexer.AssertExpectations(s.T())
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenStorageFails() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(fakeAlert, false, errFake)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenTheAlertWasNotFoundInStorage() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(fakeAlert, false, nil)

	err := s.dataStore.MarkAlertStale(alerttest.FakeAlertID)

	s.EqualError(err, fmt.Sprintf("alert with id '%s' does not exist", alerttest.FakeAlertID))
}
