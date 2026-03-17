//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	scanStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	vmV2Store "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	vmV2Postgres "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVMScanV2DataStore(t *testing.T) {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(vmScanV2DataStoreTestSuite))
}

type vmScanV2DataStoreTestSuite struct {
	suite.Suite

	db        *pgtest.TestPostgres
	datastore DataStore
	vmStore   vmV2Store.Store

	sacCtx   context.Context
	noSacCtx context.Context
}

func (s *vmScanV2DataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	store := scanStore.New(s.db)
	s.datastore = New(store)
	s.vmStore = vmV2Postgres.New(s.db, concurrency.NewKeyFence())

	s.sacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *vmScanV2DataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *vmScanV2DataStoreTestSuite) createTestVM(clusterID, namespace string, index int) *storage.VirtualMachineV2 {
	vmID := uuid.NewV5FromNonUUIDs(clusterID, fmt.Sprintf("%s-%d", namespace, index)).String()
	return &storage.VirtualMachineV2{
		Id:          vmID,
		Name:        fmt.Sprintf("vm-%d", index),
		Namespace:   namespace,
		ClusterId:   clusterID,
		ClusterName: fmt.Sprintf("cluster-%s", clusterID),
	}
}

func (s *vmScanV2DataStoreTestSuite) createTestScan(vmID string, index int) *storage.VirtualMachineScanV2 {
	scanID := uuid.NewV5FromNonUUIDs(vmID, fmt.Sprintf("scan-%d", index)).String()
	return &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanOs:   "rhel:9",
		ScanTime: timestamppb.Now(),
		TopCvss:  7.5,
	}
}

func (s *vmScanV2DataStoreTestSuite) insertVMAndScan(clusterID, namespace string, vmIndex, scanIndex int) *storage.VirtualMachineScanV2 {
	ctx := sac.WithAllAccess(context.Background())
	vm := s.createTestVM(clusterID, namespace, vmIndex)
	s.Require().NoError(s.vmStore.UpsertVM(ctx, vm))
	scan := s.createTestScan(vm.GetId(), scanIndex)
	scanPgStore := scanStore.New(s.db)
	s.Require().NoError(scanPgStore.Upsert(ctx, scan))
	return scan
}

