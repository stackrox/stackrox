package generic

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/suite"
)

var (
	alert1   = fixtures.GetAlertWithID("1")
	alert1ID = alert1.GetId()

	alert2   = fixtures.GetAlertWithID("2")
	alert2ID = alert2.GetId()

	alert3   = fixtures.GetAlertWithID("3")
	alert3ID = alert3.GetId()

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
	db  *rocksdb.RocksDB

	crud db.Crud
}

func (s *CRUDTestSuite) SetupTest() {
	var err error
	dir := s.T().TempDir()

	s.dir = dir

	s.db, err = rocksdb.New(dir)
	s.NoError(err)
	s.crud = NewCRUD(s.db, []byte("bucket"), alertKeyFunc, alloc, true)
}

func (s *CRUDTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *CRUDTestSuite) TestWalkAllWithID() {
	var ids []string
	var alerts []*storage.Alert
	do := func(id []byte, msg proto.Message) error {
		if bytes.Equal(id, []byte(alert3ID)) {
			return nil
		}
		ids = append(ids, string(id))
		alerts = append(alerts, msg.(*storage.Alert))
		return nil
	}

	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2, alert3}))

	err := s.crud.WalkAllWithID(do)
	s.NoError(err)
	s.ElementsMatch([]string{alert1ID, alert2ID}, ids)
	s.ElementsMatch([]*storage.Alert{alert1, alert2}, alerts)
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

	localAlert := alert1.Clone()
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

	localAlert1 := alert1.Clone()
	localAlert1.State = storage.ViolationState_RESOLVED

	localAlert2 := alert2.Clone()
	localAlert2.State = storage.ViolationState_RESOLVED

	s.NoError(s.crud.UpsertMany([]proto.Message{localAlert1, localAlert2}))
}

func (s *CRUDTestSuite) TestUpsertWithID() {
	s.NoError(s.crud.UpsertWithID(alert1ID, alert1))

	localAlert := alert1.Clone()
	localAlert.State = storage.ViolationState_RESOLVED

	s.NoError(s.crud.UpsertWithID(alert1ID, localAlert))

	msg, exists, err := s.crud.Get(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(localAlert, msg)
}

func (s *CRUDTestSuite) TestUpsertManyWithIDs() {
	s.NoError(s.crud.UpsertManyWithIDs([]string{alert1ID}, []proto.Message{alert1}))
	s.NoError(s.crud.UpsertManyWithIDs([]string{alert1ID, alert2ID}, []proto.Message{alert1, alert2}))

	localAlert1 := alert1.Clone()
	localAlert1.State = storage.ViolationState_RESOLVED

	localAlert2 := alert2.Clone()
	localAlert2.State = storage.ViolationState_RESOLVED

	s.NoError(s.crud.UpsertManyWithIDs([]string{alert1ID, alert2ID}, []proto.Message{localAlert1, localAlert2}))
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
