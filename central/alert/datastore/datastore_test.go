package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/alert/convert"
	indexMocks "github.com/stackrox/rox/central/alert/index/mocks"
	searchMocks "github.com/stackrox/rox/central/alert/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/store/mocks"
	"github.com/stackrox/rox/central/alerttest"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

	result, err := s.dataStore.SearchAlerts(context.TODO(), &v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*v1.SearchResult{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchRawAlerts() {
	s.searcher.EXPECT().SearchRawAlerts(&v1.Query{}).Return([]*storage.Alert{{Id: alerttest.FakeAlertID}}, errFake)

	result, err := s.dataStore.SearchRawAlerts(context.TODO(), &v1.Query{})

	s.Equal(errFake, err)
	s.Equal([]*storage.Alert{{Id: alerttest.FakeAlertID}}, result)
}

func (s *alertDataStoreTestSuite) TestSearchListAlerts() {
	s.searcher.EXPECT().SearchListAlerts(&v1.Query{}).Return(alerttest.NewFakeListAlertSlice(), errFake)

	result, err := s.dataStore.SearchListAlerts(context.TODO(), &v1.Query{})

	s.Equal(errFake, err)
	s.Equal(alerttest.NewFakeListAlertSlice(), result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Success() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.indexer.EXPECT().Search(expectedQ).Return([]search.Result{
		{ID: alerttest.FakeAlertID},
	}, nil)

	result, err := s.dataStore.CountAlerts(context.TODO())

	s.NoError(err)
	s.Equal(1, result)
}

func (s *alertDataStoreTestSuite) TestCountAlerts_Error() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	s.indexer.EXPECT().Search(expectedQ).Return(nil, errFake)

	_, err := s.dataStore.CountAlerts(context.TODO())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlert() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Return(errFake)

	err := s.dataStore.AddAlert(context.TODO(), alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestAddAlertWhenTheIndexerFails() {
	s.storage.EXPECT().AddAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.AddAlert(context.TODO(), alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlert() {
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(nil)
	s.indexer.EXPECT().AddListAlert(convert.AlertToListAlert(alerttest.NewFakeAlert())).Return(errFake)

	err := s.dataStore.UpdateAlert(context.TODO(), alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestUpdateAlertWhenTheIndexerFails() {
	s.storage.EXPECT().UpdateAlert(alerttest.NewFakeAlert()).Return(errFake)

	err := s.dataStore.UpdateAlert(context.TODO(), alerttest.NewFakeAlert())

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStale() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	s.storage.EXPECT().UpdateAlert(gomock.Any()).Return(nil)
	s.indexer.EXPECT().AddListAlert(gomock.Any()).Return(nil)

	err := s.dataStore.MarkAlertStale(context.TODO(), alerttest.FakeAlertID)
	s.NoError(err)

	s.Equal(storage.ViolationState_RESOLVED, fakeAlert.GetState())
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenStorageFails() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, errFake)

	err := s.dataStore.MarkAlertStale(context.TODO(), alerttest.FakeAlertID)

	s.Equal(errFake, err)
}

func (s *alertDataStoreTestSuite) TestMarkAlertStaleWhenTheAlertWasNotFoundInStorage() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.EXPECT().GetAlert(alerttest.FakeAlertID).Return(fakeAlert, false, nil)

	err := s.dataStore.MarkAlertStale(context.TODO(), alerttest.FakeAlertID)

	s.EqualError(err, fmt.Sprintf("alert with id '%s' does not exist", alerttest.FakeAlertID))
}
