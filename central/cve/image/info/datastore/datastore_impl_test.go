//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ImageCVEInfoDataStoreSuite struct {
	suite.Suite
	datastore DataStore
	testDB    *pgtest.TestPostgres
	ctx       context.Context
}

func TestImageCVEInfoDataStore(t *testing.T) {
	suite.Run(t, new(ImageCVEInfoDataStoreSuite))
}

func (s *ImageCVEInfoDataStoreSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ImageCVEInfoDataStoreSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE image_cve_infos CASCADE")
	s.T().Log("image_cve_infos", tag)
	s.NoError(err)
}

// TestUpdateTimestamps_NilOld tests that when old is nil, new is returned unchanged.
func (s *ImageCVEInfoDataStoreSuite) TestUpdateTimestamps_NilOld() {
	now := time.Now()
	newInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: timestamppb.New(now),
		FixAvailableTimestamp: timestamppb.New(now),
	}

	result := updateTimestamps(nil, newInfo)

	protoassert.Equal(s.T(), newInfo, result)
	s.Equal(now.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
	s.Equal(now.Unix(), result.GetFixAvailableTimestamp().AsTime().Unix())
}

// TestUpdateTimestamps_PreservesEarlierFirstSystemOccurrence tests that earlier timestamp is preserved.
func (s *ImageCVEInfoDataStoreSuite) TestUpdateTimestamps_PreservesEarlierFirstSystemOccurrence() {
	earlier := time.Now().Add(-24 * time.Hour)
	later := time.Now()

	oldInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: timestamppb.New(earlier),
	}
	newInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: timestamppb.New(later),
	}

	result := updateTimestamps(oldInfo, newInfo)

	// Should preserve the earlier timestamp from old
	s.Equal(earlier.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
}

// TestUpdateTimestamps_PreservesEarlierFixAvailableTimestamp tests that earlier fix timestamp is preserved.
func (s *ImageCVEInfoDataStoreSuite) TestUpdateTimestamps_PreservesEarlierFixAvailableTimestamp() {
	earlier := time.Now().Add(-24 * time.Hour)
	later := time.Now()

	oldInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FixAvailableTimestamp: timestamppb.New(earlier),
	}
	newInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FixAvailableTimestamp: timestamppb.New(later),
	}

	result := updateTimestamps(oldInfo, newInfo)

	// Should preserve the earlier timestamp from old
	s.Equal(earlier.Unix(), result.GetFixAvailableTimestamp().AsTime().Unix())
}

// TestUpdateTimestamps_UsesNewWhenOldIsZero tests that new timestamp is used when old is zero.
func (s *ImageCVEInfoDataStoreSuite) TestUpdateTimestamps_UsesNewWhenOldIsZero() {
	now := time.Now()

	oldInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: nil, // Zero timestamp
	}
	newInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: timestamppb.New(now),
	}

	result := updateTimestamps(oldInfo, newInfo)

	// Should use the new timestamp since old is zero/nil
	s.Equal(now.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
}

// TestUpdateTimestamps_UsesOldWhenNewIsZero tests that old timestamp is used when new is zero.
func (s *ImageCVEInfoDataStoreSuite) TestUpdateTimestamps_UsesOldWhenNewIsZero() {
	earlier := time.Now().Add(-24 * time.Hour)

	oldInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: timestamppb.New(earlier),
	}
	newInfo := &storage.ImageCVEInfo{
		Id:                    "test-id",
		FirstSystemOccurrence: nil, // Zero timestamp
	}

	result := updateTimestamps(oldInfo, newInfo)

	// Should use the old timestamp since new is zero/nil
	s.Equal(earlier.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
}

