package proto

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestMessageCrud(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MessageCrudTestSuite))
}

type MessageCrudTestSuite struct {
	suite.Suite

	db *bolt.DB

	crud MessageCrud
}

func (s *MessageCrudTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	testBucket := "testBucket"
	bolthelper.RegisterBucketOrPanic(db, testBucket)

	s.db = db

	// Function that provides the key for a given instance.
	keyFunc := func(msg proto.Message) []byte {
		return []byte(msg.(*storage.Alert).GetId())
	}
	// Function that provide a new empty instance when wanted.
	allocFunc := func() proto.Message {
		return &storage.Alert{}
	}
	s.crud = NewMessageCrud(db, "testBucket", keyFunc, allocFunc)
}

func (s *MessageCrudTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
		os.Remove(s.db.Path())
	}
}

func (s *MessageCrudTestSuite) TestCreate() {
	alerts := []*storage.Alert{
		{
			Id:             "createId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "createId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range alerts {
		s.NoError(s.crud.Create(a))
	}

	for _, a := range alerts {
		s.Error(s.crud.Create(a))
	}

	for _, a := range alerts {
		full, err := s.crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedAlerts, err := s.crud.ReadBatch([]string{"createId1", "createId2"})
	s.NoError(err)
	s.ElementsMatch(alerts, retrievedAlerts)
}

func (s *MessageCrudTestSuite) TestUpdate() {
	alerts := []*storage.Alert{
		{
			Id:             "updateId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "updateId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	// Create the alerts.
	for _, a := range alerts {
		s.NoError(s.crud.Create(a))
	}

	updatedAlerts := []*storage.Alert{
		{
			Id:             "updateId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_MEDIUM_SEVERITY,
			},
		},
		{
			Id:             "updateId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_MEDIUM_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range updatedAlerts {
		s.NoError(s.crud.Update(a))
	}

	for _, a := range updatedAlerts {
		full, err := s.crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedAlerts, err := s.crud.ReadBatch([]string{"updateId1", "updateId2"})
	s.NoError(err)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
}

func (s *MessageCrudTestSuite) TestUpsert() {
	alerts := []*storage.Alert{
		{
			Id:             "upsertId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "upsertId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range alerts {
		s.NoError(s.crud.Upsert(a))
	}

	updatedAlerts := []*storage.Alert{
		{
			Id:             "upsertId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_MEDIUM_SEVERITY,
			},
		},
		{
			Id:             "upsertId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_MEDIUM_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range updatedAlerts {
		s.NoError(s.crud.Upsert(a))
	}

	for _, a := range updatedAlerts {
		full, err := s.crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedAlerts, err := s.crud.ReadBatch([]string{"upsertId1", "upsertId2"})
	s.NoError(err)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
}

func (s *MessageCrudTestSuite) TestDelete() {
	alerts := []*storage.Alert{
		{
			Id:             "deleteId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "deleteId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range alerts {
		s.NoError(s.crud.Upsert(a))
	}

	for _, a := range alerts {
		s.NoError(s.crud.Delete(a.GetId()))
	}

	retrievedAlerts, err := s.crud.ReadBatch([]string{"deleteId1", "deleteId2"})
	s.Error(err, "messages should not exist")
	s.Equal(0, len(retrievedAlerts), "all alerts should be deleted")
}

func (s *MessageCrudTestSuite) TestDeleteBatch() {
	alerts := []*storage.Alert{
		{
			Id:             "deleteBatchId1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "deleteBatchId2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range alerts {
		s.NoError(s.crud.Upsert(a))
	}

	s.NoError(s.crud.DeleteBatch([]string{"deleteBatchId1", "deleteBatchId2"}))

	retrievedAlerts, err := s.crud.ReadBatch([]string{"deleteId1", "deleteId2"})
	s.Error(err, "messages should not exist")
	s.Equal(0, len(retrievedAlerts), "all alerts should be deleted")
}
