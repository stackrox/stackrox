// +build rocksdb

package generic

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

var (
	alert1   = fixtures.GetAlertWithID("1")
	alert1ID = alert1.GetId()

	alert2   = fixtures.GetAlertWithID("2")
	alert2ID = alert2.GetId()

	alerts = []*storage.Alert{alert1, alert2}
)

func alloc() proto.Message {
	return &storage.Alert{}
}

func alertKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Alert).GetId())
}

func TestGenericCRUD(t *testing.T) {
	suite.Run(t, new(CRUDTestSuite))
}

type CRUDTestSuite struct {
	suite.Suite

	dir string
	db  *gorocksdb.DB

	crud db.Crud
}

func (s *CRUDTestSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "")
	s.NoError(err)

	s.dir = dir

	openOpts := gorocksdb.NewDefaultOptions()
	openOpts.SetCreateIfMissing(true)
	s.db, err = gorocksdb.OpenDb(openOpts, dir)
	s.NoError(err)

	s.crud = NewCRUD(s.db, []byte("bucket"), alertKeyFunc, alloc)
}

func (s *CRUDTestSuite) TearDownTest() {
	s.db.Close()
	_ = os.RemoveAll(s.dir)
}

func (s *CRUDTestSuite) CountTest() {
	count, err := s.crud.Count()
	s.NoError(err)
	s.Equal(0, count)

	s.NoError(s.crud.Upsert(alert1))

	count, err = s.crud.Count()
	s.NoError(err)
	s.Equal(1, count)
}

func (s *CRUDTestSuite) TestRead() {
	_, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.False(exists)

	s.NoError(s.crud.Upsert(alert1))

	msg, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(alert1, msg)
}

func (s *CRUDTestSuite) TestExists() {
	exists, err := s.crud.Exists(alert1ID)
	s.NoError(err)
	s.False(exists)

	s.NoError(s.crud.Upsert(alert1))

	exists, err = s.crud.Exists(alert1ID)
	s.NoError(err)
	s.True(exists)
}

func (s *CRUDTestSuite) TestReadMany() {
	msgs, indices, err := s.crud.GetMany([]string{})
	s.NoError(err)
	s.Len(indices, 0)
	s.Len(msgs, 0)

	msgs, indices, err = s.crud.GetMany([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Equal([]int{0, 1}, indices)
	s.Len(msgs, 0)

	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))

	msgs, indices, err = s.crud.GetMany([]string{alert1ID, "3", alert2ID})
	s.NoError(err)
	s.Equal([]int{1}, indices)
	s.ElementsMatch(alerts, msgs)
}

func (s *CRUDTestSuite) TestUpsert() {
	s.NoError(s.crud.Upsert(alert1))

	localAlert := proto.Clone(alert1).(*storage.Alert)
	localAlert.State = storage.ViolationState_RESOLVED

	s.NoError(s.crud.Upsert(localAlert))

	msg, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(localAlert, msg)
}

func (s *CRUDTestSuite) TestUpsertMany() {
	s.NoError(s.crud.UpsertMany([]proto.Message{alert1}))
	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))

	localAlert1 := proto.Clone(alert1).(*storage.Alert)
	localAlert1.State = storage.ViolationState_RESOLVED

	localAlert2 := proto.Clone(alert2).(*storage.Alert)
	localAlert2.State = storage.ViolationState_RESOLVED

	s.NoError(s.crud.UpsertMany([]proto.Message{localAlert1, localAlert2}))
}

func (s *CRUDTestSuite) TestDelete() {
	s.NoError(s.crud.Upsert(alert1))
	s.NoError(s.crud.Delete(alert1ID))

	_, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.False(exists)
}

func (s *CRUDTestSuite) TestDeleteMany() {
	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))
	s.NoError(s.crud.DeleteMany([]string{alert1ID, alert2ID}))

	_, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.False(exists)

	_, exists, err = s.crud.Get(alert2ID)
	s.NoError(err)
	s.False(exists)
}

func (s *CRUDTestSuite) TestGetIDs() {
	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))

	ids, err := s.crud.GetKeys()
	s.NoError(err)
	s.Equal([]string{alert1ID, alert2ID}, ids)

	s.NoError(s.crud.DeleteMany([]string{alert1ID}))
	ids, err = s.crud.GetKeys()
	s.NoError(err)
	s.Equal([]string{alert2ID}, ids)

	s.NoError(s.crud.Delete(alert2ID))

	ids, err = s.crud.GetKeys()
	s.NoError(err)
	s.Len(ids, 0)
}

func (s *CRUDTestSuite) verifyKeyWasMarkedForIndexingAndReset(expectedKeys ...string) {
	keys, err := s.crud.GetKeysToIndex()
	s.NoError(err)
	s.ElementsMatch(expectedKeys, keys)

	// Verify that they can be removed as well
	s.NoError(s.crud.AckKeysIndexed(keys...))
	keys, err = s.crud.GetKeysToIndex()
	s.NoError(err)
	s.ElementsMatch(nil, keys)
}

func (s *CRUDTestSuite) TestKeysToIndex() {
	keys, err := s.crud.GetKeysToIndex()
	s.NoError(err)
	s.Empty(keys)

	s.NoError(s.crud.Upsert(alert1))
	s.verifyKeyWasMarkedForIndexingAndReset(alert1ID)

	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))
	s.verifyKeyWasMarkedForIndexingAndReset(alert1ID, alert2ID)

	s.NoError(s.crud.Delete(alert1ID))
	s.verifyKeyWasMarkedForIndexingAndReset(alert1ID)

	s.NoError(s.crud.DeleteMany([]string{alert1ID, alert2ID}))
	s.verifyKeyWasMarkedForIndexingAndReset(alert1ID, alert2ID)
}