// TestUpsert_PreservesTimestamps tests that Upsert preserves earlier timestamps.
func (s *ImageCVEInfoDataStoreSuite) TestUpsert_PreservesTimestamps() {
	earlier := time.Now().Add(-24 * time.Hour)
	later := time.Now()

	// First, insert an info with an earlier timestamp
	firstInfo := &storage.ImageCVEInfo{
		Id:                    "test-cve#test-pkg#test-ds",
		FirstSystemOccurrence: timestamppb.New(earlier),
		FixAvailableTimestamp: timestamppb.New(earlier),
	}
	err := s.datastore.Upsert(s.ctx, firstInfo)
	s.NoError(err)

	// Now upsert with a later timestamp
	secondInfo := &storage.ImageCVEInfo{
		Id:                    "test-cve#test-pkg#test-ds",
		FirstSystemOccurrence: timestamppb.New(later),
		FixAvailableTimestamp: timestamppb.New(later),
	}
	err = s.datastore.Upsert(s.ctx, secondInfo)
	s.NoError(err)

	// Retrieve and verify the earlier timestamps are preserved
	result, found, err := s.datastore.Get(s.ctx, "test-cve#test-pkg#test-ds")
	s.NoError(err)
	s.True(found)
	s.Equal(earlier.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
	s.Equal(earlier.Unix(), result.GetFixAvailableTimestamp().AsTime().Unix())
}

// TestUpsertMany_PreservesTimestamps tests that UpsertMany preserves earlier timestamps.
func (s *ImageCVEInfoDataStoreSuite) TestUpsertMany_PreservesTimestamps() {
	earlier := time.Now().Add(-24 * time.Hour)
	earlier2 := time.Now().Add(-12 * time.Hour)
	later := time.Now()

	// First, insert infos with earlier timestamps. 
	// If there are multiple infos with same id, earlier timestamps should be preserved.
	firstInfos := []*storage.ImageCVEInfo{
		{
			Id:                    "test-cve-1#test-pkg#test-ds",
			FirstSystemOccurrence: timestamppb.New(earlier),
			FixAvailableTimestamp: timestamppb.New(earlier),
		},
		{
			Id:                    "test-cve-2#test-pkg#test-ds",
			FirstSystemOccurrence: timestamppb.New(earlier),
			FixAvailableTimestamp: timestamppb.New(earlier),
		},
		{
			Id:                    "test-cve-1#test-pkg#test-ds",
			FirstSystemOccurrence: nil,
			FixAvailableTimestamp: timestamppb.New(earlier2),
		}
	}
	err := s.datastore.UpsertMany(s.ctx, firstInfos)
	s.NoError(err)

	// Now upsert with later timestamps
	secondInfos := []*storage.ImageCVEInfo{
		{
			Id:                    "test-cve-1#test-pkg#test-ds",
			FirstSystemOccurrence: timestamppb.New(later),
			FixAvailableTimestamp: timestamppb.New(later),
		},
		{
			Id:                    "test-cve-2#test-pkg#test-ds",
			FirstSystemOccurrence: timestamppb.New(later),
			FixAvailableTimestamp: timestamppb.New(later),
		},
	}
	err = s.datastore.UpsertMany(s.ctx, secondInfos)
	s.NoError(err)

	// Retrieve and verify the earlier timestamps are preserved
	results, err := s.datastore.GetBatch(s.ctx, []string{"test-cve-1#test-pkg#test-ds", "test-cve-2#test-pkg#test-ds"})
	s.NoError(err)
	s.Len(results, 2)

	for _, result := range results {
		s.Equal(earlier.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
		s.Equal(earlier.Unix(), result.GetFixAvailableTimestamp().AsTime().Unix())
	}
}

// TestUpsert_NewInfo tests that a new info is inserted correctly.
func (s *ImageCVEInfoDataStoreSuite) TestUpsert_NewInfo() {
	now := time.Now()

	info := &storage.ImageCVEInfo{
		Id:                    "new-cve#new-pkg#new-ds",
		FirstSystemOccurrence: timestamppb.New(now),
		FixAvailableTimestamp: timestamppb.New(now),
	}
	err := s.datastore.Upsert(s.ctx, info)
	s.NoError(err)

	result, found, err := s.datastore.Get(s.ctx, "new-cve#new-pkg#new-ds")
	s.NoError(err)
	s.True(found)
	s.Equal(info.GetId(), result.GetId())
	s.Equal(now.Unix(), result.GetFirstSystemOccurrence().AsTime().Unix())
	s.Equal(now.Unix(), result.GetFixAvailableTimestamp().AsTime().Unix())
}

// Ensure protocompat is used (to satisfy import)
var _ = protocompat.TimestampNow
