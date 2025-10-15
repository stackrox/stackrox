//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type NodeCVEStoreSuite struct {
	suite.Suite
	ctx    context.Context
	pool   postgres.DB
	gormDB *gorm.DB
	store  *nodeCVEStore
}

func TestNodeCVEStore(t *testing.T) {
	suite.Run(t, new(NodeCVEStoreSuite))
}

func (s *NodeCVEStoreSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())
	source := pgtest.GetConnectionString(s.T())

	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(s.ctx, config)
	s.NoError(err)
	Destroy(s.ctx, s.pool)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)

	// Create required tables
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableClustersStmt)
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableNodesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableNodeComponentsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableNodeCvesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableNodeComponentEdgesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, pkgSchema.CreateTableNodeComponentsCvesEdgesStmt)

	s.store = newNodeCVEStore()
}

func (s *NodeCVEStoreSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.gormDB != nil {
		pgtest.CloseGormDB(s.T(), s.gormDB)
	}
}

func (s *NodeCVEStoreSuite) TestCacheGetCVEs() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVEs
	cve1 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve1.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	cve2 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve2.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	// Insert CVEs using CopyFromNodeCves (which should populate cache)
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve1, cve2)
	s.NoError(err)

	// First call should hit database and populate cache
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)
	s.Len(cves, 2)
	s.Contains(cves, cve1.GetId())
	s.Contains(cves, cve2.GetId())

	// Second call should hit cache (we can't directly verify this, but it should work the same)
	cves2, err := s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)
	s.Len(cves2, 2)
	s.Equal(cves[cve1.GetId()].GetId(), cves2[cve1.GetId()].GetId())
	s.Equal(cves[cve2.GetId()].GetId(), cves2[cve2.GetId()].GetId())
}

func (s *NodeCVEStoreSuite) TestCachePartialHit() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVEs
	cve1 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve1.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	cve2 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve2.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	cve3 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve3, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve3.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	// Insert only cve1 and cve2
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve1, cve2)
	s.NoError(err)

	// First call to populate cache for cve1 and cve2
	_, err = s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)

	// Insert cve3
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve3)
	s.NoError(err)

	// Request all three - should get cve1,cve2 from cache and cve3 from DB
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId(), cve3.GetId()})
	s.NoError(err)
	s.Len(cves, 3)
	s.Contains(cves, cve1.GetId())
	s.Contains(cves, cve2.GetId())
	s.Contains(cves, cve3.GetId())
}

func (s *NodeCVEStoreSuite) TestCacheInvalidationOnDelete() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVE
	cve := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	// Insert CVE (it will be orphaned since we don't create component edges)
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve)
	s.NoError(err)

	// Verify CVE exists in cache and database
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)

	// Call RemoveOrphanedNodeCVEs which should remove the orphaned CVE and invalidate cache
	err = s.store.RemoveOrphanedNodeCVEs(s.ctx, tx)
	s.NoError(err)

	// Verify CVE is no longer returned (should not be in cache or DB)
	cves, err = s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 0)
}

func (s *NodeCVEStoreSuite) TestCacheUpdateOnCopy() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVE
	cve := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})
	cve.Snoozed = false

	// Insert CVE
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve)
	s.NoError(err)

	// Verify initial state
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)
	s.False(cves[cve.GetId()].GetSnoozed())

	// Update CVE
	cve.Snoozed = true
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve)
	s.NoError(err)

	// Verify cache has updated value
	cves, err = s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)
	s.True(cves[cve.GetId()].GetSnoozed())
}

func (s *NodeCVEStoreSuite) TestMarkOrphanedNodeCVEs() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVEs
	cve1 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve1.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})
	cve1.Orphaned = false
	cve1.OrphanedTime = nil

	cve2 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve2.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})
	cve2.Orphaned = false
	cve2.OrphanedTime = nil

	// Insert CVEs (they will be orphaned since we don't create component edges)
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve1, cve2)
	s.NoError(err)

	// Verify CVEs exist and are not marked as orphaned
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)
	s.Len(cves, 2)
	s.False(cves[cve1.GetId()].GetOrphaned())
	s.False(cves[cve2.GetId()].GetOrphaned())
	s.Nil(cves[cve1.GetId()].GetOrphanedTime())
	s.Nil(cves[cve2.GetId()].GetOrphanedTime())

	// Call MarkOrphanedNodeCVEs which should mark the orphaned CVEs and update cache
	err = s.store.MarkOrphanedNodeCVEs(s.ctx, tx)
	s.NoError(err)

	// Verify CVEs are now marked as orphaned and cache is updated
	cves, err = s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)
	s.Len(cves, 2)
	s.True(cves[cve1.GetId()].GetOrphaned())
	s.True(cves[cve2.GetId()].GetOrphaned())
	s.NotNil(cves[cve1.GetId()].GetOrphanedTime())
	s.NotNil(cves[cve2.GetId()].GetOrphanedTime())
}

