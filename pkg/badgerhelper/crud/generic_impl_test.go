package generic

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
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

func listAlloc() proto.Message {
	return &storage.ListAlert{}
}

func alertKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Alert).GetId())
}

func converter(msg proto.Message) proto.Message {
	alert := msg.(*storage.Alert)
	return &storage.ListAlert{
		Id:    alert.GetId(),
		State: alert.GetState(),
	}
}

func TestGenericCRUD(t *testing.T) {
	crudWithPartial := new(CRUDTestSuite)
	crudWithPartial.partial = true
	suite.Run(t, crudWithPartial)

	suite.Run(t, new(CRUDTestSuite))
}

type CRUDTestSuite struct {
	partial bool
	suite.Suite

	dir string
	db  *badger.DB

	crud Crud
}

func (s *CRUDTestSuite) SetupTest() {
	var err error
	s.db, s.dir, err = badgerhelper.NewTemp("generic")
	if err != nil {
		s.FailNowf("failed to create DB: %+v", err.Error())
	}
	if s.partial {
		s.crud = NewCRUDWithPartial(s.db, []byte("bucket"), alertKeyFunc, alloc, []byte("list_bucket"), listAlloc, converter)
	} else {
		s.crud = NewCRUD(s.db, []byte("bucket"), alertKeyFunc, alloc)
	}
}

func (s *CRUDTestSuite) TearDownTest() {
	_ = s.db.Close()
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
	_, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.False(exists)

	s.NoError(s.crud.Upsert(alert1))

	msg, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(alert1, msg)

	if s.partial {
		listMsg, exists, err := s.crud.ReadPartial(alert1ID)
		s.NoError(err)
		s.True(exists)
		s.Equal(converter(msg), listMsg)
	}
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
	msgs, indices, err := s.crud.ReadBatch([]string{})
	s.NoError(err)
	s.Len(indices, 0)
	s.Len(msgs, 0)

	msgs, indices, err = s.crud.ReadBatch([]string{alert1ID, alert2ID})
	s.NoError(err)
	s.Equal([]int{0, 1}, indices)
	s.Len(msgs, 0)

	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1, alert2}))

	msgs, indices, err = s.crud.ReadBatch([]string{alert1ID, "3", alert2ID})
	s.NoError(err)
	s.Equal([]int{1}, indices)
	s.ElementsMatch(alerts, msgs)

	if s.partial {
		var partialMsgs []proto.Message
		for _, a := range alerts {
			partialMsgs = append(partialMsgs, converter(a))
		}

		msgs, err := s.crud.ReadAllPartial()
		s.NoError(err)
		s.Equal([]int{1}, indices)
		s.ElementsMatch(partialMsgs, msgs)
	}
}

func (s *CRUDTestSuite) TestReadAll() {
	msgs, err := s.crud.ReadAll()
	s.NoError(err)
	s.Len(msgs, 0)

	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1, alert2}))

	msgs, err = s.crud.ReadAll()
	s.NoError(err)
	s.ElementsMatch(alerts, msgs)

	if s.partial {
		var partialMsgs []proto.Message
		for _, a := range alerts {
			partialMsgs = append(partialMsgs, converter(a))
		}

		msgs, err := s.crud.ReadAllPartial()
		s.NoError(err)
		s.ElementsMatch(partialMsgs, msgs)
	}
}

