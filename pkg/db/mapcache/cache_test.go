package mapcache

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db/mocks"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	alert1   = fixtures.GetAlertWithID("1")
	alert1ID = alert1.GetId()

	alert2   = fixtures.GetAlertWithID("2")
	alert2ID = alert2.GetId()
)

func alertKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Alert).GetId())
}

func TestCachedCRUD(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

type CacheTestSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	underlyingDB *mocks.MockCrud
	cache        *cacheImpl
}

func (s *CacheTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.underlyingDB = mocks.NewMockCrud(s.ctrl)

	s.underlyingDB.EXPECT().WalkAllWithID(gomock.Any()).Return(nil)
	cache, err := NewMapCache(s.underlyingDB, alertKeyFunc)
	s.NoError(err)
	s.cache = cache.(*cacheImpl)
}

func (s *CacheTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CacheTestSuite) TestSingleOperations() {
	_, exists, err := s.cache.Get(alert1ID)
	s.NoError(err)
	s.False(exists)

	// Upsert into cache
	s.underlyingDB.EXPECT().Upsert(alert1)
	s.NoError(s.cache.Upsert(alert1))

	// Get should be from cache
	msg, exists, err := s.cache.Get(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(alert1, msg)

	// Upsert again with a new value and make sure cache reflects the new value
	cloned1 := alert1.Clone()
	cloned1.Policy.EnforcementActions = []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT}

	s.underlyingDB.EXPECT().Upsert(cloned1)
	s.NoError(s.cache.Upsert(cloned1))

	msg, exists, err = s.cache.Get(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(cloned1, msg)

	s.underlyingDB.EXPECT().Delete(alert1ID)
	s.NoError(s.cache.Delete(alert1ID))

	_, exists, err = s.cache.Get(alert1ID)
	s.NoError(err)
	s.False(exists)
}

func (s *CacheTestSuite) TestBulkOperations() {
	_, missingIndices, err := s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Equal([]int{0, 1}, missingIndices)

	// Upsert into cache
	s.underlyingDB.EXPECT().Upsert(alert1)
	s.NoError(s.cache.Upsert(alert1))

	msgs, missingIndices, err := s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Equal([]int{1}, missingIndices)
	s.Equal([]proto.Message{alert1}, msgs)

	s.underlyingDB.EXPECT().Upsert(alert2)
	s.NoError(s.cache.Upsert(alert2))

	msgs, missingIndices, err = s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Nil(missingIndices)
	s.Equal([]proto.Message{alert1, alert2}, msgs)

	s.underlyingDB.EXPECT().DeleteMany([]string{alert1ID})
	s.NoError(s.cache.DeleteMany([]string{alert1ID}))

	msgs, missingIndices, err = s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Equal([]int{0}, missingIndices)
	s.Equal([]proto.Message{alert2}, msgs)

	s.underlyingDB.EXPECT().UpsertWithID(alert1ID, alert1)
	s.NoError(s.cache.UpsertWithID(alert1ID, alert1))

	msgs, missingIndices, err = s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Nil(missingIndices)
	s.Equal([]proto.Message{alert1, alert2}, msgs)

	s.underlyingDB.EXPECT().UpsertMany([]proto.Message{alert1, alert2})
	s.NoError(s.cache.UpsertMany([]proto.Message{alert1, alert2}))

	msgs, missingIndices, err = s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Nil(missingIndices)
	s.Equal([]proto.Message{alert1, alert2}, msgs)

	cloned1 := alert1.Clone()
	cloned1.Policy.EnforcementActions = []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT}

	s.underlyingDB.EXPECT().UpsertManyWithIDs([]string{alert1ID, alert2ID}, []proto.Message{cloned1, alert2})
	s.NoError(s.cache.UpsertManyWithIDs([]string{alert1ID, alert2ID}, []proto.Message{cloned1, alert2}))

	msgs, missingIndices, err = s.cache.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Nil(missingIndices)
	s.Equal([]proto.Message{cloned1, alert2}, msgs)
}
