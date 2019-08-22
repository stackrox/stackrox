package proto

import (
	"os"
	"testing"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCachedMessageCrud(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(cachedMessageCrudTestSuite))
}

type cachedMessageCrudTestSuite struct {
	suite.Suite

	db *bolt.DB

	crud  MessageCrud
	cache expiringcache.Cache
}

func fakeMetricFunc(a, b string) {}

func (s *cachedMessageCrudTestSuite) SetupSuite() {
	db := testutils.DBForSuite(s) //bolthelper.NewTemp(s.T().Name() + ".db")

	testBucket := []byte("testBucket")
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
	s.cache = expiringcache.NewExpiringCache(time.Hour)
	crud, err := NewCachedMessageCrud(db, []byte("testBucket"), keyFunc, allocFunc, s.cache, "testMetrifc", fakeMetricFunc)
	s.NoError(err)
	s.crud = crud
}

func (s *cachedMessageCrudTestSuite) TearDownSuite() {
	if s.db != nil {
		_ = s.db.Close()
		_ = os.Remove(s.db.Path())
	}
}

func (s *cachedMessageCrudTestSuite) TestCreate() {
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

	retrievedAlerts, missingIndices, err := s.crud.ReadBatch([]string{"createId1", "createId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(alerts, retrievedAlerts)
}

func (s *cachedMessageCrudTestSuite) TestUpdate() {
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

	retrievedAlerts, missingIndices, err := s.crud.ReadBatch([]string{"updateId1", "updateId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
}

func (s *cachedMessageCrudTestSuite) TestUpsert() {
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

	retrievedAlerts, missingIndices, err := s.crud.ReadBatch([]string{"upsertId1", "upsertId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
}

func (s *cachedMessageCrudTestSuite) TestDelete() {
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

	retrievedAlerts, missingIndices, err := s.crud.ReadBatch([]string{"deleteId1", "deleteId2"})
	s.NoError(err)
	s.ElementsMatch(missingIndices, []int{0, 1})
	s.Empty(retrievedAlerts)
}

func (s *cachedMessageCrudTestSuite) TestDeleteBatch() {
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

	ids := []string{"deleteBatchId1", "deleteBatchId2"}
	s.NoError(s.crud.DeleteBatch(ids))

	for _, id := range ids {
		alert, err := s.crud.Read(id)
		s.NoError(err)
		s.Nil(alert)
	}
}