func (s *NodeCVEStoreSuite) TestCacheMissingIDs() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVE
	cve1 := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve1.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	// Insert only cve1
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve1)
	s.NoError(err)

	// Request both existing and non-existing CVE
	nonExistentID := "CVE-DOES-NOT-EXIST"
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve1.GetId(), nonExistentID})
	s.NoError(err)

	// Should only return the existing CVE
	s.Len(cves, 1)
	s.Contains(cves, cve1.GetId())
	s.NotContains(cves, nonExistentID)

	// Test cache directly to verify missing IDs are returned
	cache := s.store.cache
	cachedCVEs, missingIDs := cache.GetMany([]string{cve1.GetId(), nonExistentID})

	// Should return cached CVE and missing ID
	s.Len(cachedCVEs, 1)
	s.Contains(cachedCVEs, cve1.GetId())
	s.Len(missingIDs, 1)
	s.Contains(missingIDs, nonExistentID)
}

func (s *NodeCVEStoreSuite) TestGetCVEsEmptyInput() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Test with empty slice
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{})
	s.NoError(err)
	s.Empty(cves)

	// Test with nil slice
	cves, err = s.store.GetCVEs(s.ctx, tx, nil)
	s.NoError(err)
	s.Empty(cves)
}

func (s *NodeCVEStoreSuite) TestGetCVEsNothingFoundInDB() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Request CVEs that don't exist in cache or DB
	nonExistentIDs := []string{"CVE-1111-1111", "CVE-2222-2222"}
	cves, err := s.store.GetCVEs(s.ctx, tx, nonExistentIDs)
	s.NoError(err)

	// Should return empty map since nothing exists
	s.Empty(cves)

	// Verify the cache path was also taken for missing IDs
	cache := s.store.cache
	cachedCVEs, missingIDs := cache.GetMany(nonExistentIDs)
	s.Empty(cachedCVEs)
	s.Len(missingIDs, 2)
}

func (s *NodeCVEStoreSuite) TestMarkOrphanedNodeCVEsNothingToMark() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Call MarkOrphanedNodeCVEs when there are no CVEs to mark as orphaned
	// (i.e., when orphanedNodeCVEs slice is empty)
	err = s.store.MarkOrphanedNodeCVEs(s.ctx, tx)
	s.NoError(err)

	// Should complete successfully even with no CVEs to mark
}

func (s *NodeCVEStoreSuite) TestRemoveOrphanedNodeCVEsNothingToRemove() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Call RemoveOrphanedNodeCVEs when there are no orphaned CVEs to remove
	err = s.store.RemoveOrphanedNodeCVEs(s.ctx, tx)
	s.NoError(err)

	// Should complete successfully even with no CVEs to remove
}

func (s *NodeCVEStoreSuite) TestCopyFromNodeCvesEmptyInput() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Test with empty slice
	err = s.store.CopyFromNodeCves(s.ctx, tx)
	s.NoError(err)

	// Test with nil slice
	err = s.store.CopyFromNodeCves(s.ctx, tx, nil...)
	s.NoError(err)
}

func (s *NodeCVEStoreSuite) TestMarkOrphanedWithDuplicateIds() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create a test CVE that will be considered orphaned
	cve := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})
	cve.Orphaned = false
	cve.OrphanedTime = nil

	// Insert the CVE (it will be orphaned since we don't create component edges)
	err = s.store.CopyFromNodeCves(s.ctx, tx, cve)
	s.NoError(err)

	// Verify initial state
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)
	s.False(cves[cve.GetId()].GetOrphaned())

	// Call MarkOrphanedNodeCVEs - this should find the orphaned CVE
	// The duplicate ID check (ids.Add) ensures each CVE is only processed once
	err = s.store.MarkOrphanedNodeCVEs(s.ctx, tx)
	s.NoError(err)

	// Verify CVE is now marked as orphaned
	cves, err = s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)
	s.True(cves[cve.GetId()].GetOrphaned())
	s.NotNil(cves[cve.GetId()].GetOrphanedTime())
}

