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
	store  NodeCVEStore
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

	s.store = NewNodeCVEStore()
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
	cache := s.store.(*nodeCVEStoreImpl).cache
	cachedCVEs, missingIDs := cache.GetMany([]string{cve1.GetId(), nonExistentID})

	// Should return cached CVE and missing ID
	s.Len(cachedCVEs, 1)
	s.Contains(cachedCVEs, cve1.GetId())
	s.Len(missingIDs, 1)
	s.Contains(missingIDs, nonExistentID)
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