func (s *CRUDTestSuite) TestUpdate() {
	s.Error(s.crud.Update(alert1))

	s.NoError(s.crud.Upsert(alert1))

	localAlert := proto.Clone(alert1).(*storage.Alert)
	localAlert.State = storage.ViolationState_RESOLVED

	val := s.crud.GetTxnCount()
	s.NoError(s.crud.Update(localAlert))
	s.Equal(val+1, s.crud.GetTxnCount())

	msg, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(localAlert, msg)

	if s.partial {
		msg, exists, err := s.crud.ReadPartial(alert1ID)
		s.NoError(err)
		s.True(exists)
		s.Equal(converter(localAlert), msg)
	}
}
func (s *CRUDTestSuite) TestUpdateMany() {
	s.Error(s.crud.UpdateBatch([]proto.Message{alert1}))
	s.NoError(s.crud.Upsert(alert1))

	s.Error(s.crud.UpdateBatch([]proto.Message{alert1, alert2}))

	s.NoError(s.crud.Upsert(alert2))

	localAlert1 := proto.Clone(alert1).(*storage.Alert)
	localAlert1.State = storage.ViolationState_RESOLVED

	localAlert2 := proto.Clone(alert2).(*storage.Alert)
	localAlert2.State = storage.ViolationState_RESOLVED

	txNum := s.crud.GetTxnCount()
	s.NoError(s.crud.UpdateBatch([]proto.Message{localAlert1, localAlert2}))
	s.Equal(txNum+1, s.crud.GetTxnCount())

	msgs, err := s.crud.ReadAll()
	s.NoError(err)

	localAlerts := []*storage.Alert{localAlert1, localAlert2}
	s.ElementsMatch(localAlerts, msgs)

	if s.partial {
		var partialMsgs []proto.Message
		for _, a := range localAlerts {
			partialMsgs = append(partialMsgs, converter(a))
		}

		msgs, err := s.crud.ReadAllPartial()
		s.NoError(err)
		s.ElementsMatch(partialMsgs, msgs)
	}
}

func (s *CRUDTestSuite) TestUpsert() {
	s.NoError(s.crud.Upsert(alert1))

	localAlert := proto.Clone(alert1).(*storage.Alert)
	localAlert.State = storage.ViolationState_RESOLVED

	txNum := s.crud.GetTxnCount()
	s.NoError(s.crud.Upsert(localAlert))
	s.Equal(txNum+1, s.crud.GetTxnCount())

	msg, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.True(exists)
	s.Equal(localAlert, msg)
}

func (s *CRUDTestSuite) TestUpsertMany() {
	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1}))
	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1, alert2}))

	localAlert1 := proto.Clone(alert1).(*storage.Alert)
	localAlert1.State = storage.ViolationState_RESOLVED

	localAlert2 := proto.Clone(alert2).(*storage.Alert)
	localAlert2.State = storage.ViolationState_RESOLVED

	txNum := s.crud.GetTxnCount()
	s.NoError(s.crud.UpsertBatch([]proto.Message{localAlert1, localAlert2}))
	s.Equal(txNum+1, s.crud.GetTxnCount())

	msgs, err := s.crud.ReadAll()
	s.NoError(err)
	s.ElementsMatch([]*storage.Alert{localAlert1, localAlert2}, msgs)
}

func (s *CRUDTestSuite) TestDelete() {
	txNum := s.crud.GetTxnCount()
	s.NoError(s.crud.Upsert(alert1))
	s.Equal(txNum+1, s.crud.GetTxnCount())
	s.NoError(s.crud.Delete(alert1ID))
	s.Equal(txNum+2, s.crud.GetTxnCount())

	_, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.False(exists)
}

func (s *CRUDTestSuite) TestDeleteMany() {
	txNum := s.crud.GetTxnCount()
	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1, alert2}))
	s.Equal(txNum+1, s.crud.GetTxnCount())
	s.NoError(s.crud.DeleteBatch([]string{alert1ID, alert2ID}))
	s.Equal(txNum+2, s.crud.GetTxnCount())

	_, exists, err := s.crud.Read(alert1ID)
	s.NoError(err)
	s.False(exists)

	_, exists, err = s.crud.Read(alert2ID)
	s.NoError(err)
	s.False(exists)

	if s.partial {
		msgs, err := s.crud.ReadAllPartial()
		s.NoError(err)
		s.Len(msgs, 0)
	}
}

func (s *CRUDTestSuite) TestGetIDs() {
	s.NoError(s.crud.UpsertBatch([]proto.Message{alert1, alert2}))

	ids, err := s.crud.GetKeys()
	s.NoError(err)
	s.Equal([]string{alert1ID, alert2ID}, ids)

	s.NoError(s.crud.DeleteBatch([]string{alert1ID}))
	ids, err = s.crud.GetKeys()
	s.NoError(err)
	s.Equal([]string{alert2ID}, ids)

	s.NoError(s.crud.Delete(alert2ID))

	ids, err = s.crud.GetKeys()
	s.NoError(err)
	s.Len(ids, 0)
}