func (s *NodeCVEStoreSuite) TestCopyFromNodeCvesZeroLengthBatch() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// This test ensures we handle the case where somehow we get an empty batch
	// which should result in len(deletes) == 0 and skip the cache deletion branch

	// Test with an empty batch - this should not fail
	err = s.store.CopyFromNodeCves(s.ctx, tx)
	s.NoError(err)
}

func (s *NodeCVEStoreSuite) TestCacheEmptyOperations() {
	// Test cache operations with empty data to ensure all branches are covered
	cache := newNodeCVECache()

	// Test GetMany with empty slice
	result, missing := cache.GetMany([]string{})
	s.Empty(result)
	s.Empty(missing)

	// Test SetMany with empty map
	cache.SetMany(map[string]*storage.NodeCVE{})

	// Test DeleteMany with empty slice
	cache.DeleteMany([]string{})

	// Verify cache is still empty
	result, missing = cache.GetMany([]string{"test"})
	s.Empty(result)
	s.Len(missing, 1)
	s.Equal("test", missing[0])
}

func (s *NodeCVEStoreSuite) TestGetCVEsFromDatabaseWithCacheMiss() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create test CVE
	cve := &storage.NodeCVE{}
	s.NoError(testutils.FullInit(cve, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	cve.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})

	// Insert CVE directly into database (bypassing cache) to simulate cache miss
	serialized, err := cve.MarshalVT()
	s.Require().NoError(err)

	_, err = tx.Exec(s.ctx, `
		INSERT INTO `+nodeCVEsTable+` (
			id, cvebaseinfo_cve, cvebaseinfo_publishedon, cvebaseinfo_createdat,
			operatingsystem, cvss, severity, impactscore, snoozed, snoozeexpiry,
			orphaned, orphanedtime, serialized
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		cve.GetId(),
		cve.GetCveBaseInfo().GetCve(),
		protocompat.NilOrTime(cve.GetCveBaseInfo().GetPublishedOn()),
		protocompat.NilOrTime(cve.GetCveBaseInfo().GetCreatedAt()),
		cve.GetOperatingSystem(),
		cve.GetCvss(),
		cve.GetSeverity(),
		cve.GetImpactScore(),
		cve.GetSnoozed(),
		protocompat.NilOrTime(cve.GetSnoozeExpiry()),
		cve.GetOrphaned(),
		protocompat.NilOrTime(cve.GetOrphanedTime()),
		serialized,
	)
	s.Require().NoError(err)

	// Verify cache is empty for this CVE (since we bypassed it)
	cache := s.store.cache
	cachedCVEs, missingIDs := cache.GetMany([]string{cve.GetId()})
	s.Empty(cachedCVEs)
	s.Len(missingIDs, 1)
	s.Equal(cve.GetId(), missingIDs[0])

	// Query through store - should fetch from database and populate cache
	cves, err := s.store.GetCVEs(s.ctx, tx, []string{cve.GetId()})
	s.NoError(err)
	s.Len(cves, 1)
	s.Contains(cves, cve.GetId())
	s.Equal(cve.GetId(), cves[cve.GetId()].GetId())

	// Verify cache now contains the CVE
	cachedCVEs, missingIDs = cache.GetMany([]string{cve.GetId()})
	s.Len(cachedCVEs, 1)
	s.Contains(cachedCVEs, cve.GetId())
	s.Empty(missingIDs)
}

func (s *NodeCVEStoreSuite) TestBatchingBehavior() {
	conn, err := s.pool.Acquire(s.ctx)
	s.Require().NoError(err)
	defer conn.Release()

	tx, err := conn.Begin(s.ctx)
	s.Require().NoError(err)
	defer func() {
		s.NoError(tx.Commit(s.ctx))
	}()

	// Create a large number of CVEs to test batching (more than batchSize = 500)
	const numCVEs = 1200
	cves := make([]*storage.NodeCVE, numCVEs)
	expectedIDs := make([]string, numCVEs)

	for i := 0; i < numCVEs; i++ {
		cve := &storage.NodeCVE{}
		s.NoError(testutils.FullInit(cve, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		cve.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&time.Time{})
		cves[i] = cve
		expectedIDs[i] = cve.GetId()
	}

	// Insert all CVEs (should be processed in multiple batches)
	err = s.store.CopyFromNodeCves(s.ctx, tx, cves...)
	s.NoError(err)

	// Verify all CVEs were inserted and cached
	retrievedCVEs, err := s.store.GetCVEs(s.ctx, tx, expectedIDs)
	s.NoError(err)
	s.Len(retrievedCVEs, numCVEs)

	// Verify all expected IDs are present
	for _, expectedID := range expectedIDs {
		s.Contains(retrievedCVEs, expectedID)
	}
}