// TestGet verifies basic get functionality.
func (s *vmScanV2DataStoreTestSuite) TestGet() {
	scan := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	retrievedScan, found, err := s.datastore.Get(s.sacCtx, scan.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(scan.GetId(), retrievedScan.GetId())
	s.Equal(scan.GetScanOs(), retrievedScan.GetScanOs())

	// Non-existent scan
	_, found, err = s.datastore.Get(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(found)
}

// TestExists verifies existence checks.
func (s *vmScanV2DataStoreTestSuite) TestExists() {
	scan := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	exists, err := s.datastore.Exists(s.sacCtx, scan.GetId())
	s.NoError(err)
	s.True(exists)

	exists, err = s.datastore.Exists(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(exists)
}

// TestCount verifies count functionality.
func (s *vmScanV2DataStoreTestSuite) TestCount() {
	count, err := s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	count, err = s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(2, count)
}

// TestGetBatch verifies batch retrieval.
func (s *vmScanV2DataStoreTestSuite) TestGetBatch() {
	scan1 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	scan2 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	scans, err := s.datastore.GetBatch(s.sacCtx, []string{scan1.GetId(), scan2.GetId()})
	s.NoError(err)
	s.Len(scans, 2)

	fetchedIDs := make([]string, len(scans))
	for i, sc := range scans {
		fetchedIDs[i] = sc.GetId()
	}
	s.ElementsMatch([]string{scan1.GetId(), scan2.GetId()}, fetchedIDs)
}

// TestSearch verifies search functionality.
func (s *vmScanV2DataStoreTestSuite) TestSearch() {
	scan1 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	scan2 := s.insertVMAndScan(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	results, err := s.datastore.Search(s.sacCtx, nil)
	s.NoError(err)
	s.Len(results, 2)

	fetchedIDs := make([]string, len(results))
	for i, r := range results {
		fetchedIDs[i] = r.ID
	}
	s.ElementsMatch([]string{scan1.GetId(), scan2.GetId()}, fetchedIDs)
}

// TestSearchRawVMScans verifies raw search.
func (s *vmScanV2DataStoreTestSuite) TestSearchRawVMScans() {
	scan1 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertVMAndScan(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	scans, err := s.datastore.SearchRawVMScans(s.sacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(scans, 2)

	// Verify scan data is populated
	for _, sc := range scans {
		s.NotEmpty(sc.GetId())
		s.NotEmpty(sc.GetScanOs())
	}

	// Search by VM scan OS
	q := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineScanOS, scan1.GetScanOs()).ProtoQuery()
	scans, err = s.datastore.SearchRawVMScans(s.sacCtx, q)
	s.NoError(err)
	s.Len(scans, 2)
}

// TestSACDeniedAccess verifies that SAC-denied contexts cannot access scans.
func (s *vmScanV2DataStoreTestSuite) TestSACDeniedAccess() {
	scan := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	// Get should silently return not found
	_, found, err := s.datastore.Get(s.noSacCtx, scan.GetId())
	s.NoError(err)
	s.False(found)

	// Exists should return false
	exists, err := s.datastore.Exists(s.noSacCtx, scan.GetId())
	s.NoError(err)
	s.False(exists)

	// Count should return 0
	count, err := s.datastore.Count(s.noSacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	// Search should return empty
	results, err := s.datastore.Search(s.noSacCtx, nil)
	s.NoError(err)
	s.Empty(results)

	// GetBatch should return empty
	scans, err := s.datastore.GetBatch(s.noSacCtx, []string{scan.GetId()})
	s.NoError(err)
	s.Empty(scans)

	// SearchRaw should return empty
	rawScans, err := s.datastore.SearchRawVMScans(s.noSacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(rawScans)
}

// TestSACNamespaceScopedAccess verifies namespace-scoped SAC filtering.
func (s *vmScanV2DataStoreTestSuite) TestSACNamespaceScopedAccess() {
	// Insert scans across different clusters and namespaces
	scan1 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	scan2 := s.insertVMAndScan(testconsts.Cluster1, testconsts.NamespaceB, 1, 1)
	scan3 := s.insertVMAndScan(testconsts.Cluster2, testconsts.NamespaceA, 1, 1)
	scan4 := s.insertVMAndScan(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	contexts := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.VirtualMachine)

	for name, tc := range map[string]struct {
		scopeKey    string
		expectedIDs []string
	}{
		"Unrestricted read access returns all scans": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectedIDs: []string{scan1.GetId(), scan2.GetId(), scan3.GetId(), scan4.GetId()},
		},
		"Unrestricted read-write access returns all scans": {
			scopeKey:    testutils.UnrestrictedReadWriteCtx,
			expectedIDs: []string{scan1.GetId(), scan2.GetId(), scan3.GetId(), scan4.GetId()},
		},
		"Cluster1 read-write access returns only cluster1 scans": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectedIDs: []string{scan1.GetId(), scan2.GetId()},
		},
		"Cluster2 read-write access returns only cluster2 scans": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectedIDs: []string{scan3.GetId(), scan4.GetId()},
		},
		"Cluster3 read-write access returns no scans": {
			scopeKey:    testutils.Cluster3ReadWriteCtx,
			expectedIDs: nil,
		},
		"Cluster1 NamespaceA access returns only cluster1 namespaceA scans": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectedIDs: []string{scan1.GetId()},
		},
		"Cluster1 NamespaceB access returns only cluster1 namespaceB scans": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectedIDs: []string{scan2.GetId()},
		},
		"Cluster1 NamespacesAB access returns cluster1 namespaces A and B scans": {
			scopeKey:    testutils.Cluster1NamespacesABReadWriteCtx,
			expectedIDs: []string{scan1.GetId(), scan2.GetId()},
		},
		"Cluster2 NamespaceB access returns only cluster2 namespaceB scans": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectedIDs: []string{scan4.GetId()},
		},
		"Mixed cluster and namespace access returns scans from allowed scopes": {
			scopeKey:    testutils.MixedClusterAndNamespaceReadCtx,
			expectedIDs: []string{scan1.GetId(), scan3.GetId(), scan4.GetId()},
		},
	} {
		s.Run(name, func() {
			ctx := contexts[tc.scopeKey]

			// Test Count
			count, err := s.datastore.Count(ctx, nil)
			s.NoError(err)
			s.Equal(len(tc.expectedIDs), count)

			// Test Search
			results, err := s.datastore.Search(ctx, nil)
			s.NoError(err)
			fetchedIDs := make([]string, len(results))
			for i, r := range results {
				fetchedIDs[i] = r.ID
			}
			s.ElementsMatch(tc.expectedIDs, fetchedIDs)

			// Test Exists for a scan in cluster1 namespaceA
			exists, err := s.datastore.Exists(ctx, scan1.GetId())
			s.NoError(err)
			expectedExists := false
			for _, id := range tc.expectedIDs {
				if id == scan1.GetId() {
					expectedExists = true
					break
				}
			}
			s.Equal(expectedExists, exists)

			// Test Get for a scan in cluster1 namespaceA
			_, found, err := s.datastore.Get(ctx, scan1.GetId())
			s.NoError(err)
			s.Equal(expectedExists, found)

			// Test GetBatch
			allIDs := []string{scan1.GetId(), scan2.GetId(), scan3.GetId(), scan4.GetId()}
			scans, err := s.datastore.GetBatch(ctx, allIDs)
			s.NoError(err)
			batchIDs := make([]string, len(scans))
			for i, sc := range scans {
				batchIDs[i] = sc.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, batchIDs)

			// Test SearchRawVMScans
			rawScans, err := s.datastore.SearchRawVMScans(ctx, search.EmptyQuery())
			s.NoError(err)
			rawIDs := make([]string, len(rawScans))
			for i, sc := range rawScans {
				rawIDs[i] = sc.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, rawIDs)
		})
	}
}

// TestSingletonGatedByFeatureFlag verifies the singleton is nil when the feature flag is disabled.
func (s *vmScanV2DataStoreTestSuite) TestSingletonGatedByFeatureFlag() {
	s.T().Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "false")
	s.Nil(Singleton())
}
