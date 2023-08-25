package proto

import (
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/storecache"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestCachedMessageCrud(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(cachedMessageCrudTestSuite))
}

// cacheTracker will be used by the tests to assert on cache hits and cache misses
type cacheTracker struct {
	cacheHit  int
	cacheMiss int
}

func (m *cacheTracker) fakeMetricFunc(a, _ string) {
	if a == "hit" {
		m.cacheHit++
	} else {
		m.cacheMiss++
	}
}

type cachedMessageCrudTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (s *cachedMessageCrudTestSuite) SetupSuite() {
	db := testutils.DBForSuite(s) // bolthelper.NewTemp(s.T().Name() + ".db")

	testBucket := []byte("testBucket")
	bolthelper.RegisterBucketOrPanic(db, testBucket)

	s.db = db
}

func (s *cachedMessageCrudTestSuite) TearDownSuite() {
	if s.db != nil {
		_ = s.db.Close()
		_ = os.Remove(s.db.Path())
	}
}

func (s *cachedMessageCrudTestSuite) makeCachedMessageCrud() (MessageCrud, *cacheTracker) {
	// Function that provides the key for a given instance.
	keyFunc := func(msg proto.Message) []byte {
		return []byte(msg.(*storage.Alert).GetId())
	}
	// Function that provide a new empty instance when wanted.
	allocFunc := func() proto.Message {
		return &storage.Alert{}
	}
	cache := storecache.NewMapBackedCache()
	wrappedCrud, err := NewMessageCrud(s.db, []byte("testBucket"), keyFunc, allocFunc)
	s.NoError(err)
	tracker := &cacheTracker{}
	return NewCachedMessageCrud(wrappedCrud, cache, "testMetric", tracker.fakeMetricFunc), tracker
}

func (s *cachedMessageCrudTestSuite) TestCreate() {
	crud, tracker := s.makeCachedMessageCrud()
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
		s.NoError(crud.Create(a))
	}

	for _, a := range alerts {
		s.Error(crud.Create(a))
	}

	for _, a := range alerts {
		// Creates don't put things in the cache.  This get should be a cache miss.
		expectedMisses := tracker.cacheMiss + 1
		full, err := crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
		s.Equal(expectedMisses, tracker.cacheMiss)
	}

	// These results were previously gotten.  These gets should be cache hits.
	expectedHits := tracker.cacheHit + len(alerts)
	retrievedAlerts, missingIndices, err := crud.ReadBatch([]string{"createId1", "createId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(alerts, retrievedAlerts)
	s.Equal(expectedHits, tracker.cacheHit)
}

func (s *cachedMessageCrudTestSuite) TestUpdate() {
	crud, tracker := s.makeCachedMessageCrud()
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
		s.NoError(crud.Create(a))
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

	var expectedWriteVersion uint64
	for _, a := range updatedAlerts {
		expectedWriteVersion++
		writeVersion, _, err := crud.Update(a)
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
	}

	for _, a := range updatedAlerts {
		expectedHits := tracker.cacheHit + 1
		full, err := crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
		s.Equal(expectedHits, tracker.cacheHit)
	}

	expectedHits := tracker.cacheHit + len(updatedAlerts)
	retrievedAlerts, missingIndices, err := crud.ReadBatch([]string{"updateId1", "updateId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
	s.Equal(expectedHits, tracker.cacheHit)
}

func (s *cachedMessageCrudTestSuite) TestUpsert() {
	crud, tracker := s.makeCachedMessageCrud()
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

	var expectedWriteVersion uint64
	for _, a := range alerts {
		expectedWriteVersion++
		writeVersion, _, err := crud.Upsert(a)
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
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
		expectedWriteVersion++
		writeVersion, _, err := crud.Upsert(a)
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
	}

	for _, a := range updatedAlerts {
		expectedCacheHits := tracker.cacheHit + 1
		full, err := crud.Read(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
		s.Equal(expectedCacheHits, tracker.cacheHit)
	}

	expectedCacheHits := tracker.cacheHit + len(alerts)
	retrievedAlerts, missingIndices, err := crud.ReadBatch([]string{"upsertId1", "upsertId2"})
	s.NoError(err)
	s.Empty(missingIndices)
	s.ElementsMatch(updatedAlerts, retrievedAlerts)
	s.Equal(expectedCacheHits, tracker.cacheHit)
}

func (s *cachedMessageCrudTestSuite) TestDelete() {
	crud, tracker := s.makeCachedMessageCrud()
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

	var expectedWriteVersion uint64
	for _, a := range alerts {
		expectedWriteVersion++
		writeVersion, _, err := crud.Upsert(a)
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
	}

	for _, a := range alerts {
		expectedWriteVersion++
		writeVersion, _, err := crud.Delete(a.GetId())
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
	}

	expectedCacheMisses := tracker.cacheMiss + len(alerts)
	retrievedAlerts, missingIndices, err := crud.ReadBatch([]string{"deleteId1", "deleteId2"})
	s.NoError(err)
	s.ElementsMatch(missingIndices, []int{0, 1})
	s.Empty(retrievedAlerts)
	s.Equal(expectedCacheMisses, tracker.cacheMiss)
}

func (s *cachedMessageCrudTestSuite) TestDeleteBatch() {
	crud, tracker := s.makeCachedMessageCrud()
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

	var expectedWriteVersion uint64
	for _, a := range alerts {
		expectedWriteVersion++
		writeVersion, _, err := crud.Upsert(a)
		s.NoError(err)
		s.Equal(expectedWriteVersion, writeVersion)
	}

	ids := []string{"deleteBatchId1", "deleteBatchId2"}
	expectedWriteVersion++
	writeVersion, _, err := crud.DeleteBatch(ids)
	s.NoError(err)
	s.Equal(expectedWriteVersion, writeVersion)

	for _, id := range ids {
		expectedCacheMisses := tracker.cacheMiss + 1
		alert, err := crud.Read(id)
		s.NoError(err)
		s.Nil(alert)
		s.Equal(expectedCacheMisses, tracker.cacheMiss)
	}
}
